package video

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	videopb "kairos/api/videopb"
	"kairos/pkg/redis"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	RedisKeyHot       = "feed:hot"
	RedisKeyLatestIDs = "feed:latest:ids"
	latestIDsCacheTTL = 60 * time.Second
)

var _ videopb.VideoServiceServer = (*grpcVideoServer)(nil)

// grpcVideoServer 实现 Video gRPC 服务（供 Feed 调用）
type grpcVideoServer struct {
	videopb.UnimplementedVideoServiceServer
	videoRepo *VideoRepository
	likeRepo  *LikeRepository
	rdb       *redis.Client // 可选，非 nil 时 ListByPopularity 优先读 Redis
}

// NewGrpcVideoServer 创建 gRPC 服务实现，rdb 可为 nil（仅走 MySQL）
func NewGrpcVideoServer(videoRepo *VideoRepository, likeRepo *LikeRepository, rdb *redis.Client) *grpcVideoServer {
	return &grpcVideoServer{videoRepo: videoRepo, likeRepo: likeRepo, rdb: rdb}
}

func videoToPB(v *Video) *videopb.VideoInfo {
	if v == nil {
		return nil
	}
	return &videopb.VideoInfo{
		Id:             uint32(v.ID),
		AuthorId:       uint32(v.AuthorID),
		Username:       v.Username,
		Title:          v.Title,
		Description:    v.Description,
		PlayUrl:        v.PlayURL,
		CoverUrl:       v.CoverURL,
		CreatedAt:      v.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		LikesCount:     v.LikesCount,
		Popularity:     v.Popularity,
		PlayCount:      v.PlayCount,
		FavoritesCount: v.FavoritesCount,
	}
}

// ListLatest 最新视频列表（offset=0,limit<=100 时缓存 ID 列表，Publish 时失效）
func (s *grpcVideoServer) ListLatest(ctx context.Context, req *videopb.ListLatestRequest) (*videopb.ListLatestResponse, error) {
	limit, offset := 20, 0
	if req != nil {
		if req.Limit > 0 {
			limit = int(req.Limit)
		}
		if req.Offset > 0 {
			offset = int(req.Offset)
		}
	}
	var list []Video
	if s.rdb != nil && offset == 0 && limit <= 100 {
		b, err := s.rdb.GetBytes(ctx, RedisKeyLatestIDs)
		if err == nil && len(b) > 0 {
			var ids []uint
			if json.Unmarshal(b, &ids) == nil && len(ids) >= limit {
				page := ids[:limit]
				var err2 error
				list, err2 = s.videoRepo.GetByIDs(ctx, page)
				if err2 == nil {
					list = orderByIDs(list, page)
					log.Printf("cache hit: feed:latest:ids limit=%d", limit)
				}
			}
		}
	}
	if len(list) == 0 {
		var err error
		list, err = s.videoRepo.ListLatest(ctx, limit, offset)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		if s.rdb != nil && offset == 0 && len(list) > 0 {
			ids := make([]uint, len(list))
			for i := range list {
				ids[i] = list[i].ID
			}
			if b, err := json.Marshal(ids); err == nil {
				s.rdb.SetBytes(ctx, RedisKeyLatestIDs, b, redis.TTLWithJitter(latestIDsCacheTTL, 0.2))
			}
		}
	}
	videos := make([]*videopb.VideoInfo, len(list))
	for i := range list {
		videos[i] = videoToPB(&list[i])
	}
	return &videopb.ListLatestResponse{Videos: videos}, nil
}

// ListByAuthorIDs 按多作者获取视频
func (s *grpcVideoServer) ListByAuthorIDs(ctx context.Context, req *videopb.ListByAuthorIDsRequest) (*videopb.ListByAuthorIDsResponse, error) {
	if req == nil || len(req.AuthorIds) == 0 {
		return &videopb.ListByAuthorIDsResponse{Videos: nil}, nil
	}
	ids := make([]uint, len(req.AuthorIds))
	for i, id := range req.AuthorIds {
		ids[i] = uint(id)
	}
	list, err := s.videoRepo.ListByAuthorIDs(ctx, ids)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	videos := make([]*videopb.VideoInfo, len(list))
	for i := range list {
		videos[i] = videoToPB(&list[i])
	}
	return &videopb.ListByAuthorIDsResponse{Videos: videos}, nil
}

