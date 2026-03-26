package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"kairos/pkg/config"
	"kairos/pkg/events"
	"kairos/pkg/moderation"
	"kairos/pkg/rabbitmq"
	"kairos/pkg/redis"
	"kairos/pkg/registry"

	redislib "github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Redis key 与 Feed 热榜约定一致
const (
	RedisKeyHotLikes = "feed:hot:likes" // video_id -> likes_count
	RedisKeyHot      = "feed:hot"       // video_id -> popularity
)

func commentListKey(videoID uint) string {
	return fmt.Sprintf("comment:list:%d", videoID)
}

// ProcessedEvent MQ 幂等表：用于保证 at-least-once 投递不重复执行副作用
type ProcessedEvent struct {
	EventID   string    `gorm:"primaryKey;size:64;column:event_id"`
	EventType string    `gorm:"size:32;index;column:event_type"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (ProcessedEvent) TableName() string { return "processed_events" }

func main() {
	config.LoadEnvFromSearchPaths(true)
	cfg := config.Load()

	// etcd 注册（可选）：worker 仅注册用于观测/运维（无对外 HTTP）
	var reg *registry.EtcdRegistry
	var regCancel context.CancelFunc
	if len(cfg.Etcd.Endpoints) > 0 {
		r, err := registry.NewEtcd(registry.Config{Endpoints: cfg.Etcd.Endpoints, Prefix: cfg.Etcd.Prefix, TTL: cfg.Etcd.TTL})
		if err != nil {
			log.Printf("worker: etcd disabled: %v", err)
		} else {
			reg = r
			instanceID := fmt.Sprintf("worker-%d", os.Getpid())
			regCancel, _ = reg.Register(context.Background(), "worker", registry.ServiceHTTP, instanceID, "worker")
		}
	}
	defer func() {
		if regCancel != nil {
			regCancel()
		}
		_ = reg.Close()
	}()

	if cfg.RabbitMQ.URL == "" {
		log.Fatalf("RABBITMQ_URL is required for worker")
	}

	mq, err := rabbitmq.New(cfg.RabbitMQ.URL)
	if err != nil {
		log.Fatalf("rabbitmq: %v", err)
	}
	defer mq.Close()

	rdb := redis.New(redis.Config{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := rdb.Ping(context.Background()); err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer rdb.Close()

	db, err := gorm.Open(mysql.Open(dsn(cfg.Database)), &gorm.Config{})
	if err != nil {
		log.Fatalf("mysql: %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	// 幂等表：自动建表（与 video/account 共用 DB）
	if err := db.AutoMigrate(&ProcessedEvent{}); err != nil {
		log.Printf("worker: auto migrate processed_events failed: %v", err)
	}

	// Ark 大模型审核（可选：没配置 key 则禁用）
	var arkCli *moderation.ArkClient
	if cfg.Ark.APIKey != "" {
		if c, err := moderation.NewArkClient(moderation.ArkConfig{APIKey: cfg.Ark.APIKey, BaseURL: cfg.Ark.BaseURL, Model: cfg.Ark.Model}); err != nil {
			log.Printf("worker: ark disabled: %v", err)
		} else {
			arkCli = c
			log.Printf("worker: ark moderation enabled (model=%s)", cfg.Ark.Model)
		}
	}

	// 启动时从 MySQL 全量同步点赞数、热度到 Redis
	if err := syncLikesAndPopularityFromMySQL(context.Background(), db, cfg, rdb); err != nil {
		log.Printf("worker: sync from mysql failed (non-fatal): %v", err)
	} else {
		log.Printf("worker: synced likes/popularity from MySQL to Redis")
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	// 定时全量同步，纠正 MQ 丢消息等导致的 Redis 与 MySQL 不一致
	syncIntervalMin := cfg.Worker.SyncIntervalMin
	if syncIntervalMin > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ticker := time.NewTicker(time.Duration(syncIntervalMin) * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := syncLikesAndPopularityFromMySQL(ctx, db, cfg, rdb); err != nil {
						log.Printf("worker: periodic sync failed: %v", err)
					} else {
						log.Printf("worker: periodic sync done")
					}
				}
			}
		}()
	}
	defer cancel()

	// Like 消费者：更新 Redis feed:hot:likes
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("worker: consuming %s", events.QueueLike)
		_ = mq.Consume(events.QueueLike, func(body []byte) error {
			var e events.LikeEvent
			if err := json.Unmarshal(body, &e); err != nil {
				return err
			}
			if e.EventID != "" {
				ok, err := recordProcessedEvent(ctx, db, e.EventID, "like")
				if err != nil {
					return err
				}
				if !ok {
					return nil
				}
			}
			member := formatVideoID(e.VideoID)
			return rdb.ZIncrBy(ctx, RedisKeyHotLikes, float64(e.Delta), member)
		})
	}()

	// Comment 消费者：仅 ack（可选扩展）
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("worker: consuming %s", events.QueueComment)
		_ = mq.Consume(events.QueueComment, func(body []byte) error {
			var e events.CommentEvent
			if err := json.Unmarshal(body, &e); err != nil {
				return err
			}
			if e.EventID != "" {
				ok, err := recordProcessedEvent(ctx, db, e.EventID, "comment")
				if err != nil {
					return err
				}
				if !ok {
					return nil
				}
			}
			return nil
		})
	}()

	// Social 消费者：仅 ack
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("worker: consuming %s", events.QueueSocial)
		_ = mq.Consume(events.QueueSocial, func(body []byte) error {
			var e events.SocialEvent
			if err := json.Unmarshal(body, &e); err != nil {
				return err
			}
			if e.EventID != "" {
				ok, err := recordProcessedEvent(ctx, db, e.EventID, "social")
				if err != nil {
					return err
				}
				if !ok {
					return nil
				}
			}
			return nil
		})
	}()

	// CommentAudit 消费者：调用大模型审核评论并回写 MySQL
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("worker: consuming %s", events.QueueCommentAudit)
		_ = mq.Consume(events.QueueCommentAudit, func(body []byte) error {
			var e events.CommentAuditEvent
			if err := json.Unmarshal(body, &e); err != nil {
				return err
			}
			if e.CommentID == 0 || e.VideoID == 0 {
				return nil
			}
			if e.EventID != "" {
				ok, err := recordProcessedEvent(ctx, db, e.EventID, "comment_audit")
				if err != nil {
					return err
				}
				if !ok {
					return nil
				}
			}
			if arkCli == nil {
				changed, err := approveComment(ctx, db, e.CommentID, e.VideoID, "ark", "", "ark disabled (auto-approve)")
				if err == nil && changed {
					_ = rdb.Del(ctx, commentListKey(e.VideoID))
				}
				return err
			}
			res, err := arkCli.AuditComment(ctx, e.Content)
			if err != nil {
				changed, e2 := markSuspect(ctx, db, e.CommentID, "ark", "其他", "ark error: "+err.Error())
				if e2 == nil && changed {
					_ = rdb.Del(ctx, commentListKey(e.VideoID))
				}
				return e2
			}
			cats := joinCats(res.Categories)
			switch res.Decision {
			case moderation.DecisionAllow:
				changed, e2 := approveComment(ctx, db, e.CommentID, e.VideoID, "ark", cats, res.Reason)
				if e2 == nil && changed {
					_ = rdb.Del(ctx, commentListKey(e.VideoID))
				}
				return e2
			case moderation.DecisionBlock:
				changed, e2 := blockComment(ctx, db, e.CommentID, "ark", cats, res.Reason)
				if e2 == nil && changed {
					_ = rdb.Del(ctx, commentListKey(e.VideoID))
				}
				return e2
			default:
				changed, e2 := markSuspect(ctx, db, e.CommentID, "ark", cats, res.Reason)
				if e2 == nil && changed {
					_ = rdb.Del(ctx, commentListKey(e.VideoID))
				}
				return e2
			}
		})
	}()

	// Popularity 消费者：更新 Redis feed:hot
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("worker: consuming %s", events.QueuePopularity)
		_ = mq.Consume(events.QueuePopularity, func(body []byte) error {
			var e events.PopularityEvent
			if err := json.Unmarshal(body, &e); err != nil {
				return err
			}
			if e.EventID != "" {
				ok, err := recordProcessedEvent(ctx, db, e.EventID, "popularity")
				if err != nil {
					return err
				}
				if !ok {
					return nil
				}
			}
			member := formatVideoID(e.VideoID)
			return rdb.ZIncrBy(ctx, RedisKeyHot, float64(e.Delta), member)
		})
	}()

	// Play 消费者：异步落库 play_records 并更新 play_count、popularity
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("worker: consuming %s", events.QueuePlay)
		_ = mq.Consume(events.QueuePlay, func(body []byte) error {
			var e events.PlayEvent
			if err := json.Unmarshal(body, &e); err != nil {
				return err
			}
			return handlePlayEventIdempotent(ctx, db, e)
		})
	}()

	log.Printf("worker started, consuming 6 queues. Press Ctrl+C to stop")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Printf("worker shutting down...")
	cancel()
	mq.Close() // 关闭连接使 Consume 返回
	wg.Wait()
	log.Printf("worker stopped")
}

func joinCats(cats []string) string {
	out := make([]byte, 0, 64)
	for i := range cats {
		if i > 0 {
			out = append(out, ',')
		}
		out = append(out, []byte(cats[i])...)
	}
	return string(out)
}

func approveComment(ctx context.Context, db *gorm.DB, commentID, videoID uint, auditType, cats, note string) (bool, error) {
	now := time.Now()
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Exec(`UPDATE comments SET status='approved', audit_type=?, audit_cats=?, audit_note=?, reviewed_at=? WHERE id=? AND status='pending' AND deleted_at IS NULL`, auditType, cats, note, now, commentID)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			// 已处理（非 pending）则幂等忽略，避免重复计数
			return nil
		}
		// 评论权重=4（与 Video/Worker 热度公式一致）
		return tx.Exec(`UPDATE videos SET comment_count = GREATEST(comment_count + 1, 0), popularity = GREATEST(popularity + 4, 0) WHERE id = ?`, videoID).Error
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func blockComment(ctx context.Context, db *gorm.DB, commentID uint, auditType, cats, note string) (bool, error) {
	now := time.Now()
	res := db.WithContext(ctx).Exec(`UPDATE comments SET status='blocked', audit_type=?, audit_cats=?, audit_note=?, reviewed_at=? WHERE id=? AND status='pending' AND deleted_at IS NULL`, auditType, cats, note, now, commentID)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func markSuspect(ctx context.Context, db *gorm.DB, commentID uint, auditType, cats, note string) (bool, error) {
	now := time.Now()
	res := db.WithContext(ctx).Exec(`UPDATE comments SET status='suspect', audit_type=?, audit_cats=?, audit_note=?, reviewed_at=? WHERE id=? AND status='pending' AND deleted_at IS NULL`, auditType, cats, note, now, commentID)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func formatVideoID(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}

// videoScore 用于全量同步
type videoScore struct {
	ID             uint  `gorm:"column:id"`
	LikesCount     int64 `gorm:"column:likes_count"`
	CommentCount   int64 `gorm:"column:comment_count"`
	FavoritesCount int64 `gorm:"column:favorites_count"`
	PlayCount      int64 `gorm:"column:play_count"`
}

func (videoScore) TableName() string { return "videos" }

// 热度权重（与 Video 服务一致）：点赞2 评论4 收藏3 观看1
const (
	wLike     = 2
	wComment  = 4
	wFavorite = 3
	wPlay     = 1
)

// handlePlayEvent 处理播放事件：upsert play_records、更新 play_count 和 popularity（Redis 由 Popularity 消费者更新）
func handlePlayEvent(ctx context.Context, db *gorm.DB, e events.PlayEvent) error {
	now := time.Now()
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			INSERT INTO play_records (account_id, video_id, play_count, last_play_at, created_at)
			VALUES (?, ?, 1, ?, ?)
			ON DUPLICATE KEY UPDATE play_count = play_count + 1, last_play_at = VALUES(last_play_at)
		`, e.AccountID, e.VideoID, now, now).Error; err != nil {
			return err
		}
		return tx.Exec(`UPDATE videos SET play_count = GREATEST(play_count + 1, 0), popularity = GREATEST(popularity + ?, 0) WHERE id = ?`, wPlay, e.VideoID).Error
	})
}

