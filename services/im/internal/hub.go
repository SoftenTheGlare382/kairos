package im

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// Hub WebSocket 连接管理，单实例内存广播
type Hub struct {
	mu          sync.RWMutex
	connections map[uint]map[*conn]struct{} // userID -> set of conns
}

type conn struct {
	userID uint
	ws     *websocket.Conn
	send   chan []byte
}

// NewHub 创建
func NewHub() *Hub {
	return &Hub{connections: make(map[uint]map[*conn]struct{})}
}

// Register 注册连接
func (h *Hub) Register(userID uint, ws *websocket.Conn) *conn {
	c := &conn{userID: userID, ws: ws, send: make(chan []byte, 64)}
	h.mu.Lock()
	if h.connections[userID] == nil {
		h.connections[userID] = make(map[*conn]struct{})
	}
	h.connections[userID][c] = struct{}{}
	h.mu.Unlock()
	return c
}

// Unregister 注销连接
func (h *Hub) Unregister(userID uint, c *conn) {
	h.mu.Lock()
	if m := h.connections[userID]; m != nil {
		delete(m, c)
		if len(m) == 0 {
			delete(h.connections, userID)
		}
	}
	h.mu.Unlock()
	close(c.send)
}

// BroadcastTo 向指定用户推送消息
func (h *Hub) BroadcastTo(userID uint, msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.mu.RLock()
	conns := h.connections[userID]
	h.mu.RUnlock()
	for c := range conns {
		select {
		case c.send <- data:
		default:
			log.Printf("im hub: drop message for user %d", userID)
		}
	}
}

// Run 启动 conn 的读写循环（在 goroutine 中调用）
func (c *conn) Run(h *Hub) {
	defer func() {
		h.Unregister(c.userID, c)
		c.ws.Close()
	}()
	go c.writePump()
	c.readPump()
}

func (c *conn) readPump() {
	defer c.ws.Close()
	c.ws.SetReadLimit(64 << 10)
	for {
		_, _, err := c.ws.ReadMessage()
		if err != nil {
			break
		}
		// 可扩展：处理 ping/pong、心跳等
	}
}

func (c *conn) writePump() {
	for b := range c.send {
		if err := c.ws.WriteMessage(websocket.TextMessage, b); err != nil {
			break
		}
	}
}
