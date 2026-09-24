package controllers

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"quietsignal/backend/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// 忘记密码：验证码校验通过后重置密码，旧会话立即失效
func TestResetPasswordFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newCommunityTestDB(t)
	hash, _ := bcrypt.GenerateFromPassword([]byte("old-password-1"), bcrypt.DefaultCost)
	user := models.User{Username: "alice", Nickname: "Alice", Email: "alice@example.com", PasswordHash: string(hash), Role: "user", Status: "active", SessionVersion: 1}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	var sentCode string
	auth := &AuthController{DB: db, Secret: "test-secret", EmailCodeTTL: time.Minute, SendEmailCode: func(_, code string) error {
		sentCode = code
		return nil
	}}
	if resp := performRequest(auth.RequestEmailCode, http.MethodPost, "/api/auth/email-code/request", `{"email":"alice@example.com"}`, nil); resp.Code != http.StatusOK {
		t.Fatalf("request code returned %d", resp.Code)
	}
	body := fmt.Sprintf(`{"email":"alice@example.com","code":%q,"newPassword":"new-password-9"}`, sentCode)
	resp := performRequest(auth.ResetPassword, http.MethodPost, "/api/auth/password/reset", body, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("reset returned %d: %s", resp.Code, resp.Body.String())
	}
	var updated models.User
	if err := db.First(&updated, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(updated.PasswordHash), []byte("new-password-9")) != nil {
		t.Fatal("password was not updated")
	}
	if updated.SessionVersion != 2 {
		t.Fatalf("expected session version bump, got %d", updated.SessionVersion)
	}
	// 验证码一次性：复用同一条码应失败
	again := performRequest(auth.ResetPassword, http.MethodPost, "/api/auth/password/reset", body, nil)
	if again.Code != http.StatusUnauthorized {
		t.Fatalf("reused code must be rejected, got %d", again.Code)
	}
}

// 评论编辑：作者可改，他人不可，EditedAt 落库
func TestUpdateComment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newCommunityTestDB(t)
	users := []models.User{{Username: "alice", Nickname: "Alice", Role: "user", Status: "active"}, {Username: "bob", Nickname: "Bob", Role: "user", Status: "active"}}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	post := models.Post{AuthorID: users[0].ID, Title: "文章", Slug: "post", Content: "body", Status: "published", PublishedAt: time.Now()}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	comment := models.Comment{PostID: post.ID, AuthorID: users[0].ID, Content: "原文", Status: "published"}
	if err := db.Create(&comment).Error; err != nil {
		t.Fatal(err)
	}
	community := &CommunityController{DB: db}
	params := gin.Params{{Key: "id", Value: fmt.Sprint(comment.ID)}}
	edited := authenticatedRequest(community.UpdateComment, http.MethodPatch, "/api/comments/1", `{"content":"改过的内容"}`, params, users[0].ID, "user")
	if edited.Code != http.StatusOK || !strings.Contains(edited.Body.String(), "改过的内容") || !strings.Contains(edited.Body.String(), "editedAt") {
		t.Fatalf("edit returned %d: %s", edited.Code, edited.Body.String())
	}
	foreign := authenticatedRequest(community.UpdateComment, http.MethodPatch, "/api/comments/1", `{"content":"窃取"}`, params, users[1].ID, "user")
	if foreign.Code != http.StatusForbidden {
		t.Fatalf("foreign edit must be 403, got %d", foreign.Code)
	}
}

// 旧 slug：按版本历史 301 到当前 slug
func TestOldSlugRedirect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newCommunityTestDB(t)
	post := models.Post{AuthorID: 1, Title: "文章", Slug: "new-slug", Content: "body", Status: "published", PublishedAt: time.Now()}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.PostRevision{PostID: post.ID, Title: "文章", Slug: "old-slug", Content: "body", EditorID: 1}).Error; err != nil {
		t.Fatal(err)
	}
	controller := &PostController{DB: db}
	resp := performRequest(controller.Detail, http.MethodGet, "/api/posts/old-slug", "", gin.Params{{Key: "slug", Value: "old-slug"}})
	if resp.Code != http.StatusMovedPermanently {
		t.Fatalf("expected 301, got %d: %s", resp.Code, resp.Body.String())
	}
	if location := resp.Header().Get("Location"); location != "/api/posts/new-slug" {
		t.Fatalf("unexpected redirect target: %q", location)
	}
}

// 多类型搜索：文章之外同时返回话题与用户
func TestSearchIncludesTopicsAndUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newCommunityTestDB(t)
	SetupPostSearch(db)
	defer postSearchFTS.Store(false)
	user := models.User{Username: "searcher", Nickname: "检索者", Role: "user", Status: "active"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.ForumTopic{AuthorID: user.ID, Content: "聊聊检索这件事", Kind: "discuss", Status: "published"}).Error; err != nil {
		t.Fatal(err)
	}
	controller := &PostController{DB: db}
	// q=检索（%E6%A3%80%E7%B4%A2）
	resp := performRequest(controller.Search, http.MethodGet, "/api/search?q=%E6%A3%80%E7%B4%A2", "", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("search returned %d: %s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	if !strings.Contains(body, "聊聊检索这件事") {
		t.Fatalf("expected forum topic in results: %s", body)
	}
	if !strings.Contains(body, "searcher") {
		t.Fatalf("expected user in results: %s", body)
	}
}
