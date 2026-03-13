package video

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// PlayEventPublisher 播放事件发布接口（Worker 异步落库）
type PlayEventPublisher interface {
	PublishPlay(accountID, videoID uint)
}

// PlayService 播放统计服务
type PlayService struct {
	db             *gorm.DB
	playRecordRepo *PlayRecordRepository
	videoRepo      *VideoRepository
	publisher      PopularityPublisher  // 更新 Redis 热度
	playPublisher  PlayEventPublisher   // 异步落库，非 nil 时走 MQ
}

// NewPlayService 创建，publisher/playPublisher 可为 nil；playPublisher 非 nil 时 RecordPlay 走异步
func NewPlayService(db *gorm.DB, playRecordRepo *PlayRecordRepository, videoRepo *VideoRepository, publisher PopularityPublisher, playPublisher PlayEventPublisher) *PlayService {
	return &PlayService{db: db, playRecordRepo: playRecordRepo, videoRepo: videoRepo, publisher: publisher, playPublisher: playPublisher}
}

// RecordPlay 记录一次播放（需登录）；playPublisher 非 nil 时异步落库，否则同步写 MySQL
func (s *PlayService) RecordPlay(ctx context.Context, accountID, videoID uint) error {
	if accountID == 0 || videoID == 0 {
		return errors.New("account_id and video_id are required")
	}
	ok, err := s.videoRepo.IsExist(ctx, videoID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("video not found")
	}
	if s.playPublisher != nil {
		if s.publisher != nil {
			s.publisher.PublishPopularity(videoID, PopularityWeightPlay)
		}
		s.playPublisher.PublishPlay(accountID, videoID)
		return nil
	}
	now := time.Now()
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			INSERT INTO play_records (account_id, video_id, play_count, last_play_at, created_at)
			VALUES (?, ?, 1, ?, ?)
			ON DUPLICATE KEY UPDATE play_count = play_count + 1, last_play_at = VALUES(last_play_at)
		`, accountID, videoID, now, now).Error; err != nil {
			return err
		}
		if err := s.videoRepo.UpdatePlayCountInTx(ctx, tx, videoID, 1); err != nil {
			return err
		}
		return s.videoRepo.UpdatePopularityInTx(ctx, tx, videoID, PopularityWeightPlay)
	})
	if err == nil && s.publisher != nil {
		s.publisher.PublishPopularity(videoID, PopularityWeightPlay)
	}
	return err
}

// ListPlayRecords 获取某视频的播放记录（谁播放了几次、最近播放时间），仅作者可查
func (s *PlayService) ListPlayRecords(ctx context.Context, videoID uint, authorID uint, limit, offset int) ([]PlayRecord, error) {
	if videoID == 0 {
		return nil, errors.New("video_id is required")
	}
	v, err := s.videoRepo.GetByID(ctx, videoID)
	if err != nil || v == nil {
		return nil, errors.New("video not found")
	}
	if v.AuthorID != authorID {
		return nil, errors.New("permission denied: only video author can view play records")
	}
	return s.playRecordRepo.ListByVideoID(ctx, videoID, limit, offset)
}
