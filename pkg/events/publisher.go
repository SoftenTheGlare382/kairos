package events

import (
	"kairos/pkg/rabbitmq"
)

// Publisher 事件发布者（Video、Social 使用，可选）
type Publisher struct {
	mq *rabbitmq.Client
}

// NewPublisher 创建发布者，mq 为 nil 时所有 Publish 均为空操作
func NewPublisher(mq *rabbitmq.Client) *Publisher {
	return &Publisher{mq: mq}
}

// PublishLike 发布点赞事件
func (p *Publisher) PublishLike(videoID, accountID uint, delta int64) {
	if p == nil || p.mq == nil {
		return
	}
	_ = p.mq.Publish(QueueLike, LikeEvent{VideoID: videoID, AccountID: accountID, Delta: delta})
}

// PublishComment 发布评论事件
func (p *Publisher) PublishComment(videoID uint, delta int64) {
	if p == nil || p.mq == nil {
		return
	}
	_ = p.mq.Publish(QueueComment, CommentEvent{VideoID: videoID, Delta: delta})
}

// PublishPopularity 发布热度事件
func (p *Publisher) PublishPopularity(videoID uint, delta int64) {
	if p == nil || p.mq == nil {
		return
	}
	_ = p.mq.Publish(QueuePopularity, PopularityEvent{VideoID: videoID, Delta: delta})
}

// PublishSocial 发布关注事件
func (p *Publisher) PublishSocial(followerID, followingID uint, action string) {
	if p == nil || p.mq == nil {
		return
	}
	_ = p.mq.Publish(QueueSocial, SocialEvent{FollowerID: followerID, FollowingID: followingID, Action: action})
}

// PublishPlay 发布播放事件（Worker 异步写入 play_records 并更新 play_count/popularity）
func (p *Publisher) PublishPlay(accountID, videoID uint) {
	if p == nil || p.mq == nil {
		return
	}
	_ = p.mq.Publish(QueuePlay, PlayEvent{AccountID: accountID, VideoID: videoID})
}
