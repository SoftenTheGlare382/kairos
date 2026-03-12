package account

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	accountpb "kairos/api/accountpb"
)

// Ensure grpcAccountServer implements accountpb.AccountServiceServer
var _ accountpb.AccountServiceServer = (*grpcAccountServer)(nil)

// grpcAccountServer 实现 Account gRPC 服务
type grpcAccountServer struct {
	accountpb.UnimplementedAccountServiceServer
	svc *Service
}

// NewGrpcAccountServer 创建 gRPC 服务实现
func NewGrpcAccountServer(svc *Service) *grpcAccountServer {
	return &grpcAccountServer{svc: svc}
}

// FindByID 按 ID 获取用户
func (s *grpcAccountServer) FindByID(ctx context.Context, req *accountpb.FindByIDRequest) (*accountpb.FindByIDResponse, error) {
	if req == nil || req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	account, err := s.svc.FindByID(ctx, uint(req.Id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "account not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	if account == nil {
		return nil, status.Error(codes.NotFound, "account not found")
	}
	return &accountpb.FindByIDResponse{
		Id:       uint32(account.ID),
		Username: account.Username,
	}, nil
}
