package controllers

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"quietsignal/backend/models"

	"github.com/gin-gonic/gin"
)

// 全文搜索：FTS5 trigram 命中中文与英文子串；索引不可用时回退 LIKE
func TestPostFullTextSearch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newCommunityTestDB(t)
	SetupPostSearch(db)
	if !postSearchFTS.Load() {
		t.Skip("sqlite driver without FTS5 trigram, LIKE fallback covered elsewhere")
	}
	posts := []models.Post{
		{AuthorID: 1, Title: "把复杂系统写成安静的界面", Slug: "quiet", Content: "正文讲界面与克制", Status: "published", PublishedAt: time.Now()},
		{AuthorID: 1, Title: "full text search", Slug: "fts", Content: "search with sqlite trigram", Status: "published", PublishedAt: time.Now()},
		{AuthorID: 1, Title: "无关的一篇", Slug: "other", Content: "别的内容", Status: "published", PublishedAt: time.Now()},
	}
	if err := db.Create(&posts).Error; err != nil {
		t.Fatal(err)
	}
	community := &CommunityController{DB: db}

	// 中文子串（≥3 字符）经全文索引命中
	hit := performRequest(community.Feed, http.MethodGet, "/api/feed?q=%E5%AE%89%E9%9D%99%E7%9A%84", "", nil)
	if hit.Code != http.StatusOK || !strings.Contains(hit.Body.String(), "安静的界面") {
		t.Fatalf("expected Chinese FTS hit, got %d %s", hit.Code, hit.Body.String())
	}
	if strings.Contains(hit.Body.String(), "无关的一篇") {
		t.Fatalf("unrelated post leaked into FTS result: %s", hit.Body.String())
	}
	// 英文子串命中
	english := performRequest(community.Feed, http.MethodGet, "/api/feed?q=trigram", "", nil)
	if !strings.Contains(english.Body.String(), "full text search") {
		t.Fatalf("expected English FTS hit, got %s", english.Body.String())
	}
	// 过短的关键字回退 LIKE，行为与旧版一致
	short := performRequest(community.Feed, http.MethodGet, "/api/feed?q=%E6%AD%A3%E6%96%87", "", nil) // 正文
	if !strings.Contains(short.Body.String(), "安静的界面") {
		t.Fatalf("expected LIKE fallback for short keyword, got %s", short.Body.String())
	}
	postSearchFTS.Store(false)
}

func TestFTSMatchExpression(t *testing.T) {
	if got := ftsMatchExpression("quiet signal"); got != "\"quiet\" \"signal\"" {
		t.Fatalf("unexpected expression: %q", got)
	}
	if got := ftsMatchExpression("全文"); got != "" {
		t.Fatalf("short term must be empty for LIKE fallback, got %q", got)
	}
	if got := ftsMatchExpression("\"恶意\" 注入"); !strings.Contains(got, "\"\"\"恶意\"\"\"") {
		t.Fatalf("quotes must be escaped: %q", got)
	}
}

// 过期数据清理：旧验证码、已读旧通知删除；未读新通知保留
func TestCleanupOldData(t *testing.T) {
	db := newCommunityTestDB(t)
	old := time.Now().Add(-48 * time.Hour)
	if err := db.Create(&models.EmailLoginCode{Email: "a@b.co", CodeHash: "x", ExpiresAt: old, CreatedAt: old}).Error; err != nil {
		t.Fatal(err)
	}
	readAt := time.Now().Add(-100 * 24 * time.Hour)
	stale := models.Notification{UserID: 1, ActorID: 2, Type: "like", ResourceID: 1, ReadAt: &readAt}
	fresh := models.Notification{UserID: 1, ActorID: 2, Type: "follow", ResourceID: 2}
	if err := db.Create(&stale).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&fresh).Error; err != nil {
		t.Fatal(err)
	}
	if err := CleanupOldData(db, time.Now()); err != nil {
		t.Fatal(err)
	}
	var codes, notifications int64
	db.Model(&models.EmailLoginCode{}).Count(&codes)
	db.Model(&models.Notification{}).Count(&notifications)
	if codes != 0 {
		t.Fatalf("expected old email codes removed, got %d", codes)
	}
	if notifications != 1 {
		t.Fatalf("expected only fresh unread notification kept, got %d", notifications)
	}
}

// 同一邮箱 60 秒内只能请求一次验证码
func TestEmailCodePerEmailRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newCommunityTestDB(t)
	auth := &AuthController{DB: db, Secret: "test-secret", EmailCodeTTL: time.Minute, SendEmailCode: func(_, _ string) error { return nil }}
	first := performRequest(auth.RequestEmailCode, http.MethodPost, "/api/auth/email-code/request", "{\"email\":\"writer@example.com\"}", nil)
	if first.Code != http.StatusOK {
		t.Fatalf("first request returned %d: %s", first.Code, first.Body.String())
	}
	second := performRequest(auth.RequestEmailCode, http.MethodPost, "/api/auth/email-code/request", "{\"email\":\"writer@example.com\"}", nil)
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second request within 60s must be 429, got %d", second.Code)
	}
	other := performRequest(auth.RequestEmailCode, http.MethodPost, "/api/auth/email-code/request", "{\"email\":\"other@example.com\"}", nil)
	if other.Code != http.StatusOK {
		t.Fatalf("different email must not be limited, got %d", other.Code)
	}
}

// 用户主页文章列表分页
func TestProfilePagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newCommunityTestDB(t)
	user := models.User{Username: "alice", Nickname: "Alice", Role: "user", Status: "active"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 15; i++ {
		post := models.Post{AuthorID: user.ID, Title: "文章标题编号", Slug: strings.Repeat("p", i+1) + "-slug", Content: "body", Status: "published", PublishedAt: time.Now()}
		if err := db.Create(&post).Error; err != nil {
			t.Fatal(err)
		}
	}
	community := &CommunityController{DB: db}
	page1 := performRequest(community.Profile, http.MethodGet, "/api/users/alice?page=1&pageSize=12", "", gin.Params{{Key: "username", Value: "alice"}})
	if page1.Code != http.StatusOK || !strings.Contains(page1.Body.String(), "\"postCount\":15") {
		t.Fatalf("unexpected profile page 1: %d %s", page1.Code, page1.Body.String())
	}
	if strings.Count(page1.Body.String(), "文章标题编号") != 12 {
		t.Fatalf("expected 12 posts on page 1, got %s", page1.Body.String())
	}
	page2 := performRequest(community.Profile, http.MethodGet, "/api/users/alice?page=2&pageSize=12", "", gin.Params{{Key: "username", Value: "alice"}})
	if page2.Code != http.StatusOK || strings.Count(page2.Body.String(), "文章标题编号") != 3 {
		t.Fatalf("expected 3 posts on page 2, got %s", page2.Body.String())
	}
}
