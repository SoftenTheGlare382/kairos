package im

import (
	"kairos/pkg/rabbitmq"
)

// MQMessagePublisher 基于 RabbitMQ 的消息持久化发布者
type MQMessagePublisher struct {
	mq *rabbitmq.Client
}

// NewMQMessagePublisher 创建
func NewMQMessagePublisher(mq *rabbitmq.Client) *MQMessagePublisher {
	return &MQMessagePublisher{mq: mq}
}

// PublishMessage 发布到 im.message.persist 队列（队列由 main 启动时 EnsureQueueWithDLQ 预先声明，此处不再声明）
func (p *MQMessagePublisher) PublishMessage(payload MessagePersistPayload) error {
	return p.mq.PublishToQueue(QueueIMMessagePersist, payload)
}
