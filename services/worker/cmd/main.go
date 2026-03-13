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
	"kairos/pkg/rabbitmq"
	"kairos/pkg/redis"

	redislib "github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Redis key 与 Feed 热榜约定一致
const (
	RedisKeyHotLikes = "feed:hot:likes" // video_id -> likes_count
	RedisKeyHot      = "feed:hot"       // video_id -> popularity
)

func main() {
	config.LoadEnvFromSearchPaths(true)
	cfg := config.Load()

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
			return nil
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
			return handlePlayEvent(ctx, db, e)
		})
	}()

	log.Printf("worker started, consuming 5 queues. Press Ctrl+C to stop")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Printf("worker shutting down...")
	cancel()
	mq.Close() // 关闭连接使 Consume 返回
	wg.Wait()
	log.Printf("worker stopped")
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
