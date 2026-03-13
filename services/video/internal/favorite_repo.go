package video

import (
	"context"
	"errors"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// FavoriteRepository 收藏仓储
type FavoriteRepository struct {
	db *gorm.DB
}

// NewFavoriteRepository 创建
func NewFavoriteRepository(db *gorm.DB) *FavoriteRepository {
	return &FavoriteRepository{db: db}
}

// Create 添加收藏
func (r *FavoriteRepository) Create(ctx context.Context, f *Favorite) error {
	return r.db.WithContext(ctx).Create(f).Error
}

// Delete 取消收藏
func (r *FavoriteRepository) Delete(ctx context.Context, videoID, accountID uint) (bool, error) {
	res := r.db.WithContext(ctx).
		Where("video_id = ? AND account_id = ?", videoID, accountID).
		Delete(&Favorite{})
	return res.RowsAffected > 0, res.Error
}

// IsFavorited 是否已收藏
func (r *FavoriteRepository) IsFavorited(ctx context.Context, videoID, accountID uint) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Favorite{}).
		Where("video_id = ? AND account_id = ?", videoID, accountID).
		Count(&n).Error
	return n > 0, err
}

// ListFavoritedVideoIDs 用户收藏的视频 ID 列表
func (r *FavoriteRepository) ListFavoritedVideoIDs(ctx context.Context, accountID uint) ([]uint, error) {
	var list []Favorite
	err := r.db.WithContext(ctx).Where("account_id = ?", accountID).
		Order("created_at desc").Find(&list).Error
	if err != nil {
		return nil, err
	}
	ids := make([]uint, len(list))
	for i, f := range list {
		ids[i] = f.VideoID
	}
	return ids, nil
}

func isFavoriteDupKey(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}
