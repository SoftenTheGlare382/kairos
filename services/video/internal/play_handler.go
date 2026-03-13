package video

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"kairos/pkg/grpc"
	"kairos/pkg/middleware"
)

// PlayHandler 播放统计 HTTP 处理器
type PlayHandler struct {
	playSvc   *PlayService
	accountCli *grpc.AccountClient
}

// NewPlayHandler 创建
func NewPlayHandler(playSvc *PlayService, accountCli *grpc.AccountClient) *PlayHandler {
	return &PlayHandler{playSvc: playSvc, accountCli: accountCli}
}

// RecordPlay 记录一次播放
// POST /video/recordPlay
func (h *PlayHandler) RecordPlay(c *gin.Context) {
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
	if err := h.playSvc.RecordPlay(c.Request.Context(), accountID, req.VideoID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "play recorded"})
}

// PlayRecordItem 播放记录项（含用户名）
type PlayRecordItem struct {
	AccountID  uint   `json:"account_id"`
	Username   string `json:"username"`
	PlayCount  int64  `json:"play_count"`
	LastPlayAt string `json:"last_play_at"`
}

// ListPlayRecords 获取某视频的播放记录（谁播放了几次、最近播放时间），仅作者可查
// POST /video/listPlayRecords
func (h *PlayHandler) ListPlayRecords(c *gin.Context) {
	var req struct {
		VideoID uint `json:"video_id"`
		Limit   int  `json:"limit"`
		Offset  int  `json:"offset"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.VideoID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "video_id is required"})
		return
	}
	accountID, ok := middleware.GetAccountID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	list, err := h.playSvc.ListPlayRecords(c.Request.Context(), req.VideoID, accountID, req.Limit, req.Offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(list) == 0 {
		c.JSON(http.StatusOK, []PlayRecordItem{})
		return
	}
	accountIDs := make([]uint, len(list))
	for i := range list {
		accountIDs[i] = list[i].AccountID
	}
	users, err := h.accountCli.GetByIDs(c.Request.Context(), accountIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get users"})
		return
	}
	userMap := make(map[uint]grpc.UserInfo)
	for _, u := range users {
		userMap[u.ID] = u
	}
	result := make([]PlayRecordItem, len(list))
	for i, r := range list {
		username := ""
		if u, ok := userMap[r.AccountID]; ok {
			username = u.Username
		}
		result[i] = PlayRecordItem{
			AccountID:  r.AccountID,
			Username:   username,
			PlayCount:  r.PlayCount,
			LastPlayAt: r.LastPlayAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	c.JSON(http.StatusOK, result)
}
