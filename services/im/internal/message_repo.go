package im

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ErrDuplicateIdempotencyKey 幂等键重复（已落库，视为成功）
var ErrDuplicateIdempotencyKey = errors.New("duplicate idempotency key")

// MessageRepository 消息仓储
type MessageRepository struct {
	db *gorm.DB
}

// NewMessageRepository 创建
func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

// Create 创建消息
func (r *MessageRepository) Create(ctx context.Context, m *Message) error {
	return r.db.WithContext(ctx).Create(m).Error
}

// CreateWithIdempotency 创建消息（带幂等键）；若幂等键已存在则返回 ErrDuplicateIdempotencyKey 表示已落库
func (r *MessageRepository) CreateWithIdempotency(ctx context.Context, m *Message) error {
	err := r.db.WithContext(ctx).Create(m).Error
	if err != nil && isDuplicateKeyErr(err) {
		return ErrDuplicateIdempotencyKey
	}
	return err
}

func isDuplicateKeyErr(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	// MySQL 1062 = Duplicate entry for key
	return strings.Contains(s, "Duplicate entry") || strings.Contains(s, "1062")
}

// CountFromSenderToReceiver 统计 sender 发给 receiver 的消息数（用于「非互关仅能发一条」规则）
func (r *MessageRepository) CountFromSenderToReceiver(ctx context.Context, senderID, receiverID uint) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Message{}).
		Where("sender_id = ? AND receiver_id = ?", senderID, receiverID).
		Count(&n).Error
	return n, err
}

// ListByConversationID 按会话 ID 分页查询消息
func (r *MessageRepository) ListByConversationID(ctx context.Context, convID uint, limit, offset int) ([]Message, error) {
	var list []Message
	err := r.db.WithContext(ctx).
		Where("conversation_id = ?", convID).
		Order("created_at desc").
		Limit(limit).Offset(offset).
		Find(&list).Error
	return list, err
}

// ListAllForSearch 列出所有消息（用于 Meilisearch 初始同步）
func (r *MessageRepository) ListAllForSearch(ctx context.Context) ([]Message, error) {
	var list []Message
	err := r.db.WithContext(ctx).Find(&list).Error
	return list, err
}

// GetByIDs 按 ID 列表查询消息（不保证顺序）
func (r *MessageRepository) GetByIDs(ctx context.Context, ids []uint) ([]Message, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var list []Message
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error
	return list, err
}

// CountUnread 统计用户在某会话中收到的、晚于 lastReadAt 的消息数
func (r *MessageRepository) CountUnread(ctx context.Context, convID, userID uint, lastReadAt time.Time) (int64, error) {
	var n int64
	q := r.db.WithContext(ctx).Model(&Message{}).
		Where("conversation_id = ? AND receiver_id = ?", convID, userID)
	if !lastReadAt.IsZero() {
		q = q.Where("created_at > ?", lastReadAt)
	}
	err := q.Count(&n).Error
	return n, err
}
