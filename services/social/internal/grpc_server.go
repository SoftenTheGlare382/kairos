package social

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	socialpb "kairos/api/socialpb"
)

// Ensure grpcSocialServer implements socialpb.SocialServiceServer
var _ socialpb.SocialServiceServer = (*grpcSocialServer)(nil)

// grpcSocialServer 实现 Social gRPC 服务（供 Feed 调用）
type grpcSocialServer struct {
	socialpb.UnimplementedSocialServiceServer
	repo *SocialRepository
}

// NewGrpcSocialServer 创建 gRPC 服务实现
func NewGrpcSocialServer(repo *SocialRepository) *grpcSocialServer {
	return &grpcSocialServer{repo: repo}
}

// GetFollowingIDs 获取某用户关注的用户 ID 列表
func (s *grpcSocialServer) GetFollowingIDs(ctx context.Context, req *socialpb.GetFollowingIDsRequest) (*socialpb.GetFollowingIDsResponse, error) {
	if req == nil || req.FollowerId == 0 {
		return &socialpb.GetFollowingIDsResponse{FollowingIds: nil}, nil
	}
	ids, err := s.repo.ListFollowingIDsAll(ctx, uint(req.FollowerId))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	followingIds := make([]uint32, len(ids))
	for i, id := range ids {
		followingIds[i] = uint32(id)
	}
	return &socialpb.GetFollowingIDsResponse{FollowingIds: followingIds}, nil
}

// IsMutualFollow 是否互相关注（供 IM 校验私聊权限）
func (s *grpcSocialServer) IsMutualFollow(ctx context.Context, req *socialpb.IsMutualFollowRequest) (*socialpb.IsMutualFollowResponse, error) {
	if req == nil || req.UserA == 0 || req.UserB == 0 {
		return &socialpb.IsMutualFollowResponse{Mutual: false}, nil
	}
	mutual, err := s.repo.IsMutualFollow(ctx, uint(req.UserA), uint(req.UserB))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &socialpb.IsMutualFollowResponse{Mutual: mutual}, nil
}
