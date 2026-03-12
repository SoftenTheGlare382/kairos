package video

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// VideoRepository 视频仓储
type VideoRepository struct {
	db *gorm.DB
}

// NewVideoRepository 创建
func NewVideoRepository(db *gorm.DB) *VideoRepository {
	return &VideoRepository{db: db}
}

// Create 创建视频
func (r *VideoRepository) Create(ctx context.Context, v *Video) error {
	return r.db.WithContext(ctx).Create(v).Error
}

// Delete 删除
func (r *VideoRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&Video{}, id).Error
}

// GetByID 按 ID 查询
func (r *VideoRepository) GetByID(ctx context.Context, id uint) (*Video, error) {
	var v Video
	if err := r.db.WithContext(ctx).First(&v, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &v, nil
}

// ListByAuthorID 按作者列表
func (r *VideoRepository) ListByAuthorID(ctx context.Context, authorID uint) ([]Video, error) {
	var list []Video
	err := r.db.WithContext(ctx).Where("author_id = ?", authorID).
		Order("create_at desc").Find(&list).Error
	return list, err
}

// ListByAuthorIDs 按多作者列表
func (r *VideoRepository) ListByAuthorIDs(ctx context.Context, authorIDs []uint) ([]Video, error) {
	if len(authorIDs) == 0 {
		return nil, nil
	}
	var list []Video
	err := r.db.WithContext(ctx).Where("author_id IN ?", authorIDs).
		Order("create_at desc").Find(&list).Error
	return list, err
}

// GetByIDs 按 ID 列表查询
func (r *VideoRepository) GetByIDs(ctx context.Context, ids []uint) ([]Video, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var list []Video
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error
	return list, err
}

// IsExist 是否存在
func (r *VideoRepository) IsExist(ctx context.Context, id uint) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Video{}).Where("id = ?", id).Count(&n).Error
	return n > 0, err
}

// UpdateLikesCount 更新点赞数
func (r *VideoRepository) UpdateLikesCount(ctx context.Context, id uint, delta int64) error {
	return r.updateLikesCount(ctx, r.db, id, delta)
}

// UpdateLikesCountInTx 在事务内更新点赞数
func (r *VideoRepository) UpdateLikesCountInTx(ctx context.Context, tx *gorm.DB, id uint, delta int64) error {
	return r.updateLikesCount(ctx, tx, id, delta)
}

func (r *VideoRepository) updateLikesCount(ctx context.Context, db *gorm.DB, id uint, delta int64) error {
	return db.WithContext(ctx).Model(&Video{}).Where("id = ?", id).
		UpdateColumn("likes_count", gorm.Expr("GREATEST(likes_count + ?, 0)", delta)).Error
}

// UpdatePopularity 更新热度
func (r *VideoRepository) UpdatePopularity(ctx context.Context, id uint, delta int64) error {
	return r.db.WithContext(ctx).Model(&Video{}).Where("id = ?", id).
		UpdateColumn("popularity", gorm.Expr("GREATEST(popularity + ?, 0)", delta)).Error
}
