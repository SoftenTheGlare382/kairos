package video

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// PlayRecordRepository 播放记录仓储
type PlayRecordRepository struct {
	db *gorm.DB
}

// NewPlayRecordRepository 创建
func NewPlayRecordRepository(db *gorm.DB) *PlayRecordRepository {
	return &PlayRecordRepository{db: db}
}

// Upsert 创建或更新：存在则 play_count+1、last_play_at 更新，不存在则插入
func (r *PlayRecordRepository) Upsert(ctx context.Context, accountID, videoID uint, now time.Time) error {
	// 使用原生 SQL 或 GORM 的 Clauses 实现 upsert
	// MySQL: INSERT ... ON DUPLICATE KEY UPDATE play_count=play_count+1, last_play_at=VALUES(last_play_at)
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO play_records (account_id, video_id, play_count, last_play_at, created_at)
		VALUES (?, ?, 1, ?, ?)
		ON DUPLICATE KEY UPDATE play_count = play_count + 1, last_play_at = VALUES(last_play_at)
	`, accountID, videoID, now, now).Error
}

// ListByVideoID 按视频 ID 获取播放记录列表（谁播放了几次、最近播放时间），分页
func (r *PlayRecordRepository) ListByVideoID(ctx context.Context, videoID uint, limit, offset int) ([]PlayRecord, error) {
	var list []PlayRecord
	err := r.db.WithContext(ctx).Where("video_id = ?", videoID).
		Order("last_play_at desc").
		Limit(limit).Offset(offset).
		Find(&list).Error
	return list, err
}
