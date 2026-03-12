package video

import (
	"context"
	"errors"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// LikeRepository 点赞仓储
type LikeRepository struct {
	db *gorm.DB
}

// NewLikeRepository 创建
func NewLikeRepository(db *gorm.DB) *LikeRepository {
	return &LikeRepository{db: db}
}

// Create 创建点赞
func (r *LikeRepository) Create(ctx context.Context, like *Like) error {
	return r.create(ctx, r.db, like)
}

// CreateInTx 在事务内创建点赞
func (r *LikeRepository) CreateInTx(ctx context.Context, tx *gorm.DB, like *Like) error {
	return r.create(ctx, tx, like)
}

func (r *LikeRepository) create(ctx context.Context, db *gorm.DB, like *Like) error {
	return db.WithContext(ctx).Create(like).Error
}

// Delete 删除点赞
func (r *LikeRepository) Delete(ctx context.Context, videoID, accountID uint) (bool, error) {
	return r.delete(ctx, r.db, videoID, accountID)
}

// DeleteInTx 在事务内删除点赞
func (r *LikeRepository) DeleteInTx(ctx context.Context, tx *gorm.DB, videoID, accountID uint) (bool, error) {
	return r.delete(ctx, tx, videoID, accountID)
}

func (r *LikeRepository) delete(ctx context.Context, db *gorm.DB, videoID, accountID uint) (bool, error) {
	res := db.WithContext(ctx).
		Where("video_id = ? AND account_id = ?", videoID, accountID).
		Delete(&Like{})
	return res.RowsAffected > 0, res.Error
}

// IsLiked 是否已点赞
func (r *LikeRepository) IsLiked(ctx context.Context, videoID, accountID uint) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Like{}).
		Where("video_id = ? AND account_id = ?", videoID, accountID).
		Count(&n).Error
	return n > 0, err
}

// BatchIsLiked 批量查询是否点赞
func (r *LikeRepository) BatchIsLiked(ctx context.Context, videoIDs []uint, accountID uint) (map[uint]bool, error) {
	out := make(map[uint]bool)
	if len(videoIDs) == 0 || accountID == 0 {
		return out, nil
	}
	var list []Like
	err := r.db.WithContext(ctx).Where("video_id IN ? AND account_id = ?", videoIDs, accountID).
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	for _, l := range list {
		out[l.VideoID] = true
	}
	return out, nil
}

// ListLikedVideoIDs 获取用户点赞的视频 ID 列表
func (r *LikeRepository) ListLikedVideoIDs(ctx context.Context, accountID uint) ([]uint, error) {
	var list []Like
	err := r.db.WithContext(ctx).Where("account_id = ?", accountID).
		Order("created_at desc").Find(&list).Error
	if err != nil {
		return nil, err
	}
	ids := make([]uint, len(list))
	for i, l := range list {
		ids[i] = l.VideoID
	}
	return ids, nil
}

func isDupKey(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}
