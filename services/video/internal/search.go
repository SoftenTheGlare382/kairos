package video

import (
	"context"
	"encoding/json"
	"log"
	"strconv"

	"github.com/meilisearch/meilisearch-go"
)

const searchIndexName = "videos"

// SearchClient 搜索索引接口（Meilisearch 等实现）
type SearchClient interface {
	IndexVideo(ctx context.Context, v *Video) error
	DeleteVideo(ctx context.Context, id uint) error
	Search(ctx context.Context, query string, limit, offset int) ([]uint, int64, error)
}

// meilisearchClient Meilisearch 实现
type meilisearchClient struct {
	client meilisearch.ServiceManager
	index  string
}

// NewMeilisearchClient 创建 Meilisearch 搜索客户端
func NewMeilisearchClient(host, apiKey, indexName string) *meilisearchClient {
	var client meilisearch.ServiceManager
	if apiKey != "" {
		client = meilisearch.New(host, meilisearch.WithAPIKey(apiKey))
	} else {
		client = meilisearch.New(host)
	}
	if indexName == "" {
		indexName = searchIndexName
	}
	return &meilisearchClient{client: client, index: indexName}
}

// videoDoc Meilisearch 文档格式
type videoDoc struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (c *meilisearchClient) IndexVideo(ctx context.Context, v *Video) error {
	if v == nil {
		return nil
	}
	doc := videoDoc{
		ID:          strconv.FormatUint(uint64(v.ID), 10),
		Title:       v.Title,
		Description: v.Description,
	}
	_, err := c.client.Index(c.index).AddDocuments([]videoDoc{doc}, nil)
	if err != nil {
		log.Printf("meilisearch index video %d: %v", v.ID, err)
		return err
	}
	return nil
}

func (c *meilisearchClient) DeleteVideo(ctx context.Context, id uint) error {
	_, err := c.client.Index(c.index).DeleteDocument(strconv.FormatUint(uint64(id), 10), nil)
	if err != nil {
		log.Printf("meilisearch delete video %d: %v", id, err)
		return err
	}
	return nil
}


func (c *meilisearchClient) Search(ctx context.Context, query string, limit, offset int) ([]uint, int64, error) {
	if query == "" {
		return nil, 0, nil
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	res, err := c.client.Index(c.index).Search(query, &meilisearch.SearchRequest{
		Limit:  int64(limit),
		Offset: int64(offset),
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

// EnsureIndex 确保索引存在并配置可搜索属性
func (c *meilisearchClient) EnsureIndex(ctx context.Context) error {
	_, _ = c.client.CreateIndex(&meilisearch.IndexConfig{
		Uid:        c.index,
		PrimaryKey: "id",
	})
	// 已存在时 CreateIndex 会报错，忽略；设置可搜索属性：标题优先，描述次要
	_, _ = c.client.Index(c.index).UpdateSearchableAttributes(&[]string{"title", "description"})
	return nil
}
