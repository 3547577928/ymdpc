package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// TestRateLimit 验证窗口内超过限流次数返回 429，进入新窗口后恢复
func TestRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := RateLimit(2, 50*time.Millisecond)
	router := gin.New()
	router.POST("/ping", handler, func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	for i := 0; i < 2; i++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/ping", nil)
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("request %d returned %d, expected 200", i+1, recorder.Code)
		}
	}
	blocked := httptest.NewRecorder()
	router.ServeHTTP(blocked, httptest.NewRequest(http.MethodPost, "/ping", nil))
	if blocked.Code != http.StatusTooManyRequests {
		t.Fatalf("throttled request returned %d, expected 429", blocked.Code)
	}
	// 等待窗口结束后恢复
	time.Sleep(60 * time.Millisecond)
	recovered := httptest.NewRecorder()
	router.ServeHTTP(recovered, httptest.NewRequest(http.MethodPost, "/ping", nil))
	if recovered.Code != http.StatusOK {
		t.Fatalf("request after window returned %d, expected 200", recovered.Code)
	}
}
