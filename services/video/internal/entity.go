package video

import (
	"time"

	"gorm.io/gorm"
)

// Video 视频实体
type Video struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	AuthorID    uint           `gorm:"index;not null" json:"author_id"`
	Username    string         `gorm:"size:191;not null" json:"username"`
	Title       string         `gorm:"size:255;not null" json:"title"`
	Description string         `gorm:"size:500" json:"description,omitempty"`
	PlayURL     string         `gorm:"size:512;not null" json:"play_url"`
	CoverURL    string         `gorm:"size:512;not null" json:"cover_url"`
	CreatedAt   time.Time      `gorm:"column:create_at;autoCreateTime" json:"created_at"`
	LikesCount  int64          `gorm:"column:likes_count;not null;default:0" json:"likes_count"`
	Popularity  int64          `gorm:"column:popularity;not null;default:0" json:"popularity"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// Like 点赞
type Like struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	VideoID   uint      `gorm:"uniqueIndex:idx_like_video_account;not null" json:"video_id"`
	AccountID uint      `gorm:"uniqueIndex:idx_like_video_account;not null" json:"account_id"`
	CreatedAt time.Time `json:"created_at"`
}

// Comment 评论
type Comment struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Username  string         `gorm:"size:191;index" json:"username"`
	VideoID   uint           `gorm:"index" json:"video_id"`
	AuthorID  uint           `gorm:"index" json:"author_id"`
	Content   string         `gorm:"type:text" json:"content"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
