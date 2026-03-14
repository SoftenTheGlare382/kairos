package im

import (
	"time"
)

// Conversation 会话（单聊，user_a < user_b 保证唯一）
type Conversation struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserA          uint      `gorm:"uniqueIndex:idx_conv_ab;not null" json:"user_a"`
	UserB          uint      `gorm:"uniqueIndex:idx_conv_ab;not null" json:"user_b"`
	LastMessageAt  time.Time      `gorm:"index" json:"last_message_at"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// Message 消息
type Message struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ConversationID uint      `gorm:"index;not null" json:"conversation_id"`
	SenderID       uint      `gorm:"index;not null" json:"sender_id"`
	ReceiverID     uint      `gorm:"index;not null" json:"receiver_id"`
	Content        string    `gorm:"type:text;not null" json:"content"`
	IdempotencyKey *string   `gorm:"uniqueIndex:idx_msg_idempotency;size:64" json:"temp_id,omitempty"` // 幂等键，异步落库去重用；NULL 表示历史数据
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// ConversationRead 用户对会话的已读位置（用于计算未读数）
type ConversationRead struct {
	UserID         uint      `gorm:"primaryKey" json:"user_id"`
	ConversationID uint      `gorm:"primaryKey" json:"conversation_id"`
	LastReadAt     time.Time `gorm:"not null" json:"last_read_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
