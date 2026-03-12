package video

import (
	"context"
	"errors"
	"strings"
)

// VideoService 视频服务
type VideoService struct {
	repo    *VideoRepository
	storage Storage
}

// NewVideoService 创建
func NewVideoService(repo *VideoRepository, storage Storage) *VideoService {
	return &VideoService{repo: repo, storage: storage}
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
	return s.repo.Create(ctx, v)
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
	return s.repo.Delete(ctx, id)
}

// ListByAuthorID 按作者列表
func (s *VideoService) ListByAuthorID(ctx context.Context, authorID uint) ([]Video, error) {
	return s.repo.ListByAuthorID(ctx, authorID)
}

// GetDetail 视频详情
func (s *VideoService) GetDetail(ctx context.Context, id uint) (*Video, error) {
	return s.repo.GetByID(ctx, id)
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