func handlePlayEventIdempotent(ctx context.Context, db *gorm.DB, e events.PlayEvent) error {
	// 兼容老消息：没有 event_id 则退化为非幂等处理
	if e.EventID == "" {
		return handlePlayEvent(ctx, db, e)
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ok, err := recordProcessedEventTx(ctx, tx, e.EventID, "play")
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		now := time.Now()
		if err := tx.Exec(`
			INSERT INTO play_records (account_id, video_id, play_count, last_play_at, created_at)
			VALUES (?, ?, 1, ?, ?)
			ON DUPLICATE KEY UPDATE play_count = play_count + 1, last_play_at = VALUES(last_play_at)
		`, e.AccountID, e.VideoID, now, now).Error; err != nil {
			return err
		}
		return tx.Exec(`UPDATE videos SET play_count = GREATEST(play_count + 1, 0), popularity = GREATEST(popularity + ?, 0) WHERE id = ?`, wPlay, e.VideoID).Error
	})
}

// recordProcessedEvent 写入幂等表：返回 true 表示首次处理，false 表示重复事件。
func recordProcessedEvent(ctx context.Context, db *gorm.DB, eventID, eventType string) (bool, error) {
	if eventID == "" {
		return true, nil
	}
	var ok bool
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		b, err := recordProcessedEventTx(ctx, tx, eventID, eventType)
		if err != nil {
			return err
		}
		ok = b
		return nil
	})
	return ok, err
}

