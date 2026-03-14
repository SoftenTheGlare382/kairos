package video

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"kairos/pkg/bloomfilter"
	"kairos/pkg/redis"
)

const videoDetailCacheTTL = 5 * time.Minute

func videoDetailKey(id uint) string { return fmt.Sprintf("video:detail:%d", id) }

// VideoService 视频服务
type VideoService struct {
	repo         *VideoRepository
	storage      Storage
	rdb          *redis.Client      // 可选，GetDetail 缓存、Delete 时失效
	bloom        *bloomfilter.Filter // 可选，防缓存穿透
	searchClient SearchClient       // 可选，Meilisearch 模糊搜索
}

// NewVideoService 创建
func NewVideoService(repo *VideoRepository, storage Storage, rdb *redis.Client, bloom *bloomfilter.Filter, searchClient SearchClient) *VideoService {
	return &VideoService{repo: repo, storage: storage, rdb: rdb, bloom: bloom, searchClient: searchClient}
}

// Publish 发布视频
func (s *VideoService) Publish(ctx context.Context, v *Video) error {
	if v == nil {
		return errors.New("video is nil")
	}
	v.Title = strings.TrimSpace(v.Title)
	v.PlayURL = strings.TrimSpace(v.PlayURL)
	v.CoverURL = strings.TrimSpace(v.CoverURL)
	if v.Title == "" {
		return errors.New("title is required")
	}
	if v.PlayURL == "" {
		return errors.New("play_url is required")
	}
	if v.CoverURL == "" {
		return errors.New("cover_url is required")
	}
	if err := s.repo.Create(ctx, v); err != nil {
		return err
	}
	if s.bloom != nil {
		s.bloom.Add("video:" + strconv.FormatUint(uint64(v.ID), 10))
	}
	if s.searchClient != nil {
		_ = s.searchClient.IndexVideo(ctx, v)
	}
	if s.rdb != nil {
		s.rdb.Del(ctx, "feed:latest:ids")
	}
	return nil
}

// Delete 删除视频（仅作者可删）
func (s *VideoService) Delete(ctx context.Context, id uint, authorID uint) error {
	v, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if v == nil {
		return errors.New("video not found")
	}
	if v.AuthorID != authorID {
		return errors.New("unauthorized")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if s.searchClient != nil {
		_ = s.searchClient.DeleteVideo(ctx, id)
	}
	if s.rdb != nil {
		s.rdb.Del(ctx, videoDetailKey(id))
		s.rdb.Del(ctx, "feed:latest:ids")       // 最新列表含该视频 ID，需失效
		s.rdb.Del(ctx, fmt.Sprintf("comment:list:%d", id)) // 该视频评论列表可废弃
	}
	return nil
}

// ListByAuthorID 按作者列表
func (s *VideoService) ListByAuthorID(ctx context.Context, authorID uint) ([]Video, error) {
	return s.repo.ListByAuthorID(ctx, authorID)
}

// GetDetail 视频详情（布隆过滤防穿透，有 rdb 时读缓存）
func (s *VideoService) GetDetail(ctx context.Context, id uint) (*Video, error) {
	if s.bloom != nil && s.bloom.ShouldReject("video:"+strconv.FormatUint(uint64(id), 10)) {
		log.Printf("bloom reject: video:%d", id)
		return nil, nil
	}
	if s.rdb != nil {
		b, err := s.rdb.GetBytes(ctx, videoDetailKey(id))
		if err == nil && len(b) > 0 {
			var v Video
			if json.Unmarshal(b, &v) == nil {
				log.Printf("cache hit: video:detail:%d", id)
				return &v, nil
			}
		}
	}
	v, err := s.repo.GetByID(ctx, id)
	if err != nil || v == nil {
		return v, err
	}
	if s.rdb != nil {
		if b, err := json.Marshal(v); err == nil {
			s.rdb.SetBytes(ctx, videoDetailKey(id), b, redis.TTLWithJitter(videoDetailCacheTTL, 0.2))
		}
	}
	return v, nil
}

// ListByAuthorIDs 按多个作者列表（供 Feed 等调用）
func (s *VideoService) ListByAuthorIDs(ctx context.Context, authorIDs []uint) ([]Video, error) {
	return s.repo.ListByAuthorIDs(ctx, authorIDs)
}

// GetByIDs 按 ID 列表查询（供 Feed 等调用）
func (s *VideoService) GetByIDs(ctx context.Context, ids []uint) ([]Video, error) {
	return s.repo.GetByIDs(ctx, ids)
}

// UpdateLikesCount 更新点赞数（供 Worker 或直接调用）
func (s *VideoService) UpdateLikesCount(ctx context.Context, id uint, delta int64) error {
	return s.repo.UpdateLikesCount(ctx, id, delta)
}

// UpdatePopularity 更新热度
func (s *VideoService) UpdatePopularity(ctx context.Context, id uint, delta int64) error {
	return s.repo.UpdatePopularity(ctx, id, delta)
}

// Search 按标题/描述模糊搜索视频（依赖 Meilisearch）
func (s *VideoService) Search(ctx context.Context, query string, limit, offset int) ([]Video, int64, error) {
	if s.searchClient == nil {
		return nil, 0, nil
	}
	ids, total, err := s.searchClient.Search(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	if len(ids) == 0 {
		return []Video{}, total, nil
	}
	list, err := s.repo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	// 保持 Meilisearch 返回的排序
	idOrder := make(map[uint]int)
	for i, id := range ids {
		idOrder[id] = i
	}
	// 按 ids 顺序排列
	sorted := make([]Video, len(list))
	for _, v := range list {
		if idx, ok := idOrder[v.ID]; ok {
			sorted[idx] = v
		}
	}
	return sorted, total, nil
}
