package controllers

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"quietsignal/backend/models"
)

func newInteractionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "-") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Post{}, &models.Tag{}, &models.Comment{}, &models.PostLike{}, &models.Follow{}, &models.Category{}, &models.Favorite{}, &models.CommentLike{}, &models.Notification{}, &models.Report{}, &models.AdminLog{}, &models.Setting{}, &models.PostRevision{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestResolveCategoryReturnsDatabaseError(t *testing.T) {
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "-") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	categoryID := uint(1)
	valid, err := resolveCategory(db, &categoryID)
	if err == nil {
		t.Fatal("expected category lookup error when the categories table is missing")
	}
	if valid {
		t.Fatal("category should not be valid after a failed lookup")
	}
}

// TestFavoritePostFlow 验证收藏、重复收藏幂等与取消收藏
func TestFavoritePostFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newInteractionTestDB(t)
	users := []models.User{{Username: "author", Nickname: "Author", Role: "user", Status: "active"}, {Username: "alice", Nickname: "Alice", Role: "user", Status: "active"}}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	post := models.Post{AuthorID: users[0].ID, Title: "Post", Slug: "post", Content: "body", Status: "published", PublishedAt: time.Now()}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	ic := &InteractionController{DB: db}
	favoriteParams := gin.Params{{Key: "slug", Value: "post"}}

	favorite := authenticatedRequest(ic.FavoritePost, http.MethodPost, "/api/posts/post/favorite", "", favoriteParams, users[1].ID, "user")
	if favorite.Code != http.StatusOK {
		t.Fatalf("favorite returned %d: %s", favorite.Code, favorite.Body.String())
	}
	again := authenticatedRequest(ic.FavoritePost, http.MethodPost, "/api/posts/post/favorite", "", favoriteParams, users[1].ID, "user")
	if again.Code != http.StatusOK {
		t.Fatalf("duplicate favorite returned %d", again.Code)
	}
	var stored models.Post
	db.First(&stored, post.ID)
	if stored.FavoriteCount != 1 {
		t.Fatalf("expected favorite count 1, got %d", stored.FavoriteCount)
	}

	unfavorite := authenticatedRequest(ic.UnfavoritePost, http.MethodDelete, "/api/posts/post/favorite", "", favoriteParams, users[1].ID, "user")
	if unfavorite.Code != http.StatusOK {
		t.Fatalf("unfavorite returned %d", unfavorite.Code)
	}
	if !strings.Contains(unfavorite.Body.String(), `"favoriteCount":0`) {
		t.Fatalf("unexpected unfavorite response: %s", unfavorite.Body.String())
	}
	db.First(&stored, post.ID)
	if stored.FavoriteCount != 0 {
		t.Fatalf("expected favorite count 0 after unfavorite, got %d", stored.FavoriteCount)
	}
}