func recordProcessedEventTx(ctx context.Context, tx *gorm.DB, eventID, eventType string) (bool, error) {
	// MySQL: INSERT IGNORE 利用唯一键实现幂等
	res := tx.WithContext(ctx).Exec(`INSERT IGNORE INTO processed_events (event_id, event_type, created_at) VALUES (?, ?, ?)`, eventID, eventType, time.Now())
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// syncLikesAndPopularityFromMySQL 从 MySQL 全量同步到 Redis ZSET，并按加权公式重算 popularity
func syncLikesAndPopularityFromMySQL(ctx context.Context, db *gorm.DB, cfg config.Config, rdb *redis.Client) error {
	// 回填 comment_count（从 comments 表同步，兼容历史数据）
	_ = db.WithContext(ctx).Exec(`
		UPDATE videos v SET comment_count = (
			SELECT COUNT(*) FROM comments c WHERE c.video_id = v.id AND c.deleted_at IS NULL
		)
	`).Error

	var list []videoScore
	if err := db.WithContext(ctx).Model(&videoScore{}).Select("id", "likes_count", "comment_count", "favorites_count", "play_count").Find(&list).Error; err != nil {
		return fmt.Errorf("query videos: %w", err)
	}
	if len(list) == 0 {
		return nil
	}
	// 先清空再写入，避免残留已删除视频
	_ = rdb.Del(ctx, RedisKeyHotLikes)
	_ = rdb.Del(ctx, RedisKeyHot)
	likesMembers := make([]redislib.Z, 0, len(list))
	hotMembers := make([]redislib.Z, 0, len(list))
	for _, v := range list {
		m := strconv.FormatUint(uint64(v.ID), 10)
		likesMembers = append(likesMembers, redislib.Z{Score: float64(v.LikesCount), Member: m})
		// 热度 = 点赞2 + 评论4 + 收藏3 + 观看1
		popularity := wLike*v.LikesCount + wComment*v.CommentCount + wFavorite*v.FavoritesCount + wPlay*v.PlayCount
		hotMembers = append(hotMembers, redislib.Z{Score: float64(popularity), Member: m})
		// 回写 MySQL 以纠正历史数据
		_ = db.WithContext(ctx).Exec("UPDATE videos SET popularity = ? WHERE id = ?", popularity, v.ID).Error
	}
	if err := rdb.ZAdd(ctx, RedisKeyHotLikes, likesMembers...); err != nil {
		return fmt.Errorf("zadd likes: %w", err)
	}
	if err := rdb.ZAdd(ctx, RedisKeyHot, hotMembers...); err != nil {
		return fmt.Errorf("zadd hot: %w", err)
	}
	return nil
}

func dsn(cfg config.DatabaseConfig) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
}