// GetByIDs 按 ID 列表获取视频
func (s *grpcVideoServer) GetByIDs(ctx context.Context, req *videopb.GetByIDsRequest) (*videopb.GetByIDsResponse, error) {
	if req == nil || len(req.Ids) == 0 {
		return &videopb.GetByIDsResponse{Videos: nil}, nil
	}
	ids := make([]uint, len(req.Ids))
	for i, id := range req.Ids {
		ids[i] = uint(id)
	}
	list, err := s.videoRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	videos := make([]*videopb.VideoInfo, len(list))
	for i := range list {
		videos[i] = videoToPB(&list[i])
	}
	return &videopb.GetByIDsResponse{Videos: videos}, nil
}

// ListByPopularity 按热度排序（优先从 Redis ZSET 取 ID，再查 MySQL 详情；Redis 为空时回退 MySQL）
func (s *grpcVideoServer) ListByPopularity(ctx context.Context, req *videopb.ListByPopularityRequest) (*videopb.ListByPopularityResponse, error) {
	limit, offset := 20, 0
	if req != nil {
		if req.Limit > 0 {
			limit = int(req.Limit)
		}
		if req.Offset > 0 {
			offset = int(req.Offset)
		}
	}
	var list []Video
	if s.rdb != nil {
		members, err := s.rdb.ZRevRange(ctx, RedisKeyHot, int64(offset), int64(offset+limit-1))
		if err == nil && len(members) > 0 {
			ids := make([]uint, 0, len(members))
			for _, m := range members {
				id, _ := strconv.ParseUint(m, 10, 64)
				if id > 0 {
					ids = append(ids, uint(id))
				}
			}
			if len(ids) > 0 {
				var err2 error
				list, err2 = s.videoRepo.GetByIDs(ctx, ids)
				if err2 != nil {
					list = nil
				} else {
					list = orderByIDs(list, ids)
					log.Printf("cache hit: feed:hot offset=%d limit=%d", offset, limit)
				}
			}
		}
	}
	if len(list) == 0 {
		var err error
		list, err = s.videoRepo.ListByPopularity(ctx, limit, offset)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}
	videos := make([]*videopb.VideoInfo, len(list))
	for i := range list {
		videos[i] = videoToPB(&list[i])
	}
	return &videopb.ListByPopularityResponse{Videos: videos}, nil
}

func orderByIDs(list []Video, ids []uint) []Video {
	byID := make(map[uint]Video)
	for _, v := range list {
		byID[v.ID] = v
	}
	out := make([]Video, 0, len(ids))
	for _, id := range ids {
		if v, ok := byID[id]; ok {
			out = append(out, v)
		}
	}
	return out
}

// BatchIsLiked 批量是否点赞
func (s *grpcVideoServer) BatchIsLiked(ctx context.Context, req *videopb.BatchIsLikedRequest) (*videopb.BatchIsLikedResponse, error) {
	if req == nil || len(req.VideoIds) == 0 || req.AccountId == 0 {
		return &videopb.BatchIsLikedResponse{Liked: nil}, nil
	}
	videoIDs := make([]uint, len(req.VideoIds))
	for i, id := range req.VideoIds {
		videoIDs[i] = uint(id)
	}
	m, err := s.likeRepo.BatchIsLiked(ctx, videoIDs, uint(req.AccountId))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	liked := make(map[uint32]bool)
	for id, v := range m {
		liked[uint32(id)] = v
	}
	return &videopb.BatchIsLikedResponse{Liked: liked}, nil
}
