package im

import (
	"context"
	"encoding/json"
	"log"
	"strconv"

	"github.com/meilisearch/meilisearch-go"
)

// MessageSearchClient 消息搜索接口（Meilisearch 等实现）
type MessageSearchClient interface {
	IndexMessage(ctx context.Context, m *Message) error
	Search(ctx context.Context, userID uint, query string, limit, offset int) ([]uint, int64, error)
	EnsureIndex(ctx context.Context) error
}

// meilisearchMessageClient Meilisearch 实现
type meilisearchMessageClient struct {
	client meilisearch.ServiceManager
	index  string
}

// NewMeilisearchMessageClient 创建
func NewMeilisearchMessageClient(host, apiKey, indexName string) *meilisearchMessageClient {
	var client meilisearch.ServiceManager
	if apiKey != "" {
		client = meilisearch.New(host, meilisearch.WithAPIKey(apiKey))
	} else {
		client = meilisearch.New(host)
	}
	if indexName == "" {
		indexName = "im_messages"
	}
	return &meilisearchMessageClient{client: client, index: indexName}
}

// messageDoc Meilisearch 文档格式
type messageDoc struct {
	ID             string `json:"id"`
	ConversationID uint   `json:"conversation_id"`
	SenderID       uint   `json:"sender_id"`
	ReceiverID     uint   `json:"receiver_id"`
	Content        string `json:"content"`
	CreatedAt      int64  `json:"created_at"` // Unix 秒，用于排序
}

func (c *meilisearchMessageClient) IndexMessage(ctx context.Context, m *Message) error {
	if m == nil {
		return nil
	}
	doc := messageDoc{
		ID:             strconv.FormatUint(uint64(m.ID), 10),
		ConversationID: m.ConversationID,
		SenderID:       m.SenderID,
		ReceiverID:     m.ReceiverID,
		Content:        m.Content,
		CreatedAt:      m.CreatedAt.Unix(),
	}
	_, err := c.client.Index(c.index).AddDocuments([]messageDoc{doc}, nil)
	if err != nil {
		log.Printf("meilisearch index message %d: %v", m.ID, err)
		return err
	}
	return nil
}

func (c *meilisearchMessageClient) Search(ctx context.Context, userID uint, query string, limit, offset int) ([]uint, int64, error) {
	if query == "" {
		return nil, 0, nil
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	// 仅搜索当前用户参与的消息（发送或接收）
	filter := "sender_id = " + strconv.FormatUint(uint64(userID), 10) + " OR receiver_id = " + strconv.FormatUint(uint64(userID), 10)
	res, err := c.client.Index(c.index).Search(query, &meilisearch.SearchRequest{
		Limit:  int64(limit),
		Offset: int64(offset),
		Filter: filter,
		Sort:   []string{"created_at:desc"},
	})
	if err != nil {
		return nil, 0, err
	}
	ids := make([]uint, 0, len(res.Hits))
	for _, h := range res.Hits {
		idVal, ok := h["id"]
		if !ok {
			continue
		}
		var id uint
		var decoded interface{}
		if err := json.Unmarshal(idVal, &decoded); err != nil {
			continue
		}
		switch v := decoded.(type) {
		case string:
			u, _ := strconv.ParseUint(v, 10, 64)
			id = uint(u)
		case float64:
			id = uint(v)
		default:
			continue
		}
		ids = append(ids, id)
	}
	return ids, res.EstimatedTotalHits, nil
}

func (c *meilisearchMessageClient) EnsureIndex(ctx context.Context) error {
	_, _ = c.client.CreateIndex(&meilisearch.IndexConfig{
		Uid:        c.index,
		PrimaryKey: "id",
	})
	_, _ = c.client.Index(c.index).UpdateSearchableAttributes(&[]string{"content"})
	_, _ = c.client.Index(c.index).UpdateFilterableAttributes(&[]interface{}{"sender_id", "receiver_id", "conversation_id", "created_at"})
	_, _ = c.client.Index(c.index).UpdateSortableAttributes(&[]string{"created_at"})
	return nil
}
