package video

import (
	"context"
	"errors"
	"strings"
)

// CommentService 评论服务
type CommentService struct {
	commentRepo *CommentRepository
	videoRepo   *VideoRepository
}

// NewCommentService 创建
func NewCommentService(commentRepo *CommentRepository, videoRepo *VideoRepository) *CommentService {
	return &CommentService{commentRepo: commentRepo, videoRepo: videoRepo}
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
	return s.videoRepo.UpdatePopularity(ctx, c.VideoID, 1)
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
	return s.videoRepo.UpdatePopularity(ctx, c.VideoID, -1)
}

// GetAll 获取视频下所有评论
func (s *CommentService) GetAll(ctx context.Context, videoID uint) ([]Comment, error) {
	ok, err := s.videoRepo.IsExist(ctx, videoID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("video not found")
	}
	return s.commentRepo.ListByVideoID(ctx, videoID)
}
