package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// rateBucket 单个 IP 在一个时间窗口内的请求计数
type rateBucket struct {
	count     int
	windowEnd time.Time
}

// rateLimiter 基于固定窗口的内存限流器，按客户端 IP 计数
type rateLimiter struct {
	mu     sync.Mutex
	counts map[string]*rateBucket
	limit  int
	window time.Duration
}

// RateLimit 返回按 IP 限流的中间件，窗口内超过 limit 次返回 429
func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	limiter := &rateLimiter{counts: map[string]*rateBucket{}, limit: limit, window: window}
	return func(c *gin.Context) {
		if !limiter.allow(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"code": 429, "message": "请求过于频繁，请稍后再试"})
			return
		}
		c.Next()
	}
}

func (l *rateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	bucket, ok := l.counts[key]
	if !ok || now.After(bucket.windowEnd) {
		l.counts[key] = &rateBucket{count: 1, windowEnd: now.Add(l.window)}
		return true
	}
	bucket.count++
	// 计数条目过多时清理过期窗口，避免 map 无限增长
	if len(l.counts) > 10000 {
		for entryKey, entry := range l.counts {
			if now.After(entry.windowEnd) {
				delete(l.counts, entryKey)
			}
		}
	}
	return bucket.count <= l.limit
}
