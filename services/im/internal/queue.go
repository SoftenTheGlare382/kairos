package im

const (
	// QueueIMMessagePersist IM 消息持久化队列
	QueueIMMessagePersist = "im.message.persist"
	// QueueIMMessagePersistDLQ IM 消息持久化死信队列
	QueueIMMessagePersistDLQ = "im.message.persist.dlq"
)

// MessagePersistPayload MQ 消息持久化载荷（保证一致性：幂等键 + 有序消费）
type MessagePersistPayload struct {
	ConversationID uint   `json:"conversation_id"`
	SenderID       uint   `json:"sender_id"`
	ReceiverID     uint   `json:"receiver_id"`
	Content        string `json:"content"`
	IdempotencyKey string `json:"idempotency_key"` // UUID，去重防重试重复落库
}
