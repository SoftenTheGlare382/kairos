package social

import (
	"context"

	"gorm.io/gorm"
)

// SocialRepository 关注关系仓储
type SocialRepository struct {
	db *gorm.DB
}

// NewSocialRepository 创建
func NewSocialRepository(db *gorm.DB) *SocialRepository {
	return &SocialRepository{db: db}
}

// Create 创建关注
// 若存在软删除的旧记录，先硬删除以避免唯一索引冲突（取关后再次关注）
func (r *SocialRepository) Create(ctx context.Context, follow *Follow) error {
	r.db.WithContext(ctx).Unscoped().
		Where("follower_id = ? AND following_id = ?", follow.FollowerID, follow.FollowingID).
		Delete(&Follow{})
	return r.db.WithContext(ctx).Create(follow).Error
}

// Delete 删除关注（硬删除，避免取关后再次关注时唯一索引冲突）
func (r *SocialRepository) Delete(ctx context.Context, followerID, followingID uint) (bool, error) {
	res := r.db.WithContext(ctx).Unscoped().
		Where("follower_id = ? AND following_id = ?", followerID, followingID).
		Delete(&Follow{})
	return res.RowsAffected > 0, res.Error
}

// IsFollowing 是否已关注
func (r *SocialRepository) IsFollowing(ctx context.Context, followerID, followingID uint) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Follow{}).
		Where("follower_id = ? AND following_id = ?", followerID, followingID).
		Count(&n).Error
	return n > 0, err
}

// ListFollowerIDs 某用户的粉丝 ID 列表（谁关注了该用户），支持分页
func (r *SocialRepository) ListFollowerIDs(ctx context.Context, userID uint, limit, offset int) ([]uint, error) {
	var list []Follow
	err := r.db.WithContext(ctx).Where("following_id = ?", userID).
		Order("created_at desc").Limit(limit).Offset(offset).Find(&list).Error
	if err != nil {
		return nil, err
	}
	ids := make([]uint, len(list))
	for i, f := range list {
		ids[i] = f.FollowerID
	}
	return ids, nil
}

// CountFollowers 某用户的粉丝总数
func (r *SocialRepository) CountFollowers(ctx context.Context, userID uint) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Follow{}).Where("following_id = ?", userID).Count(&n).Error
	return n, err
}

// ListFollowingIDs 某用户关注的 ID 列表（该用户关注了谁），支持分页
func (r *SocialRepository) ListFollowingIDs(ctx context.Context, userID uint, limit, offset int) ([]uint, error) {
	var list []Follow
	err := r.db.WithContext(ctx).Where("follower_id = ?", userID).
		Order("created_at desc").Limit(limit).Offset(offset).Find(&list).Error
	if err != nil {
		return nil, err
	}
	ids := make([]uint, len(list))
	for i, f := range list {
		ids[i] = f.FollowingID
	}
	return ids, nil
}

// CountFollowing 某用户关注的总数
func (r *SocialRepository) CountFollowing(ctx context.Context, userID uint) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Follow{}).Where("follower_id = ?", userID).Count(&n).Error
	return n, err
}

// ListFollowingIDsAll 某用户关注的 ID 列表（全量，供 Feed gRPC 等调用）
func (r *SocialRepository) ListFollowingIDsAll(ctx context.Context, userID uint) ([]uint, error) {
	var list []Follow
	err := r.db.WithContext(ctx).Where("follower_id = ?", userID).
		Order("created_at desc").Find(&list).Error
	if err != nil {
		return nil, err
	}
	ids := make([]uint, len(list))
	for i, f := range list {
		ids[i] = f.FollowingID
	}
	return ids, nil
}
