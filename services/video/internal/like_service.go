package video

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// LikeService 点赞服务
type LikeService struct {
	db        *gorm.DB
	likeRepo  *LikeRepository
	videoRepo *VideoRepository
}

// NewLikeService 创建
func NewLikeService(db *gorm.DB, likeRepo *LikeRepository, videoRepo *VideoRepository) *LikeService {
	return &LikeService{db: db, likeRepo: likeRepo, videoRepo: videoRepo}
}

// Like 点赞（Create + Update 在同一事务内，保证一致性）
func (s *LikeService) Like(ctx context.Context, like *Like) error {
	if like == nil || like.VideoID == 0 || like.AccountID == 0 {
		return errors.New("video_id and account_id are required")
	}
	ok, err := s.videoRepo.IsExist(ctx, like.VideoID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("video not found")
	}
	liked, err := s.likeRepo.IsLiked(ctx, like.VideoID, like.AccountID)
	if err != nil {
		return err
	}
	if liked {
		return errors.New("user has liked this video")
	}
	like.CreatedAt = time.Now()
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.likeRepo.CreateInTx(ctx, tx, like); err != nil {
			if isDupKey(err) {
				return errors.New("user has liked this video")
			}
			return err
		}
		return s.videoRepo.UpdateLikesCountInTx(ctx, tx, like.VideoID, 1)
	})
	return err
}

// Unlike 取消点赞（Delete + Update 在同一事务内，保证一致性）
func (s *LikeService) Unlike(ctx context.Context, like *Like) error {
	if like == nil || like.VideoID == 0 || like.AccountID == 0 {
		return errors.New("video_id and account_id are required")
	}
	ok, err := s.videoRepo.IsExist(ctx, like.VideoID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("video not found")
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		deleted, err := s.likeRepo.DeleteInTx(ctx, tx, like.VideoID, like.AccountID)
		if err != nil {
			return err
		}
		if !deleted {
			return errors.New("user has not liked this video")
		}
		return s.videoRepo.UpdateLikesCountInTx(ctx, tx, like.VideoID, -1)
	})
	return err
}

// IsLiked 是否已点赞
func (s *LikeService) IsLiked(ctx context.Context, videoID, accountID uint) (bool, error) {
	return s.likeRepo.IsLiked(ctx, videoID, accountID)
}

// BatchIsLiked 批量是否点赞
func (s *LikeService) BatchIsLiked(ctx context.Context, videoIDs []uint, accountID uint) (map[uint]bool, error) {
	return s.likeRepo.BatchIsLiked(ctx, videoIDs, accountID)
}

// ListLikedVideos 用户点赞的视频列表
func (s *LikeService) ListLikedVideos(ctx context.Context, accountID uint) ([]Video, error) {
	ids, err := s.likeRepo.ListLikedVideoIDs(ctx, accountID)
	if err != nil || len(ids) == 0 {
		return nil, err
	}
	return s.videoRepo.GetByIDs(ctx, ids)
}
