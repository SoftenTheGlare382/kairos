package social

import (
	"context"
	"errors"

	"kairos/pkg/grpc"
)

// SocialPublisher 关注事件发布接口（可选，供 Worker 消费）
type SocialPublisher interface {
	PublishSocial(followerID, followingID uint, action string)
}

// SocialService 关注关系服务
type SocialService struct {
	repo       *SocialRepository
	accountCli *grpc.AccountClient
	publisher  SocialPublisher
}

// NewSocialService 创建，publisher 可为 nil
func NewSocialService(repo *SocialRepository, accountCli *grpc.AccountClient, publisher SocialPublisher) *SocialService {
	return &SocialService{repo: repo, accountCli: accountCli, publisher: publisher}
}

// Follow 关注某用户
func (s *SocialService) Follow(ctx context.Context, followerID, followingID uint) error {
	if followerID == 0 || followingID == 0 {
		return errors.New("follower_id and following_id are required")
	}
	if followerID == followingID {
		return errors.New("cannot follow yourself")
	}
	// 校验被关注者是否存在
	users, err := s.accountCli.GetByIDs(ctx, []uint{followingID})
	if err != nil {
		return err
	}
	if len(users) == 0 {
		return errors.New("user not found")
	}
	ok, err := s.repo.IsFollowing(ctx, followerID, followingID)
	if err != nil {
		return err
	}
	if ok {
		return errors.New("already following")
	}
	if err := s.repo.Create(ctx, &Follow{FollowerID: followerID, FollowingID: followingID}); err != nil {
		return err
	}
	if s.publisher != nil {
		s.publisher.PublishSocial(followerID, followingID, "follow")
	}
	return nil
}

// Unfollow 取关某用户
func (s *SocialService) Unfollow(ctx context.Context, followerID, followingID uint) error {
	if followerID == 0 || followingID == 0 {
		return errors.New("follower_id and following_id are required")
	}
	deleted, err := s.repo.Delete(ctx, followerID, followingID)
	if err != nil {
		return err
	}
	if !deleted {
		return errors.New("not following")
	}
	if s.publisher != nil {
		s.publisher.PublishSocial(followerID, followingID, "unfollow")
	}
	return nil
}

// UserWithID 带 ID 的用户信息（粉丝/关注列表项）
type UserWithID struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

// ListResult 分页列表结果
type ListResult struct {
	List    []UserWithID `json:"list"`
	Total   int64       `json:"total"`
	HasMore bool        `json:"has_more"`
}

// ListFollowers 某用户的粉丝列表（补全用户信息，分页）
func (s *SocialService) ListFollowers(ctx context.Context, userID uint, page, pageSize int) (*ListResult, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	total, err := s.repo.CountFollowers(ctx, userID)
	if err != nil {
		return nil, err
	}
	ids, err := s.repo.ListFollowerIDs(ctx, userID, pageSize, offset)
	if err != nil {
		return nil, err
	}
	hasMore := int64(offset+len(ids)) < total
	if len(ids) == 0 {
		return &ListResult{List: nil, Total: total, HasMore: false}, nil
	}
	users, err := s.accountCli.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	userMap := make(map[uint]grpc.UserInfo)
	for _, u := range users {
		userMap[u.ID] = u
	}
	result := make([]UserWithID, 0, len(ids))
	for _, id := range ids {
		if u, ok := userMap[id]; ok {
			result = append(result, UserWithID{ID: u.ID, Username: u.Username})
		}
	}
	return &ListResult{List: result, Total: total, HasMore: hasMore}, nil
}

// ListFollowing 某用户的关注列表（补全用户信息，分页）
func (s *SocialService) ListFollowing(ctx context.Context, userID uint, page, pageSize int) (*ListResult, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	total, err := s.repo.CountFollowing(ctx, userID)
	if err != nil {
		return nil, err
	}
	ids, err := s.repo.ListFollowingIDs(ctx, userID, pageSize, offset)
	if err != nil {
		return nil, err
	}
	hasMore := int64(offset+len(ids)) < total
	if len(ids) == 0 {
		return &ListResult{List: nil, Total: total, HasMore: false}, nil
	}
	users, err := s.accountCli.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	userMap := make(map[uint]grpc.UserInfo)
	for _, u := range users {
		userMap[u.ID] = u
	}
	result := make([]UserWithID, 0, len(ids))
	for _, id := range ids {
		if u, ok := userMap[id]; ok {
			result = append(result, UserWithID{ID: u.ID, Username: u.Username})
		}
	}
	return &ListResult{List: result, Total: total, HasMore: hasMore}, nil
}
