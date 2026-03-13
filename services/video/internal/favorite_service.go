package video

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// PopularityPublisher 热度事件发布接口（可选）
type PopularityPublisher interface {
	PublishPopularity(videoID uint, delta int64)
}

// FavoriteService 收藏服务
type FavoriteService struct {
	db           *gorm.DB
	favoriteRepo *FavoriteRepository
	videoRepo    *VideoRepository
	publisher    PopularityPublisher
}

// NewFavoriteService 创建，publisher 可为 nil
func NewFavoriteService(db *gorm.DB, favoriteRepo *FavoriteRepository, videoRepo *VideoRepository, publisher PopularityPublisher) *FavoriteService {
	return &FavoriteService{db: db, favoriteRepo: favoriteRepo, videoRepo: videoRepo, publisher: publisher}
}

// Favorite 收藏视频
func (s *FavoriteService) Favorite(ctx context.Context, accountID, videoID uint) error {
	if accountID == 0 || videoID == 0 {
		return errors.New("account_id and video_id are required")
	}
	ok, err := s.videoRepo.IsExist(ctx, videoID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("video not found")
	}
	favorited, err := s.favoriteRepo.IsFavorited(ctx, videoID, accountID)
	if err != nil {
		return err
	}
	if favorited {
		return errors.New("already favorited")
	}
	f := &Favorite{AccountID: accountID, VideoID: videoID, CreatedAt: time.Now()}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(f).Error; err != nil {
			if isFavoriteDupKey(err) {
				return errors.New("already favorited")
			}
			return err
		}
		if err := s.videoRepo.UpdateFavoritesCountInTx(ctx, tx, videoID, 1); err != nil {
			return err
		}
		return s.videoRepo.UpdatePopularityInTx(ctx, tx, videoID, PopularityWeightFavorite)
	})
	if err == nil && s.publisher != nil {
		s.publisher.PublishPopularity(videoID, PopularityWeightFavorite)
	}
	return err
}

// Unfavorite 取消收藏
func (s *FavoriteService) Unfavorite(ctx context.Context, accountID, videoID uint) error {
	if accountID == 0 || videoID == 0 {
		return errors.New("account_id and video_id are required")
	}
	ok, err := s.videoRepo.IsExist(ctx, videoID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("video not found")
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("video_id = ? AND account_id = ?", videoID, accountID).Delete(&Favorite{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("not favorited")
		}
		if err := s.videoRepo.UpdateFavoritesCountInTx(ctx, tx, videoID, -1); err != nil {
			return err
		}
		return s.videoRepo.UpdatePopularityInTx(ctx, tx, videoID, -PopularityWeightFavorite)
	})
	if err == nil && s.publisher != nil {
		s.publisher.PublishPopularity(videoID, -PopularityWeightFavorite)
	}
	return err
}

// IsFavorited 是否已收藏
func (s *FavoriteService) IsFavorited(ctx context.Context, videoID, accountID uint) (bool, error) {
	return s.favoriteRepo.IsFavorited(ctx, videoID, accountID)
}

// ListMyFavoritedVideos 我收藏的视频列表
func (s *FavoriteService) ListMyFavoritedVideos(ctx context.Context, accountID uint) ([]Video, error) {
	ids, err := s.favoriteRepo.ListFavoritedVideoIDs(ctx, accountID)
	if err != nil || len(ids) == 0 {
		return nil, err
	}
	return s.videoRepo.GetByIDs(ctx, ids)
}
