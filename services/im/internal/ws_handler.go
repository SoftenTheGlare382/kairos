package im

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"kairos/pkg/auth"
	"kairos/pkg/config"
	"kairos/pkg/redis"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

const authTimeout = 10 * time.Second

// HandleWebSocket 处理 WebSocket 连接。连接成功后，客户端需在 10 秒内发送首条消息：
// {"type":"auth","token":"<jwt>"}，校验通过后返回 {"type":"auth_ok"}，之后可接收实时消息。
// 避免在 URL query 中传 token，防止日志、Referer 泄露。
func (h *IMHandler) HandleWebSocket(c *gin.Context, rdb *redis.Client, cfgJwt config.JwtConfig) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	// 不在此处 defer ws.Close()，conn.Run 接管连接生命周期

	ws.SetReadDeadline(time.Now().Add(authTimeout))
	_, msgBytes, err := ws.ReadMessage()
	if err != nil {
		ws.Close()
		return
	}

	var authMsg struct {
		Type  string `json:"type"`
		Token string `json:"token"`
	}
	if json.Unmarshal(msgBytes, &authMsg) != nil || authMsg.Type != "auth" || authMsg.Token == "" {
		_ = ws.WriteJSON(map[string]string{"type": "auth_err", "error": "invalid auth message, expected {\"type\":\"auth\",\"token\":\"...\"}"})
		ws.Close()
		return
	}

	claims, err := auth.ParseToken(authMsg.Token, cfgJwt)
	if err != nil {
		_ = ws.WriteJSON(map[string]string{"type": "auth_err", "error": "invalid or expired token"})
		ws.Close()
		return
	}
	key := fmt.Sprintf("account:%d", claims.AccountID)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 50*time.Millisecond)
	b, err := rdb.GetBytes(ctx, key)
	cancel()
	if err != nil || string(b) != authMsg.Token {
		_ = ws.WriteJSON(map[string]string{"type": "auth_err", "error": "token revoked"})
		ws.Close()
		return
	}

	ws.SetReadDeadline(time.Time{}) // 取消超时

	conn := h.hub.Register(uint(claims.AccountID), ws)
	_ = ws.WriteJSON(map[string]string{"type": "auth_ok"})
	go conn.Run(h.hub)
}

