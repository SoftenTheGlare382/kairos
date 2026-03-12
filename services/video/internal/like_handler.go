package video

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"kairos/pkg/middleware"
)

// LikeHandler 点赞 HTTP 处理器
type LikeHandler struct {
	svc *LikeService
}

// NewLikeHandler 创建
func NewLikeHandler(svc *LikeService) *LikeHandler {
	return &LikeHandler{svc: svc}
}

// Like 点赞
func (h *LikeHandler) Like(c *gin.Context) {
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
	accountID, ok := middleware.GetAccountID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "accountID not found"})
		return
	}
	like := &Like{VideoID: req.VideoID, AccountID: accountID}
	if err := h.svc.Like(c.Request.Context(), like); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "like success"})
}

// Unlike 取消点赞
func (h *LikeHandler) Unlike(c *gin.Context) {
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
	accountID, ok := middleware.GetAccountID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "accountID not found"})
		return
	}
	like := &Like{VideoID: req.VideoID, AccountID: accountID}
	if err := h.svc.Unlike(c.Request.Context(), like); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "unlike success"})
}

// IsLiked 是否已点赞
func (h *LikeHandler) IsLiked(c *gin.Context) {
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
	accountID, ok := middleware.GetAccountID(c)
	if !ok {
		c.JSON(http.StatusOK, gin.H{"is_liked": false})
		return
	}
	liked, err := h.svc.IsLiked(c.Request.Context(), req.VideoID, accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"is_liked": liked})
}

// ListMyLikedVideos 我点赞的视频列表
func (h *LikeHandler) ListMyLikedVideos(c *gin.Context) {
	accountID, ok := middleware.GetAccountID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "accountID not found"})
		return
	}
	list, err := h.svc.ListLikedVideos(c.Request.Context(), accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}
