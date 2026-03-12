package social

import (
	"time"

	"gorm.io/gorm"
)

// Follow 关注关系
// follower_id: 关注者（谁点了关注）
// following_id: 被关注者（被谁关注）
// A 关注 B => follower_id=A, following_id=B
type Follow struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	FollowerID  uint           `gorm:"uniqueIndex:idx_follow_follower_following;not null" json:"follower_id"`  // 关注者
	FollowingID uint           `gorm:"uniqueIndex:idx_follow_follower_following;not null" json:"following_id"` // 被关注者
	CreatedAt   time.Time      `json:"created_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
