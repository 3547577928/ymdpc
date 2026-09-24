package controllers

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"quietsignal/backend/models"

	"github.com/gin-gonic/gin"
)

// NotificationHub 维护在线用户的 SSE 订阅：新通知落库后秒级推送到导航铃铛，
// 取代前端 30 秒轮询（轮询保留作为断线兜底）
type NotificationHub struct {
	mu          sync.Mutex
	subscribers map[uint]map[chan struct{}]struct{}
}

var notificationHub = &NotificationHub{subscribers: map[uint]map[chan struct{}]struct{}{}}

func (h *NotificationHub) Subscribe(userID uint) chan struct{} {
	ch := make(chan struct{}, 8)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.subscribers[userID] == nil {
		h.subscribers[userID] = map[chan struct{}]struct{}{}
	}
	h.subscribers[userID][ch] = struct{}{}
	return ch
}

func (h *NotificationHub) Unsubscribe(userID uint, ch chan struct{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.subscribers[userID], ch)
	if len(h.subscribers[userID]) == 0 {
		delete(h.subscribers, userID)
	}
}

// Notify 非阻塞提醒：通道满时丢弃，客户端下一次心跳或轮询会追上，
// 永远不会因为慢消费者阻塞点赞/评论等业务写路径
func (h *NotificationHub) Notify(userID uint) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subscribers[userID] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// NotificationStream SSE 端点：推送当前未读通知数。
// 连接建立先推一次当前值，之后每次 hub 提醒时重算并推送；
// 25 秒注释行心跳防止代理掐断空闲连接。
func (ic *InteractionController) NotificationStream(c *gin.Context) {
	userID := currentUserID(c)
	unread := func() int64 {
		var count int64
		ic.DB.Model(&models.Notification{}).Where("user_id = ? AND read_at IS NULL", userID).Count(&count)
		return count
	}
	writer := c.Writer
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")
	// 经过 nginx 反向代理时关闭其响应缓冲，否则事件会被攒批延迟
	writer.Header().Set("X-Accel-Buffering", "no")
	ch := notificationHub.Subscribe(userID)
	defer notificationHub.Unsubscribe(userID, ch)

	send := func(payload string) bool {
		// http.Server 的 WriteTimeout 是整个请求的上限，长连接必须逐次顺延写截止时间
		_ = http.NewResponseController(writer).SetWriteDeadline(time.Now().Add(15 * time.Second))
		if _, err := fmt.Fprint(writer, payload); err != nil {
			return false
		}
		writer.Flush()
		return true
	}
	if !send(fmt.Sprintf("data: %d\n\n", unread())) {
		return
	}
	keepalive := time.NewTicker(25 * time.Second)
	defer keepalive.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ch:
			if !send(fmt.Sprintf("data: %d\n\n", unread())) {
				return
			}
		case <-keepalive.C:
			if !send(": keepalive\n\n") {
				return
			}
		}
	}
}
