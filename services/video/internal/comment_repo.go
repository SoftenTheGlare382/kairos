package video

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// CommentRepository 评论仓储
type CommentRepository struct {
	db *gorm.DB
}

// NewCommentRepository 创建
func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

// Create 创建
func (r *CommentRepository) Create(ctx context.Context, c *Comment) error {
	return r.db.WithContext(ctx).Create(c).Error
}

// GetByID 按 ID 查询
func (r *CommentRepository) GetByID(ctx context.Context, id uint) (*Comment, error) {
	var c Comment
	if err := r.db.WithContext(ctx).First(&c, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

// Delete 删除
func (r *CommentRepository) Delete(ctx context.Context, c *Comment) error {
	return r.db.WithContext(ctx).Delete(c).Error
}

// ListByVideoID 按视频列出评论
func (r *CommentRepository) ListByVideoID(ctx context.Context, videoID uint) ([]Comment, error) {
	var list []Comment
	err := r.db.WithContext(ctx).Where("video_id = ?", videoID).
		Order("created_at asc").Find(&list).Error
	return list, err
}
