package im

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"kairos/pkg/rabbitmq"
)

var ErrInvalidPayload = errors.New("invalid message payload")

// MessageSearchIndexer 消息搜索索引接口（可选，nil 时不索引）
type MessageSearchIndexer interface {
	IndexMessage(ctx context.Context, m *Message) error
}

// MessageConsumer IM 消息持久化消费者
type MessageConsumer struct {
	mq       *rabbitmq.Client
	convRepo *ConversationRepository
	msgRepo  *MessageRepository
	indexer  MessageSearchIndexer
}

// NewMessageConsumer 创建
func NewMessageConsumer(mq *rabbitmq.Client, convRepo *ConversationRepository, msgRepo *MessageRepository, indexer MessageSearchIndexer) *MessageConsumer {
	return &MessageConsumer{mq: mq, convRepo: convRepo, msgRepo: msgRepo, indexer: indexer}
}

// Run 启动消费（阻塞），应在 goroutine 中调用
func (c *MessageConsumer) Run(ctx context.Context) {
	if err := c.mq.EnsureQueueWithDLQ(QueueIMMessagePersist, QueueIMMessagePersistDLQ); err != nil {
		log.Printf("im consumer: ensure queue failed: %v", err)
		return
	}
	log.Printf("im consumer: consuming %s (dlq: %s)", QueueIMMessagePersist, QueueIMMessagePersistDLQ)
	_ = c.mq.ConsumeWithDelivery(QueueIMMessagePersist, c.handle)
}

// handle 处理单条消息；返回 (nil, _) Ack，(err, true) Nack 重试，(err, false) Nack 进 DLQ
func (c *MessageConsumer) handle(d *amqp.Delivery) (err error, requeue bool) {
	ctx := context.Background()
	var payload MessagePersistPayload
	if err := json.Unmarshal(d.Body, &payload); err != nil {
		return err, false // 解析失败不入 DLQ 也无意义，直接丢弃？或进 DLQ 便于排查。进 DLQ 更安全
	}
	if payload.IdempotencyKey == "" || payload.ConversationID == 0 {
		return ErrInvalidPayload, false
	}

	key := payload.IdempotencyKey
	msg := &Message{
		ConversationID: payload.ConversationID,
		SenderID:       payload.SenderID,
		ReceiverID:     payload.ReceiverID,
		Content:        payload.Content,
		IdempotencyKey: &key,
	}
	if err := c.msgRepo.CreateWithIdempotency(ctx, msg); err != nil {
		if err == ErrDuplicateIdempotencyKey {
			// 幂等：已落库，视为成功
			return nil, false // requeue 在 err==nil 时会被忽略
		}
		// 其他错误：首次重试，第二次进 DLQ
		if d.Redelivered {
			return err, false
		}
		return err, true
	}

	now := time.Now()
	if err := c.convRepo.UpdateLastMessageAt(ctx, payload.ConversationID, now); err != nil {
		// 消息已落库，last_message_at 失败可容忍；重试可能重复更新，幂等
		if d.Redelivered {
			return err, false
		}
		return err, true
	}

	// Meilisearch 索引（可选，失败不影响主流程）
	if c.indexer != nil {
		_ = c.indexer.IndexMessage(ctx, msg)
	}
	return nil, true
}