// TestCommentLikeAndNotificationFlow 验证评论点赞以及评论、关注产生的通知
func TestCommentLikeAndNotificationFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newInteractionTestDB(t)
	users := []models.User{{Username: "author", Nickname: "Author", Role: "user", Status: "active"}, {Username: "alice", Nickname: "Alice", Role: "user", Status: "active"}}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	post := models.Post{AuthorID: users[0].ID, Title: "Post", Slug: "post", Content: "body", Status: "published", PublishedAt: time.Now()}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	ic := &InteractionController{DB: db}
	cc := &CommunityController{DB: db}

	// alice 评论 author 的文章，author 应收到 comment 通知
	commentResp := authenticatedRequest(cc.CreateComment, http.MethodPost, "/api/posts/post/comments", `{"content":"hello"}`, gin.Params{{Key: "slug", Value: "post"}}, users[1].ID, "user")
	if commentResp.Code != http.StatusCreated {
		t.Fatalf("comment returned %d: %s", commentResp.Code, commentResp.Body.String())
	}
	var comment models.Comment
	if err := db.Where("content = ?", "hello").First(&comment).Error; err != nil {
		t.Fatal(err)
	}
	var notificationCount int64
	db.Model(&models.Notification{}).Where("user_id = ? AND type = ?", users[0].ID, "comment").Count(&notificationCount)
	if notificationCount != 1 {
		t.Fatalf("expected one comment notification, got %d", notificationCount)
	}

	// author 给 alice 的评论点赞
	like := authenticatedRequest(ic.LikeComment, http.MethodPost, "/api/comments/1/like", "", gin.Params{{Key: "id", Value: "1"}}, users[0].ID, "user")
	if like.Code != http.StatusOK {
		t.Fatalf("comment like returned %d: %s", like.Code, like.Body.String())
	}
	var likedComment models.Comment
	db.First(&likedComment, comment.ID)
	if likedComment.LikesCount != 1 {
		t.Fatalf("expected comment likes 1, got %d", likedComment.LikesCount)
	}
	unlike := authenticatedRequest(ic.UnlikeComment, http.MethodDelete, "/api/comments/1/like", "", gin.Params{{Key: "id", Value: "1"}}, users[0].ID, "user")
	if unlike.Code != http.StatusOK || !strings.Contains(unlike.Body.String(), `"likesCount":0`) {
		t.Fatalf("unexpected comment unlike response: %d %s", unlike.Code, unlike.Body.String())
	}
	db.First(&likedComment, comment.ID)
	if likedComment.LikesCount != 0 {
		t.Fatalf("expected comment likes 0 after unlike, got %d", likedComment.LikesCount)
	}

	// alice 关注 author，author 收到 follow 通知
	follow := authenticatedRequest(cc.FollowUser, http.MethodPost, "/api/users/1/follow", "", gin.Params{{Key: "id", Value: "1"}}, users[1].ID, "user")
	if follow.Code != http.StatusOK {
		t.Fatalf("follow returned %d", follow.Code)
	}
	db.Model(&models.Notification{}).Where("user_id = ? AND type = ?", users[0].ID, "follow").Count(&notificationCount)
	if notificationCount != 1 {
		t.Fatalf("expected one follow notification, got %d", notificationCount)
	}

	// 标记已读
	notifications := authenticatedRequest(ic.Notifications, http.MethodGet, "/api/notifications", "", nil, users[0].ID, "user")
	if notifications.Code != http.StatusOK || !strings.Contains(notifications.Body.String(), `"unread":2`) {
		t.Fatalf("unexpected notifications response: %d %s", notifications.Code, notifications.Body.String())
	}
	markRead := authenticatedRequest(ic.MarkNotificationRead, http.MethodPatch, "/api/notifications/1/read", "", gin.Params{{Key: "id", Value: "1"}}, users[0].ID, "user")
	if markRead.Code != http.StatusOK {
		t.Fatalf("mark read returned %d", markRead.Code)
	}
}

// TestReportFlow 验证举报提交、管理端处理与管理日志
func TestReportFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newInteractionTestDB(t)
	users := []models.User{{Username: "author", Nickname: "Author", Role: "admin", Status: "active"}, {Username: "alice", Nickname: "Alice", Role: "user", Status: "active"}}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	post := models.Post{AuthorID: users[0].ID, Title: "Bad post", Slug: "bad", Content: "body", Status: "published", PublishedAt: time.Now()}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	ic := &InteractionController{DB: db}

	reportResp := authenticatedRequest(ic.CreateReport, http.MethodPost, "/api/reports", `{"targetType":"post","targetId":1,"reason":"spam content"}`, nil, users[1].ID, "user")
	if reportResp.Code != http.StatusCreated {
		t.Fatalf("report returned %d: %s", reportResp.Code, reportResp.Body.String())
	}
	// 无效目标类型应被拒绝
	invalid := authenticatedRequest(ic.CreateReport, http.MethodPost, "/api/reports", `{"targetType":"user","targetId":1,"reason":"x"}`, nil, users[1].ID, "user")
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid target returned %d", invalid.Code)
	}

	adminList := authenticatedRequest(ic.AdminReports, http.MethodGet, "/api/admin/reports?status=pending", "", nil, users[0].ID, "admin")
	if adminList.Code != http.StatusOK || !strings.Contains(adminList.Body.String(), "spam content") {
		t.Fatalf("admin reports unexpected: %d %s", adminList.Code, adminList.Body.String())
	}
	handle := authenticatedRequest(ic.AdminHandleReport, http.MethodPatch, "/api/admin/reports/1", `{"status":"handled"}`, gin.Params{{Key: "id", Value: "1"}}, users[0].ID, "admin")
	if handle.Code != http.StatusOK {
		t.Fatalf("handle returned %d: %s", handle.Code, handle.Body.String())
	}
	var logCount int64
	db.Model(&models.AdminLog{}).Where("action = ?", "report.handled").Count(&logCount)
	if logCount != 1 {
		t.Fatalf("expected one admin log, got %d", logCount)
	}
}

