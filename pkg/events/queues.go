package events

// 队列名称（与 Worker 约定一致）
const (
	QueueLike        = "like.events"
	QueueComment     = "comment.events"
	QueueSocial      = "social.events"
	QueuePopularity  = "video.popularity.events"
	QueuePlay        = "video.play.events"
)

// LikeEvent 点赞事件
type LikeEvent struct {
	VideoID   uint  `json:"video_id"`
	AccountID uint  `json:"account_id"`
	Delta     int64 `json:"delta"` // 1=点赞 -1=取消
}

// CommentEvent 评论事件
type CommentEvent struct {
	VideoID uint  `json:"video_id"`
	Delta   int64 `json:"delta"` // 1=发布 -1=删除
}

// SocialEvent 关注事件
type SocialEvent struct {
	FollowerID  uint   `json:"follower_id"`
	FollowingID uint   `json:"following_id"`
	Action      string `json:"action"` // follow | unfollow
}

// PopularityEvent 热度事件
type PopularityEvent struct {
	VideoID uint  `json:"video_id"`
	Delta   int64 `json:"delta"`
}

// PlayEvent 播放记录事件（Worker 异步落库）
type PlayEvent struct {
	AccountID uint `json:"account_id"`
	VideoID   uint `json:"video_id"`
}
