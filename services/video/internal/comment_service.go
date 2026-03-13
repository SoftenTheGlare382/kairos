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

const commentListCacheTTL = 2 * time.Minute

func commentListKey(videoID uint) string { return fmt.Sprintf("comment:list:%d", videoID) }

// CommentPublisher 评论/热度事件发布接口（可选，供 Worker 消费）
type CommentPublisher interface {
	PublishComment(videoID uint, delta int64)
	PublishPopularity(videoID uint, delta int64)
}

// CommentService 评论服务
type CommentService struct {
	commentRepo *CommentRepository
	videoRepo   *VideoRepository
	publisher   CommentPublisher
	rdb         *redis.Client
	bloom       *bloomfilter.Filter // 可选，video 存在性防穿透
}

// NewCommentService 创建，publisher、rdb、bloom 可为 nil
func NewCommentService(commentRepo *CommentRepository, videoRepo *VideoRepository, publisher CommentPublisher, rdb *redis.Client, bloom *bloomfilter.Filter) *CommentService {
	return &CommentService{commentRepo: commentRepo, videoRepo: videoRepo, publisher: publisher, rdb: rdb, bloom: bloom}
}

// Publish 发布评论
func (s *CommentService) Publish(ctx context.Context, c *Comment) error {
	if c == nil {
		return errors.New("comment is nil")
	}
	c.Content = strings.TrimSpace(c.Content)
	if c.VideoID == 0 || c.AuthorID == 0 {
		return errors.New("video_id and author_id are required")
	}
	if c.Content == "" {
		return errors.New("content is required")
	}
	ok, err := s.videoRepo.IsExist(ctx, c.VideoID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("video not found")
	}
	if err := s.commentRepo.Create(ctx, c); err != nil {
		return err
	}
	if s.rdb != nil {
		s.rdb.Del(ctx, commentListKey(c.VideoID))
	}
	if err := s.videoRepo.UpdateCommentCount(ctx, c.VideoID, 1); err != nil {
		return err
	}
	if err := s.videoRepo.UpdatePopularity(ctx, c.VideoID, PopularityWeightComment); err != nil {
		return err
	}
	if s.publisher != nil {
		s.publisher.PublishComment(c.VideoID, 1)
		s.publisher.PublishPopularity(c.VideoID, PopularityWeightComment)
	}
	return nil
}

// Delete 删除评论（仅作者可删）
func (s *CommentService) Delete(ctx context.Context, commentID uint, accountID uint) error {
	c, err := s.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return err
	}
	if c == nil {
		return errors.New("comment not found")
	}
	if c.AuthorID != accountID {
		return errors.New("permission denied")
	}
	if err := s.commentRepo.Delete(ctx, c); err != nil {
		return err
	}
	if s.rdb != nil {
		s.rdb.Del(ctx, commentListKey(c.VideoID))
	}
	if err := s.videoRepo.UpdateCommentCount(ctx, c.VideoID, -1); err != nil {
		return err
	}
	if err := s.videoRepo.UpdatePopularity(ctx, c.VideoID, -PopularityWeightComment); err != nil {
		return err
	}
	if s.publisher != nil {
		s.publisher.PublishComment(c.VideoID, -1)
		s.publisher.PublishPopularity(c.VideoID, -PopularityWeightComment)
	}
	return nil
}

// GetAll 获取视频下所有评论（布隆过滤不存在的 video，有 rdb 时读缓存）
func (s *CommentService) GetAll(ctx context.Context, videoID uint) ([]Comment, error) {
	if s.bloom != nil && s.bloom.ShouldReject("video:"+strconv.FormatUint(uint64(videoID), 10)) {
		log.Printf("bloom reject: video:%d (comment list)", videoID)
		return nil, errors.New("video not found")
	}
	ok, err := s.videoRepo.IsExist(ctx, videoID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("video not found")
	}
	if s.rdb != nil {
		b, err := s.rdb.GetBytes(ctx, commentListKey(videoID))
		if err == nil && len(b) > 0 {
			var list []Comment
			if json.Unmarshal(b, &list) == nil {
				log.Printf("cache hit: comment:list:%d", videoID)
				return list, nil
			}
		}
	}
	list, err := s.commentRepo.ListByVideoID(ctx, videoID)
	if err != nil {
		return nil, err
	}
	if s.rdb != nil {
		if b, err := json.Marshal(list); err == nil {
			s.rdb.SetBytes(ctx, commentListKey(videoID), b, redis.TTLWithJitter(commentListCacheTTL, 0.2))
		}
	}
	return list, nil
}