// TestSettingsGates 验证关闭注册与关闭评论的系统设置生效
func TestSettingsGates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newInteractionTestDB(t)
	auth := &AuthController{DB: db}
	cc := &CommunityController{DB: db}

	if err := models.SaveSetting(db, "open_registration", "false"); err != nil {
		t.Fatal(err)
	}
	registerResp := performRequest(auth.Register, http.MethodPost, "/api/auth/register", `{"username":"newuser","password":"password123"}`, nil)
	if registerResp.Code != http.StatusForbidden {
		t.Fatalf("closed registration returned %d: %s", registerResp.Code, registerResp.Body.String())
	}

	users := []models.User{{Username: "author", Nickname: "Author", Role: "user", Status: "active"}}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	post := models.Post{AuthorID: users[0].ID, Title: "Post", Slug: "post", Content: "body", Status: "published", PublishedAt: time.Now()}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	if err := models.SaveSetting(db, "comments_enabled", "false"); err != nil {
		t.Fatal(err)
	}
	commentResp := authenticatedRequest(cc.CreateComment, http.MethodPost, "/api/posts/post/comments", `{"content":"hello"}`, gin.Params{{Key: "slug", Value: "post"}}, users[0].ID, "user")
	if commentResp.Code != http.StatusForbidden {
		t.Fatalf("disabled comments returned %d: %s", commentResp.Code, commentResp.Body.String())
	}
}

// TestCategoryCRUD 验证分类创建、文章关联与删除后引用置空
func TestCategoryCRUD(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newInteractionTestDB(t)
	users := []models.User{{Username: "author", Nickname: "Author", Role: "user", Status: "active"}}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	post := models.Post{AuthorID: users[0].ID, Title: "Post", Slug: "post", Content: "body", Status: "published", PublishedAt: time.Now()}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	ic := &InteractionController{DB: db}

	createResp := authenticatedRequest(ic.CreateCategory, http.MethodPost, "/api/admin/categories", `{"name":"工程"}`, nil, users[0].ID, "admin")
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create category returned %d: %s", createResp.Code, createResp.Body.String())
	}
	var payload struct {
		Data models.Category `json:"data"`
	}
	if err := json.Unmarshal(createResp.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}

	// 文章关联分类
	cc := &CommunityController{DB: db}
	updateResp := authenticatedRequest(cc.UpdatePost, http.MethodPut, "/api/posts/id/1", `{"title":"Post","content":"body","categoryId":1}`, gin.Params{{Key: "id", Value: "1"}}, users[0].ID, "user")
	if updateResp.Code != http.StatusOK {
		t.Fatalf("update post returned %d: %s", updateResp.Code, updateResp.Body.String())
	}
	// 不存在的分类应被拒绝
	badCategory := authenticatedRequest(cc.UpdatePost, http.MethodPut, "/api/posts/id/1", `{"title":"Post","content":"body","categoryId":99}`, gin.Params{{Key: "id", Value: "1"}}, users[0].ID, "user")
	if badCategory.Code != http.StatusBadRequest {
		t.Fatalf("invalid category returned %d", badCategory.Code)
	}

	deleteResp := authenticatedRequest(ic.DeleteCategory, http.MethodDelete, "/api/admin/categories/1", "", gin.Params{{Key: "id", Value: "1"}}, users[0].ID, "admin")
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("delete category returned %d", deleteResp.Code)
	}
	var stored models.Post
	db.First(&stored, post.ID)
	if stored.CategoryID != nil {
		t.Fatalf("expected category reference cleared, got %v", *stored.CategoryID)
	}
}
