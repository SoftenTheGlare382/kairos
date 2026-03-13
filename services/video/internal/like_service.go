package video

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"kairos/pkg/redis"

	"gorm.io/gorm"
)

const likedSetInitTTL = 24 * time.Hour

func likedSetKey(accountID uint) string { return fmt.Sprintf("user:%d:liked", accountID) }
func likedInitKey(accountID uint) string { return fmt.Sprintf("user:%d:liked:init", accountID) }

// LikePublisher 点赞事件发布接口（可选，供 Worker 消费）
type LikePublisher interface {
	PublishLike(videoID, accountID uint, delta int64)
	PublishPopularity(videoID uint, delta int64)
}

// LikeService 点赞服务
type LikeService struct {
	db        *gorm.DB
	likeRepo  *LikeRepository
	videoRepo *VideoRepository
	publisher LikePublisher
	rdb       *redis.Client // 可选，非 nil 时 BatchIsLiked 读缓存，Like/Unlike 写穿
}

// NewLikeService 创建，publisher、rdb 可为 nil
func NewLikeService(db *gorm.DB, likeRepo *LikeRepository, videoRepo *VideoRepository, publisher LikePublisher, rdb *redis.Client) *LikeService {
	return &LikeService{db: db, likeRepo: likeRepo, videoRepo: videoRepo, publisher: publisher, rdb: rdb}
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
		if err := s.videoRepo.UpdateLikesCountInTx(ctx, tx, like.VideoID, 1); err != nil {
			return err
		}
		return s.videoRepo.UpdatePopularityInTx(ctx, tx, like.VideoID, PopularityWeightLike)
	})
	if err == nil {
		if s.rdb != nil {
			s.rdb.SAdd(ctx, likedSetKey(like.AccountID), strconv.FormatUint(uint64(like.VideoID), 10))
			s.rdb.SetBytes(ctx, likedInitKey(like.AccountID), []byte("1"), redis.TTLWithJitter(likedSetInitTTL, 0.2))
		}
		if s.publisher != nil {
			s.publisher.PublishLike(like.VideoID, like.AccountID, 1)
			s.publisher.PublishPopularity(like.VideoID, PopularityWeightLike)
		}
	}
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
		if err := s.videoRepo.UpdateLikesCountInTx(ctx, tx, like.VideoID, -1); err != nil {
			return err
		}
		return s.videoRepo.UpdatePopularityInTx(ctx, tx, like.VideoID, -PopularityWeightLike)
	})
	if err == nil {
		if s.rdb != nil {
			s.rdb.SRem(ctx, likedSetKey(like.AccountID), strconv.FormatUint(uint64(like.VideoID), 10))
		}
		if s.publisher != nil {
			s.publisher.PublishLike(like.VideoID, like.AccountID, -1)
			s.publisher.PublishPopularity(like.VideoID, -PopularityWeightLike)
		}
	}
	return err
}

// IsLiked 是否已点赞（有 rdb 时复用 user:{id}:liked Set，与 BatchIsLiked 一致）
func (s *LikeService) IsLiked(ctx context.Context, videoID, accountID uint) (bool, error) {
	if accountID == 0 || videoID == 0 {
		return false, nil
	}
	if s.rdb != nil {
		s.ensureLikedSetInitialized(ctx, accountID)
		ok, err := s.rdb.SIsMember(ctx, likedSetKey(accountID), strconv.FormatUint(uint64(videoID), 10))
		if err == nil {
			log.Printf("cache hit: user:%d:liked (isLiked video=%d)", accountID, videoID)
			return ok, nil
		}
	}
	return s.likeRepo.IsLiked(ctx, videoID, accountID)
}

func (s *LikeService) ensureLikedSetInitialized(ctx context.Context, accountID uint) {
	_, err := s.rdb.GetBytes(ctx, likedInitKey(accountID))
	if err != nil {
		ids, err2 := s.likeRepo.ListLikedVideoIDs(ctx, accountID)
		if err2 != nil {
			return
		}
		setKey := likedSetKey(accountID)
		for _, id := range ids {
			s.rdb.SAdd(ctx, setKey, strconv.FormatUint(uint64(id), 10))
		}
		s.rdb.SetBytes(ctx, likedInitKey(accountID), []byte("1"), redis.TTLWithJitter(likedSetInitTTL, 0.2))
	}
}

// BatchIsLiked 批量是否点赞（有 rdb 时优先读 Redis Set，未初始化则从 DB 回填）
func (s *LikeService) BatchIsLiked(ctx context.Context, videoIDs []uint, accountID uint) (map[uint]bool, error) {
	if len(videoIDs) == 0 || accountID == 0 {
		return map[uint]bool{}, nil
	}
	if s.rdb != nil {
		s.ensureLikedSetInitialized(ctx, accountID)
		members, err := s.rdb.SMembers(ctx, likedSetKey(accountID))
		if err == nil {
			set := make(map[string]bool)
			for _, m := range members {
				set[m] = true
			}
			out := make(map[uint]bool)
			for _, vid := range videoIDs {
				out[vid] = set[strconv.FormatUint(uint64(vid), 10)]
			}
			log.Printf("cache hit: user:%d:liked (batch %d videos)", accountID, len(videoIDs))
			return out, nil
		}
	}
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
