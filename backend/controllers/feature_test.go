package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"quietsignal/backend/models"

	"github.com/gin-gonic/gin"
)

// 搜索落地页：FTS 高亮标记随结果返回
func TestSearchPageWithHighlight(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newCommunityTestDB(t)
	SetupPostSearch(db)
	if !postSearchFTS.Load() {
		t.Skip("sqlite driver without FTS5 trigram")
	}
	post := models.Post{AuthorID: 1, Title: "界面系统的安静设计", Slug: "quiet-ui", Summary: "摘要", Content: "正文讨论安静的界面应该如何设计取舍", Status: "published", PublishedAt: time.Now()}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	controller := &PostController{DB: db}
	resp := performRequest(controller.Search, http.MethodGet, "/api/search?q=%E5%AE%89%E9%9D%99%E7%9A%84%E7%95%8C%E9%9D%A2", "", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("search returned %d: %s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	if !strings.Contains(body, "\"fts\":true") {
		t.Fatalf("expected fts mode, got %s", body)
	}
	// 标题或摘要中应带高亮标记（JSON 转义为 \u0001）
	if !strings.Contains(body, "titleMarked") || !strings.Contains(body, "\\u0001") {
		t.Fatalf("expected highlight markers in response: %s", body)
	}
	postSearchFTS.Store(false)
}

// RSS：输出包含公开文章的 XML
func TestRSSFeed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newCommunityTestDB(t)
	user := models.User{Username: "alice", Nickname: "Alice", Role: "user", Status: "active"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	post := models.Post{AuthorID: user.ID, Title: "RSS 测试文章", Slug: "rss-test", Summary: "摘要", Content: "body", Status: "published", PublishedAt: time.Now()}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	controller := &PostController{DB: db}
	resp := performRequest(controller.RSSFeed, http.MethodGet, "/api/feed.xml", "", nil)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "RSS 测试文章") || !strings.Contains(resp.Body.String(), "<rss version=\"2.0\">") {
		t.Fatalf("unexpected rss response: %d %s", resp.Code, resp.Body.String())
	}
}

// 标签订阅：发布带订阅标签的文章时订阅者收到通知，且与粉丝通知去重
func TestTagSubscriptionNotifies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newCommunityTestDB(t)
	users := []models.User{{Username: "author", Nickname: "Author", Role: "user", Status: "active"}, {Username: "reader", Nickname: "Reader", Role: "user", Status: "active"}}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	// reader 同时是作者粉丝 + 标签订阅者，只能收到一条通知
	if err := db.Create(&models.Follow{FollowerID: users[1].ID, FollowingID: users[0].ID}).Error; err != nil {
		t.Fatal(err)
	}
	tag := models.Tag{Name: "工程", Slug: "engineering"}
	if err := db.Create(&tag).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.TagSubscription{UserID: users[1].ID, TagID: tag.ID}).Error; err != nil {
		t.Fatal(err)
	}
	post := models.Post{AuthorID: users[0].ID, Title: "工程文章", Slug: "eng-post", Content: "body", Status: "published", PublishedAt: time.Now()}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO post_tags (post_id, tag_id) VALUES (?, ?)", post.ID, tag.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := notifyFollowersOfPost(db, users[0].ID, post.ID); err != nil {
		t.Fatal(err)
	}
	var count int64
	db.Model(&models.Notification{}).Where("user_id = ? AND type = ?", users[1].ID, "post").Count(&count)
	if count != 1 {
		t.Fatalf("expected exactly one deduped notification, got %d", count)
	}
}

// 系列：创建、挂载文章、详情包含目录；他人系列不可挂载
func TestSeriesFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newCommunityTestDB(t)
	users := []models.User{{Username: "alice", Nickname: "Alice", Role: "user", Status: "active"}, {Username: "bob", Nickname: "Bob", Role: "user", Status: "active"}}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	series := &SeriesController{DB: db}
	created := authenticatedRequest(series.Create, http.MethodPost, "/api/series", `{"title":"系统设计笔记"}`, nil, users[0].ID, "user")
	if created.Code != http.StatusCreated {
		t.Fatalf("create series returned %d: %s", created.Code, created.Body.String())
	}
	var seriesPayload struct {
		Data SeriesListItemDTO `json:"data"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &seriesPayload); err != nil {
		t.Fatal(err)
	}
	seriesID := seriesPayload.Data.ID

	community := &CommunityController{DB: db}
	body := fmt.Sprintf(`{"title":"第一篇","content":"body","status":"published","seriesId":%d}`, seriesID)
	postResp := authenticatedRequest(community.CreatePost, http.MethodPost, "/api/posts", body, nil, users[0].ID, "user")
	if postResp.Code != http.StatusCreated {
		t.Fatalf("create post with series returned %d: %s", postResp.Code, postResp.Body.String())
	}
	// bob 不能把文章挂进 alice 的系列
	stolen := authenticatedRequest(community.CreatePost, http.MethodPost, "/api/posts", fmt.Sprintf(`{"title":"偷窃","content":"body","seriesId":%d}`, seriesID), nil, users[1].ID, "user")
	if stolen.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for foreign series, got %d", stolen.Code)
	}

	detail := performRequest(series.Detail, http.MethodGet, "/api/series/"+seriesPayload.Data.Slug, "", gin.Params{{Key: "slug", Value: seriesPayload.Data.Slug}})
	if detail.Code != http.StatusOK || !strings.Contains(detail.Body.String(), "第一篇") || !strings.Contains(detail.Body.String(), "系统设计笔记") {
		t.Fatalf("unexpected series detail: %d %s", detail.Code, detail.Body.String())
	}
}

// 作者数据看板：总量与趋势正确聚合
func TestMyStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newCommunityTestDB(t)
	users := []models.User{{Username: "alice", Nickname: "Alice", Role: "user", Status: "active"}, {Username: "bob", Nickname: "Bob", Role: "user", Status: "active"}}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	post := models.Post{AuthorID: users[0].ID, Title: "文章", Slug: "post-1", Content: "body", Status: "published", Views: 10, PublishedAt: time.Now()}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.PostLike{UserID: users[1].ID, PostID: post.ID}).Error; err != nil {
		t.Fatal(err)
	}
	interactions := &InteractionController{DB: db}
	resp := authenticatedRequest(interactions.MyStats, http.MethodGet, "/api/me/stats", "", nil, users[0].ID, "user")
	if resp.Code != http.StatusOK {
		t.Fatalf("my stats returned %d: %s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	if !strings.Contains(body, `"views":10`) || !strings.Contains(body, `"likes":1`) || !strings.Contains(body, `"published":1`) {
		t.Fatalf("unexpected stats payload: %s", body)
	}
	if !strings.Contains(body, "dailyMetrics") || !strings.Contains(body, "topPosts") {
		t.Fatalf("expected metrics and top posts: %s", body)
	}
}
