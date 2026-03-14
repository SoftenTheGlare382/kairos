package im

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"kairos/pkg/grpc"
)

var (
	ErrMutualFollowRequired = errors.New("only mutually followed users can chat freely; you can send one intro message")
	ErrIntroLimitExceeded   = errors.New("you have already sent your one intro message; mutual follow required for more")
)

// SocialClient 用于校验互关（可选，nil 时不校验）
type SocialClient interface {
	IsMutualFollow(ctx context.Context, userA, userB uint) (bool, error)
}

// MessagePublisher 消息持久化发布者（发布到 MQ，由 Consumer 异步落库）
type MessagePublisher interface {
	PublishMessage(payload MessagePersistPayload) error
}

// ConversationWithUnread 带未读数的会话（ListConversations 返回）
type ConversationWithUnread struct {
	Conversation
	UnreadCount int64 `json:"unread_count"`
}

// IMService IM 服务
type IMService struct {
	convRepo      *ConversationRepository
	msgRepo       *MessageRepository
	readRepo      *ReadRepository
	accountCli    *grpc.AccountClient
	socialCli     SocialClient
	hub           *Hub // WebSocket 广播
	mqPublisher   MessagePublisher
	searchClient  MessageSearchClient // 可选，Meilisearch 模糊搜索
}

// NewIMService 创建
func NewIMService(convRepo *ConversationRepository, msgRepo *MessageRepository, readRepo *ReadRepository, accountCli *grpc.AccountClient, socialCli SocialClient, hub *Hub, mqPublisher MessagePublisher, searchClient MessageSearchClient) *IMService {
	return &IMService{
		convRepo:     convRepo,
		msgRepo:      msgRepo,
		readRepo:     readRepo,
		accountCli:   accountCli,
		socialCli:    socialCli,
		hub:          hub,
		mqPublisher:  mqPublisher,
		searchClient: searchClient,
	}
}

// SendMessage 发送消息（互关无障碍；非互关仅能发一条「介绍」消息）
// 异步落库：校验+GetOrCreate 后发布 MQ、推送 WebSocket、立即返回；Consumer 异步写 DB
func (s *IMService) SendMessage(ctx context.Context, senderID, receiverID uint, content string) (*Message, error) {
	if content == "" {
		return nil, errors.New("content is required")
	}
	if senderID == receiverID {
		return nil, errors.New("cannot send to yourself")
	}

	// 校验互关
	mutual := true
	if s.socialCli != nil {
		var err error
		mutual, err = s.socialCli.IsMutualFollow(ctx, senderID, receiverID)
		if err != nil {
			return nil, err
		}
	}
	if !mutual {
		// 非互关：仅允许发一条（sender -> receiver 方向）
		n, err := s.msgRepo.CountFromSenderToReceiver(ctx, senderID, receiverID)
		if err != nil {
			return nil, err
		}
		if n >= 1 {
			return nil, ErrIntroLimitExceeded
		}
	}

	// 获取或创建会话（同步，需要 conv_id）
	conv, err := s.convRepo.GetOrCreate(ctx, senderID, receiverID)
	if err != nil {
		return nil, err
	}

	idempotencyKey := uuid.New().String()
	now := time.Now()

	// 异步落库：发布到 MQ
	payload := MessagePersistPayload{
		ConversationID: conv.ID,
		SenderID:       senderID,
		ReceiverID:     receiverID,
		Content:        content,
		IdempotencyKey: idempotencyKey,
	}
	if err := s.mqPublisher.PublishMessage(payload); err != nil {
		return nil, err
	}

	// 构造返回对象（ID 暂为 0，Consumer 落库后才有；客户端用 temp_id 做乐观展示）
	msg := &Message{
		ConversationID: conv.ID,
		SenderID:       senderID,
		ReceiverID:     receiverID,
		Content:        content,
		IdempotencyKey: &idempotencyKey,
		CreatedAt:      now,
	}

	// 实时推送给接收方
	if s.hub != nil {
		s.hub.BroadcastTo(receiverID, msg)
	}
	return msg, nil
}

// ListConversations 会话列表（含未读数）
func (s *IMService) ListConversations(ctx context.Context, userID uint, limit, offset int) ([]ConversationWithUnread, error) {
	list, err := s.convRepo.ListByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	result := make([]ConversationWithUnread, len(list))
	for i, c := range list {
		lastRead, _ := s.readRepo.GetLastReadAt(ctx, userID, c.ID)
		unread, _ := s.msgRepo.CountUnread(ctx, c.ID, userID, lastRead)
		result[i] = ConversationWithUnread{Conversation: c, UnreadCount: unread}
	}
	return result, nil
}

// ListMessages 消息历史（点进会话时自动标记已读）
func (s *IMService) ListMessages(ctx context.Context, userID, convID uint, limit, offset int) ([]Message, error) {
	conv, err := s.convRepo.GetByID(ctx, convID)
	if err != nil || conv == nil {
		return nil, errors.New("conversation not found")
	}
	if conv.UserA != userID && conv.UserB != userID {
		return nil, errors.New("not your conversation")
	}
	list, err := s.msgRepo.ListByConversationID(ctx, convID, limit, offset)
	if err != nil {
		return nil, err
	}
	// 点进会话即标记已读
	_ = s.readRepo.MarkRead(ctx, userID, convID, time.Now())
	return list, nil
}

// MarkRead 标记会话已读（显式调用，或点进会话时由 ListMessages 自动调用）
func (s *IMService) MarkRead(ctx context.Context, userID, convID uint) error {
	conv, err := s.convRepo.GetByID(ctx, convID)
	if err != nil || conv == nil {
		return errors.New("conversation not found")
	}
	if conv.UserA != userID && conv.UserB != userID {
		return errors.New("not your conversation")
	}
	return s.readRepo.MarkRead(ctx, userID, convID, time.Now())
}

// SearchMessages 模糊搜索消息（依赖 Meilisearch，仅搜当前用户参与的消息）
func (s *IMService) SearchMessages(ctx context.Context, userID uint, query string, limit, offset int) ([]Message, int64, error) {
	if s.searchClient == nil {
		return nil, 0, nil
	}
	ids, total, err := s.searchClient.Search(ctx, userID, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	if len(ids) == 0 {
		return []Message{}, total, nil
	}
	list, err := s.msgRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	msgMap := make(map[uint]Message)
	for _, m := range list {
		msgMap[m.ID] = m
	}
	// 按 Meilisearch 返回顺序排列，并二次校验归属
	result := make([]Message, 0, len(ids))
	for _, id := range ids {
		m, ok := msgMap[id]
		if !ok {
			continue
		}
		if m.SenderID != userID && m.ReceiverID != userID {
			continue
		}
		result = append(result, m)
	}
	return result, total, nil
}
