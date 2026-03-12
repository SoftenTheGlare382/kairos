package social

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"kairos/pkg/middleware"
)

// SocialHandler 关注 HTTP 处理器
type SocialHandler struct {
	svc *SocialService
}

// NewSocialHandler 创建
func NewSocialHandler(svc *SocialService) *SocialHandler {
	return &SocialHandler{svc: svc}
}

// Follow 关注
func (h *SocialHandler) Follow(c *gin.Context) {
	var req struct {
		FollowingID uint `json:"following_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.FollowingID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "following_id is required"})
		return
	}
	accountID, ok := middleware.GetAccountID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "accountID not found"})
		return
	}
	if err := h.svc.Follow(c.Request.Context(), accountID, req.FollowingID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "follow success"})
}

// Unfollow 取关
func (h *SocialHandler) Unfollow(c *gin.Context) {
	var req struct {
		FollowingID uint `json:"following_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.FollowingID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "following_id is required"})
		return
	}
	accountID, ok := middleware.GetAccountID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "accountID not found"})
		return
	}
	if err := h.svc.Unfollow(c.Request.Context(), accountID, req.FollowingID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "unfollow success"})
}

// Followers 粉丝列表（分页）
func (h *SocialHandler) Followers(c *gin.Context) {
	var req struct {
		UserID   uint `json:"user_id"`
		Page     int  `json:"page"`
		PageSize int  `json:"page_size"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := req.UserID
	if userID <= 0 {
		if id, ok := middleware.GetAccountID(c); ok {
			userID = id
		}
	}
	if userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}
	result, err := h.svc.ListFollowers(c.Request.Context(), userID, req.Page, req.PageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// Following 关注列表（分页）
func (h *SocialHandler) Following(c *gin.Context) {
	var req struct {
		UserID   uint `json:"user_id"`
		Page     int  `json:"page"`
		PageSize int  `json:"page_size"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := req.UserID
	if userID <= 0 {
		if id, ok := middleware.GetAccountID(c); ok {
			userID = id
		}
	}
	if userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}
	result, err := h.svc.ListFollowing(c.Request.Context(), userID, req.Page, req.PageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
