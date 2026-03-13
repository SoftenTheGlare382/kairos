package feed

import (
	"context"

	videopb "kairos/api/videopb"
)

// FeedVideo Feed 返回的视频项（含 is_liked）
type FeedVideo struct {
	ID             uint   `json:"id"`
	AuthorID       uint   `json:"author_id"`
	Username       string `json:"username"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	PlayURL        string `json:"play_url"`
	CoverURL       string `json:"cover_url"`
	CreatedAt      string `json:"created_at"`
	LikesCount     int64  `json:"likes_count"`
	Popularity     int64  `json:"popularity"`
	PlayCount      int64  `json:"play_count"`
	FavoritesCount int64  `json:"favorites_count"`
	IsLiked        bool   `json:"is_liked"`
}

// Service Feed 聚合服务（依赖 Video、Social gRPC）
type Service struct {
	videoCli  *VideoClient
	socialCli *SocialClient
}

// NewService 创建 Feed 服务
func NewService(videoCli *VideoClient, socialCli *SocialClient) *Service {
	return &Service{videoCli: videoCli, socialCli: socialCli}
}

// videoToFeed 将 gRPC VideoInfo 转为 FeedVideo，并填充 is_liked
func videoToFeed(v *videopb.VideoInfo, liked map[uint32]bool) FeedVideo {
	f := FeedVideo{
		ID:             uint(v.Id),
		AuthorID:       uint(v.AuthorId),
		Username:       v.Username,
		Title:          v.Title,
		Description:    v.Description,
		PlayURL:        v.PlayUrl,
		CoverURL:       v.CoverUrl,
		CreatedAt:      v.CreatedAt,
		LikesCount:     v.LikesCount,
		Popularity:     v.Popularity,
		PlayCount:      v.PlayCount,
		FavoritesCount: v.FavoritesCount,
	}
	if liked != nil {
		f.IsLiked = liked[uint32(v.Id)]
	}
	return f
}

// ListLatest 最新视频流
func (s *Service) ListLatest(ctx context.Context, limit, offset int32, accountID uint) ([]FeedVideo, error) {
	videos, err := s.videoCli.ListLatest(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	return s.enrichWithLiked(ctx, videos, accountID)
}

// ListByFollowing 关注流：先取关注用户 ID，再按作者查视频
func (s *Service) ListByFollowing(ctx context.Context, limit, offset int32, accountID uint) ([]FeedVideo, error) {
	if accountID == 0 {
		return nil, nil
	}
	followingIDs, err := s.socialCli.GetFollowingIDs(ctx, uint32(accountID))
	if err != nil {
		return nil, err
	}
	if len(followingIDs) == 0 {
		return []FeedVideo{}, nil
	}
	videos, err := s.videoCli.ListByAuthorIDs(ctx, followingIDs)
	if err != nil {
		return nil, err
	}
	// 简单分页：按返回顺序取 [offset:offset+limit]
	start := int(offset)
	if start >= len(videos) {
		return []FeedVideo{}, nil
	}
	end := start + int(limit)
	if end > len(videos) {
		end = len(videos)
	}
	page := videos[start:end]
	return s.enrichWithLiked(ctx, page, accountID)
}

// ListByPopularity 热度流
func (s *Service) ListByPopularity(ctx context.Context, limit, offset int32, accountID uint) ([]FeedVideo, error) {
	videos, err := s.videoCli.ListByPopularity(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	return s.enrichWithLiked(ctx, videos, accountID)
}

// enrichWithLiked 批量查询点赞状态并组装 FeedVideo 列表
func (s *Service) enrichWithLiked(ctx context.Context, videos []*videopb.VideoInfo, accountID uint) ([]FeedVideo, error) {
	if len(videos) == 0 {
		return nil, nil
	}
	ids := make([]uint32, len(videos))
	for i, v := range videos {
		ids[i] = v.Id
	}
	var liked map[uint32]bool
	if accountID > 0 {
		var err error
		liked, err = s.videoCli.BatchIsLiked(ctx, ids, uint32(accountID))
		if err != nil {
			return nil, err
		}
	}
	result := make([]FeedVideo, len(videos))
	for i, v := range videos {
		result[i] = videoToFeed(v, liked)
	}
	return result, nil
}
