package im

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// ReadRepository 已读状态仓储
type ReadRepository struct {
	db *gorm.DB
}

// NewReadRepository 创建
func NewReadRepository(db *gorm.DB) *ReadRepository {
	return &ReadRepository{db: db}
}

// GetLastReadAt 获取用户在某会话的最后已读时间，不存在返回零值（用 Find 替代 First，避免 record not found 日志）
func (r *ReadRepository) GetLastReadAt(ctx context.Context, userID, convID uint) (time.Time, error) {
	var cr ConversationRead
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND conversation_id = ?", userID, convID).
		Limit(1).
		Find(&cr)
	if result.Error != nil {
		return time.Time{}, result.Error
	}
	if result.RowsAffected == 0 {
		return time.Time{}, nil
	}
	return cr.LastReadAt, nil
}

// MarkRead 标记会话已读（更新 last_read_at），用 Find 替代 First 避免 record not found 日志
func (r *ReadRepository) MarkRead(ctx context.Context, userID, convID uint, t time.Time) error {
	var cr ConversationRead
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND conversation_id = ?", userID, convID).
		Limit(1).
		Find(&cr)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return r.db.WithContext(ctx).Create(&ConversationRead{
			UserID:         userID,
			ConversationID: convID,
			LastReadAt:     t,
			UpdatedAt:      t,
		}).Error
	}
	return r.db.WithContext(ctx).Model(&cr).Updates(map[string]interface{}{"last_read_at": t, "updated_at": t}).Error
}
