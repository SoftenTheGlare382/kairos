package rabbitmq

import (
	"encoding/json"
	"fmt"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Client RabbitMQ 客户端（发布与消费）
type Client struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	mu      sync.RWMutex
	url     string
}

// New 创建客户端，url 格式 amqp://user:pass@host:5672/
func New(url string) (*Client, error) {
	if url == "" {
		url = "amqp://guest:guest@127.0.0.1:5672/"
	}
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("rabbitmq channel: %w", err)
	}
	return &Client{conn: conn, channel: ch, url: url}, nil
}

// Close 关闭连接
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var err error
	if c.channel != nil {
		err = c.channel.Close()
		c.channel = nil
	}
	if c.conn != nil {
		if e := c.conn.Close(); e != nil && err == nil {
			err = e
		}
		c.conn = nil
	}
	return err
}

// ensureQueue 声明队列（幂等）
func (c *Client) ensureQueue(name string) error {
	c.mu.RLock()
	ch := c.channel
	c.mu.RUnlock()
	if ch == nil {
		return fmt.Errorf("channel closed")
	}
	_, err := ch.QueueDeclare(name, true, false, false, false, nil)
	return err
}

// EnsureQueueWithDLQ 声明带死信队列的主队列（幂等），需先声明 dlq 队列
func (c *Client) EnsureQueueWithDLQ(queue, dlq string) error {
	c.mu.RLock()
	ch := c.channel
	c.mu.RUnlock()
	if ch == nil {
		return fmt.Errorf("channel closed")
	}
	if _, err := ch.QueueDeclare(dlq, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dlq %s: %w", dlq, err)
	}
	args := amqp.Table{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": dlq,
	}
	if _, err := ch.QueueDeclare(queue, true, false, false, false, args); err != nil {
		return fmt.Errorf("declare queue %s: %w", queue, err)
	}
	return nil
}

// Publish 发布消息到指定队列（会声明队列，无特殊参数）
func (c *Client) Publish(queue string, body interface{}) error {
	if err := c.ensureQueue(queue); err != nil {
		return err
	}
	return c.publishToQueue(queue, body)
}

// PublishToQueue 直接发布到队列，不声明（用于已用 EnsureQueueWithDLQ 等预先声明的队列）
func (c *Client) PublishToQueue(queue string, body interface{}) error {
	return c.publishToQueue(queue, body)
}

func (c *Client) publishToQueue(queue string, body interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	c.mu.RLock()
	ch := c.channel
	c.mu.RUnlock()
	if ch == nil {
		return fmt.Errorf("channel closed")
	}
	return ch.PublishWithContext(nil, "", queue, false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         data,
		DeliveryMode: amqp.Persistent,
	})
}

// Consume 消费队列，handler 返回 error 时 Nack 并重新入队
func (c *Client) Consume(queue string, handler func(body []byte) error) error {
	if err := c.ensureQueue(queue); err != nil {
		return err
	}
	c.mu.RLock()
	ch := c.channel
	c.mu.RUnlock()
	if ch == nil {
		return fmt.Errorf("channel closed")
	}
	if err := ch.Qos(1, 0, false); err != nil {
		return err
	}
	deliveries, err := ch.Consume(queue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	for d := range deliveries {
		if err := handler(d.Body); err != nil {
			_ = d.Nack(false, true)
			continue
		}
		_ = d.Ack(false)
	}
	return nil
}

// ConsumeWithDelivery 消费队列，handler 可控制 Nack 是否 requeue；返回 (nil, false) 表示 Ack，返回 (err, true) 表示 Nack(requeue)，返回 (err, false) 表示 Nack(requeue=false) 进死信队列
func (c *Client) ConsumeWithDelivery(queue string, handler func(d *amqp.Delivery) (err error, requeue bool)) error {
	c.mu.RLock()
	ch := c.channel
	c.mu.RUnlock()
	if ch == nil {
		return fmt.Errorf("channel closed")
	}
	if err := ch.Qos(1, 0, false); err != nil {
		return err
	}
	deliveries, err := ch.Consume(queue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	for d := range deliveries {
		err, requeue := handler(&d)
		if err != nil {
			_ = d.Nack(false, requeue)
			continue
		}
		_ = d.Ack(false)
	}
	return nil
}
