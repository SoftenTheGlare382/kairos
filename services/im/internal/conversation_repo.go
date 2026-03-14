package im

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// ConversationRepository 会话仓储
type ConversationRepository struct {
	db *gorm.DB
}

// NewConversationRepository 创建
func NewConversationRepository(db *gorm.DB) *ConversationRepository {
	return &ConversationRepository{db: db}
}

// GetOrCreate 获取或创建会话（userA、userB 任意顺序，内部规范为 user_a < user_b）
func (r *ConversationRepository) GetOrCreate(ctx context.Context, userA, userB uint) (*Conversation, error) {
	if userA > userB {
		userA, userB = userB, userA
	}
	var c Conversation
	err := r.db.WithContext(ctx).Where("user_a = ? AND user_b = ?", userA, userB).First(&c).Error
	if err == nil {
		return &c, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	now := time.Now()
	c = Conversation{UserA: userA, UserB: userB, LastMessageAt: now}
	if err := r.db.WithContext(ctx).Create(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// GetByID 按 ID 获取
func (r *ConversationRepository) GetByID(ctx context.Context, id uint) (*Conversation, error) {
	var c Conversation
	if err := r.db.WithContext(ctx).First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// ListByUserID 某用户参与的会话列表，按最后消息时间倒序
func (r *ConversationRepository) ListByUserID(ctx context.Context, userID uint, limit, offset int) ([]Conversation, error) {
	var list []Conversation
	err := r.db.WithContext(ctx).
		Where("user_a = ? OR user_b = ?", userID, userID).
		Order("last_message_at desc, id desc").
		Limit(limit).Offset(offset).
		Find(&list).Error
	return list, err
}

// UpdateLastMessageAt 更新会话最后消息时间
func (r *ConversationRepository) UpdateLastMessageAt(ctx context.Context, convID uint, t time.Time) error {
	return r.db.WithContext(ctx).Model(&Conversation{}).Where("id = ?", convID).Update("last_message_at", t).Error
}
