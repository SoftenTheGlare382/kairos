package video

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	// "time"

	"github.com/gin-gonic/gin"
	"kairos/pkg/grpc"
	"kairos/pkg/middleware"
)

// VideoHandler 视频 HTTP 处理器
type VideoHandler struct {
	videoSvc   *VideoService
	accountCli *grpc.AccountClient
	storage    Storage
}

// NewVideoHandler 创建
func NewVideoHandler(videoSvc *VideoService, accountCli *grpc.AccountClient, storage Storage) *VideoHandler {
	return &VideoHandler{videoSvc: videoSvc, accountCli: accountCli, storage: storage}
}

// PublishVideo 发布视频（需 play_url、cover_url，通常先调用 UploadVideo/UploadCover）
func (h *VideoHandler) PublishVideo(c *gin.Context) {
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		PlayURL     string `json:"play_url"`
		CoverURL    string `json:"cover_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
	v := &Video{
		AuthorID:    accountID,
		Username:    user.Username,
		Title:       req.Title,
		Description: req.Description,
		PlayURL:     req.PlayURL,
		CoverURL:    req.CoverURL,
	}
	if err := h.videoSvc.Publish(c.Request.Context(), v); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, v)
}

// UploadVideo 上传视频文件，返回 play_url
func (h *VideoHandler) UploadVideo(c *gin.Context) {
	accountID, ok := middleware.GetAccountID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "accountID not found"})
		return
	}
	f, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing file"})
		return
	}
	const maxSize = 200 << 20 // 200MB
	if f.Size <= 0 || f.Size > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file size"})
		return
	}
	ext := strings.ToLower(filepath.Ext(f.Filename))
	if ext != ".mp4" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only .mp4 is allowed"})
		return
	}
	file, err := f.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer file.Close()

	filename := "video" + ext
	urlPath, err := h.storage.SaveVideo(accountID, filename, file, f.Size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	fullURL := buildAbsoluteURL(c, urlPath)
	c.JSON(http.StatusOK, gin.H{"url": fullURL, "play_url": fullURL})
}

// UploadCover 上传封面，返回 cover_url
func (h *VideoHandler) UploadCover(c *gin.Context) {
	accountID, ok := middleware.GetAccountID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "accountID not found"})
		return
	}
	f, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing file"})
		return
	}
	const maxSize = 10 << 20 // 10MB
	if f.Size <= 0 || f.Size > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file size"})
		return
	}
	ext := strings.ToLower(filepath.Ext(f.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "only .jpg/.jpeg/.png/.webp is allowed"})
		return
	}
	file, err := f.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer file.Close()

	filename := "cover" + ext
	urlPath, err := h.storage.SaveCover(accountID, filename, file, f.Size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	fullURL := buildAbsoluteURL(c, urlPath)
	c.JSON(http.StatusOK, gin.H{"url": fullURL, "cover_url": fullURL})
}

// DeleteVideo 删除视频
func (h *VideoHandler) DeleteVideo(c *gin.Context) {
	var req struct {
		ID uint `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	accountID, ok := middleware.GetAccountID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "accountID not found"})
		return
	}
	if err := h.videoSvc.Delete(c.Request.Context(), req.ID, accountID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "video deleted"})
}

// ListByAuthorID 按作者列出视频（公开）
func (h *VideoHandler) ListByAuthorID(c *gin.Context) {
	var req struct {
		AuthorID uint `json:"author_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	list, err := h.videoSvc.ListByAuthorID(c.Request.Context(), req.AuthorID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// Search 模糊搜索视频（按标题/描述，公开）
func (h *VideoHandler) Search(c *gin.Context) {
	var req struct {
		Query  string `json:"query"`
		Limit  int    `json:"limit"`
		Offset int    `json:"offset"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	list, total, err := h.videoSvc.Search(c.Request.Context(), req.Query, req.Limit, req.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"list":  list,
		"total": total,
	})
}

// GetDetail 视频详情（公开）
func (h *VideoHandler) GetDetail(c *gin.Context) {
	var req struct {
		ID uint `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	v, err := h.videoSvc.GetDetail(c.Request.Context(), req.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if v == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "video not found"})
		return
	}
	c.JSON(http.StatusOK, v)
}

func buildAbsoluteURL(c *gin.Context, path string) string {
	// 若已是完整 URL（如七牛云返回），直接返回
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if xf := c.GetHeader("X-Forwarded-Proto"); xf != "" {
		scheme = xf
	}
	return fmt.Sprintf("%s://%s%s", scheme, c.Request.Host, path)
}
