package feed

import (
	"net/http"

	"kairos/pkg/middleware"

	"github.com/gin-gonic/gin"
)

// ListLatestRequest 最新流请求
type ListLatestRequest struct {
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

// ListByFollowingRequest 关注流请求
type ListByFollowingRequest struct {
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

// ListByPopularityRequest 热度流请求
type ListByPopularityRequest struct {
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

// Handler Feed HTTP Handler
type Handler struct {
	svc *Service
}

// NewHandler 创建 Handler
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func defaultLimitOffset(limit, offset int32) (int32, int32) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

// ListLatest 最新视频流
// POST /feed/listLatest
func (h *Handler) ListLatest(c *gin.Context) {
	var req ListLatestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = ListLatestRequest{}
	}
	limit, offset := defaultLimitOffset(req.Limit, req.Offset)
	accountID, _ := middleware.GetAccountID(c)

	list, err := h.svc.ListLatest(c.Request.Context(), limit, offset, accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// ListByFollowing 关注流（需登录）
// POST /feed/listByFollowing
func (h *Handler) ListByFollowing(c *gin.Context) {
	accountID, ok := middleware.GetAccountID(c)
	if !ok || accountID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}
	var req ListByFollowingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = ListByFollowingRequest{}
	}
	limit, offset := defaultLimitOffset(req.Limit, req.Offset)

	list, err := h.svc.ListByFollowing(c.Request.Context(), limit, offset, accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// ListByPopularity 热度流
// POST /feed/listByPopularity
func (h *Handler) ListByPopularity(c *gin.Context) {
	var req ListByPopularityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = ListByPopularityRequest{}
	}
	limit, offset := defaultLimitOffset(req.Limit, req.Offset)
	accountID, _ := middleware.GetAccountID(c)

	list, err := h.svc.ListByPopularity(c.Request.Context(), limit, offset, accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}
