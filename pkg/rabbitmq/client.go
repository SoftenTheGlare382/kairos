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

// Publish 发布消息到指定队列
func (c *Client) Publish(queue string, body interface{}) error {
	if err := c.ensureQueue(queue); err != nil {
		return err
	}
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
		ContentType: "application/json",
		Body:        data,
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
