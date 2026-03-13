package video

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"kairos/pkg/middleware"
)

// FavoriteHandler 收藏 HTTP 处理器
type FavoriteHandler struct {
	svc *FavoriteService
}

// NewFavoriteHandler 创建
func NewFavoriteHandler(svc *FavoriteService) *FavoriteHandler {
	return &FavoriteHandler{svc: svc}
}

// Favorite 收藏视频
// POST /video/favorite
func (h *FavoriteHandler) Favorite(c *gin.Context) {
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}
	if err := h.svc.Favorite(c.Request.Context(), accountID, req.VideoID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "favorite success"})
}

// Unfavorite 取消收藏
// POST /video/unfavorite
func (h *FavoriteHandler) Unfavorite(c *gin.Context) {
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}
	if err := h.svc.Unfavorite(c.Request.Context(), accountID, req.VideoID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "unfavorite success"})
}

// IsFavorited 是否已收藏
// POST /video/isFavorited
func (h *FavoriteHandler) IsFavorited(c *gin.Context) {
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
		c.JSON(http.StatusOK, gin.H{"is_favorited": false})
		return
	}
	favorited, err := h.svc.IsFavorited(c.Request.Context(), req.VideoID, accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"is_favorited": favorited})
}

// ListMyFavoritedVideos 我收藏的视频列表
// POST /video/listMyFavoritedVideos
func (h *FavoriteHandler) ListMyFavoritedVideos(c *gin.Context) {
	accountID, ok := middleware.GetAccountID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}
	list, err := h.svc.ListMyFavoritedVideos(c.Request.Context(), accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}
