package video

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"kairos/pkg/grpc"
	"kairos/pkg/middleware"
)

// CommentHandler 评论 HTTP 处理器
type CommentHandler struct {
	svc        *CommentService
	accountCli *grpc.AccountClient
}

// NewCommentHandler 创建
func NewCommentHandler(svc *CommentService, accountCli *grpc.AccountClient) *CommentHandler {
	return &CommentHandler{svc: svc, accountCli: accountCli}
}

// PublishComment 发布评论
func (h *CommentHandler) PublishComment(c *gin.Context) {
	var req struct {
		VideoID uint   `json:"video_id"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.VideoID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "video_id is required"})
		return
	}
	if req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content is required"})
		return
	}
	accountID, ok := middleware.GetAccountID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "accountID not found"})
		return
	}
	user, err := h.accountCli.GetByID(c.Request.Context(), accountID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to get user: " + err.Error()})
		return
	}
	comment := &Comment{
		Username: user.Username,
		VideoID:  req.VideoID,
		AuthorID: accountID,
		Content:  req.Content,
	}
	if err := h.svc.Publish(c.Request.Context(), comment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "comment published successfully"})
}

// DeleteComment 删除评论
func (h *CommentHandler) DeleteComment(c *gin.Context) {
	var req struct {
		CommentID uint `json:"comment_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.CommentID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "comment_id is required"})
		return
	}
	accountID, ok := middleware.GetAccountID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "accountID not found"})
		return
	}
	if err := h.svc.Delete(c.Request.Context(), req.CommentID, accountID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "comment deleted successfully"})
}

// GetAllComments 获取视频下所有评论（公开）
func (h *CommentHandler) GetAllComments(c *gin.Context) {
	var req struct {
		VideoID uint `json:"video_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.VideoID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "video_id is required"})
		return
	}
	list, err := h.svc.GetAll(c.Request.Context(), req.VideoID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}
