package controllers

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"quietsignal/backend/models"

	"github.com/gin-gonic/gin"
)

// SSEEvent 一条 Server-Sent Event：Name 为事件类型（comment/topic/typing），
// 空串走默认 message 通道
type SSEEvent struct {
	Name string
	Data string
}

// EventBus 按主题分发 SSE 事件，非阻塞推送（通道满即丢弃，绝不阻塞业务写路径）。
// 主题规划：comments:{slug} 文章评论流；forum 论坛新帖流
type EventBus struct {
	mu     sync.Mutex
	topics map[string]map[chan SSEEvent]struct{}
}

var eventBus = &EventBus{topics: map[string]map[chan SSEEvent]struct{}{}}

func (b *EventBus) Subscribe(topic string) chan SSEEvent {
	ch := make(chan SSEEvent, 16)
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.topics[topic] == nil {
		b.topics[topic] = map[chan SSEEvent]struct{}{}
	}
	b.topics[topic][ch] = struct{}{}
	return ch
}

func (b *EventBus) Unsubscribe(topic string, ch chan SSEEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.topics[topic], ch)
	if len(b.topics[topic]) == 0 {
		delete(b.topics, topic)
	}
}

func (b *EventBus) Publish(topic, name, data string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.topics[topic] {
		select {
		case ch <- SSEEvent{Name: name, Data: data}:
		default:
		}
	}
}

// streamSSE 是 SSE 端点的公共循环：先发送 initial（默认通道），之后转发事件，
// 25 秒注释行心跳防代理掐断空闲连接；http.Server 的 WriteTimeout 是全程上限，
// 每次写入前顺延写截止时间
func streamSSE(c *gin.Context, events <-chan SSEEvent, initial string) {
	writer := c.Writer
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")
	// 经过 nginx 反向代理时关闭其响应缓冲，否则事件会被攒批延迟
	writer.Header().Set("X-Accel-Buffering", "no")
	send := func(payload string) bool {
		_ = http.NewResponseController(writer).SetWriteDeadline(time.Now().Add(15 * time.Second))
		if _, err := fmt.Fprint(writer, payload); err != nil {
			return false
		}
		writer.Flush()
		return true
	}
	if !send("data: " + initial + "\n\n") {
		return
	}
	keepalive := time.NewTicker(25 * time.Second)
	defer keepalive.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case event := <-events:
			payload := "data: " + event.Data + "\n\n"
			if event.Name != "" {
				payload = "event: " + event.Name + "\n" + payload
			}
			if !send(payload) {
				return
			}
		case <-keepalive.C:
			if !send(": keepalive\n\n") {
				return
			}
		}
	}
}

// CommentStream 文章评论的实时流：新评论、删除与「正在输入」提醒
func (cc *CommunityController) CommentStream(c *gin.Context) {
	slug := c.Param("slug")
	var count int64
	cc.DB.Model(&models.Post{}).Where("slug = ? AND status = ? AND moderation_status = ?", slug, "published", "normal").Count(&count)
	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文章不存在"})
		return
	}
	events := eventBus.Subscribe("comments:" + slug)
	defer eventBus.Unsubscribe("comments:"+slug, events)
	streamSSE(c, events, "ok")
}

// CommentTyping 广播「某用户正在输入」，不落库；客户端每 3 秒最多发一次
func (cc *CommunityController) CommentTyping(c *gin.Context) {
	userID := currentUserID(c)
	var user models.User
	if err := cc.DB.Select("nickname").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "unauthorized"})
		return
	}
	eventBus.Publish("comments:"+c.Param("slug"), "typing", fmt.Sprintf(`{"nickname":%q}`, user.Nickname))
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// ForumStream 论坛新帖实时流
func (fc *ForumController) ForumStream(c *gin.Context) {
	events := eventBus.Subscribe("forum")
	defer eventBus.Unsubscribe("forum", events)
	streamSSE(c, events, "ok")
}
