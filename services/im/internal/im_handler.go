package im

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"kairos/pkg/middleware"
)

// IMHandler IM HTTP 处理器
type IMHandler struct {
	svc *IMService
	hub *Hub
}

// NewIMHandler 创建
func NewIMHandler(svc *IMService, hub *Hub) *IMHandler {
	return &IMHandler{svc: svc, hub: hub}
}

// SendMessage 发送消息
func (h *IMHandler) SendMessage(c *gin.Context) {
	var req struct {
		ReceiverID uint   `json:"receiver_id"`
		Content    string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	accountID, ok := middleware.GetAccountID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	msg, err := h.svc.SendMessage(c.Request.Context(), accountID, req.ReceiverID, req.Content)
	if err != nil {
		if err == ErrIntroLimitExceeded || err == ErrMutualFollowRequired {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, msg)
}

// ListConversations 会话列表
func (h *IMHandler) ListConversations(c *gin.Context) {
	var req struct {
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Limit = 20
		req.Offset = 0
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	accountID, ok := middleware.GetAccountID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	list, err := h.svc.ListConversations(c.Request.Context(), accountID, req.Limit, req.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// MarkRead 标记会话已读
func (h *IMHandler) MarkRead(c *gin.Context) {
	var req struct {
		ConversationID uint `json:"conversation_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ConversationID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation_id is required"})
		return
	}
	accountID, ok := middleware.GetAccountID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if err := h.svc.MarkRead(c.Request.Context(), accountID, req.ConversationID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// ListMessages 消息历史
func (h *IMHandler) ListMessages(c *gin.Context) {
	var req struct {
		ConversationID uint `json:"conversation_id"`
		Limit          int  `json:"limit"`
		Offset         int  `json:"offset"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ConversationID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation_id is required"})
		return
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	accountID, ok := middleware.GetAccountID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	list, err := h.svc.ListMessages(c.Request.Context(), accountID, req.ConversationID, req.Limit, req.Offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// SearchMessages 模糊搜索消息（依赖 Meilisearch）
func (h *IMHandler) SearchMessages(c *gin.Context) {
	var req struct {
		Query  string `json:"query"`
		Limit  int    `json:"limit"`
		Offset int    `json:"offset"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Limit = 20
		req.Offset = 0
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	accountID, ok := middleware.GetAccountID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	list, total, err := h.svc.SearchMessages(c.Request.Context(), accountID, req.Query, req.Limit, req.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"messages": list, "total": total})
}
