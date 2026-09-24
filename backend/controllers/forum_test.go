package controllers

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"quietsignal/backend/models"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newForumTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "-") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.ForumTopic{}, &models.ForumTopicImage{}, &models.ForumReply{}, &models.ForumTopicLike{}, &models.ForumReplyLike{}, &models.Notification{}, &models.Report{}, &models.Setting{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestForumTopicReplyAndLikeFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newForumTestDB(t)
	users := []models.User{{Username: "alice", Nickname: "Alice", Role: "user", Status: "active"}, {Username: "bob", Nickname: "Bob", Role: "user", Status: "active"}}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	forum := &ForumController{DB: db}

	created := authenticatedRequest(forum.CreateTopic, http.MethodPost, "/api/forum", `{"content":"有人做过 SQLite 全文搜索吗？","kind":"help","images":["/uploads/a.png"]}`, nil, users[0].ID, "user")
	if created.Code != http.StatusCreated {
		t.Fatalf("create topic returned %d: %s", created.Code, created.Body.String())
	}
	var createdPayload struct {
		Data ForumTopicDTO `json:"data"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &createdPayload); err != nil {
		t.Fatal(err)
	}
	if createdPayload.Data.Kind != "help" || len(createdPayload.Data.Images) != 1 {
		t.Fatalf("unexpected topic: %#v", createdPayload.Data)
	}

	liked := authenticatedRequest(forum.LikeTopic, http.MethodPost, "/api/forum/1/like", "", gin.Params{{Key: "id", Value: "1"}}, users[1].ID, "user")
	if liked.Code != http.StatusOK || !strings.Contains(liked.Body.String(), `"likesCount":1`) {
		t.Fatalf("like topic returned %d: %s", liked.Code, liked.Body.String())
	}
	replied := authenticatedRequest(forum.CreateReply, http.MethodPost, "/api/forum/1/replies", `{"content":"可以使用 FTS5。"}`, gin.Params{{Key: "id", Value: "1"}}, users[1].ID, "user")
	if replied.Code != http.StatusCreated {
		t.Fatalf("create reply returned %d: %s", replied.Code, replied.Body.String())
	}

	detail := authenticatedRequest(forum.TopicDetail, http.MethodGet, "/api/forum/1", "", gin.Params{{Key: "id", Value: "1"}}, users[0].ID, "user")
	if detail.Code != http.StatusOK || !strings.Contains(detail.Body.String(), "FTS5") || !strings.Contains(detail.Body.String(), `"repliesCount":1`) {
		t.Fatalf("topic detail returned %d: %s", detail.Code, detail.Body.String())
	}
	var notifications int64
	if err := db.Model(&models.Notification{}).Where("user_id = ? AND type IN ?", users[0].ID, []string{"forum_reply", "forum_like"}).Count(&notifications).Error; err != nil {
		t.Fatal(err)
	}
	if notifications != 2 {
		t.Fatalf("expected two forum notifications, got %d", notifications)
	}
	interactions := &InteractionController{DB: db}
	notificationList := authenticatedRequest(interactions.Notifications, http.MethodGet, "/api/notifications?type=forum_reply", "", nil, users[0].ID, "user")
	if notificationList.Code != http.StatusOK || !strings.Contains(notificationList.Body.String(), `"resourceSlug":"1"`) {
		t.Fatalf("forum notification missing topic link: %d %s", notificationList.Code, notificationList.Body.String())
	}
}

func TestForumValidationAndReport(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newForumTestDB(t)
	users := []models.User{{Username: "alice", Nickname: "Alice", Role: "user", Status: "active"}, {Username: "bob", Nickname: "Bob", Role: "user", Status: "active"}}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	forum := &ForumController{DB: db}
	bad := authenticatedRequest(forum.CreateTopic, http.MethodPost, "/api/forum", `{"content":"x","kind":"unknown"}`, nil, users[0].ID, "user")
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("invalid kind returned %d", bad.Code)
	}
	if err := db.Create(&models.ForumTopic{AuthorID: users[0].ID, Content: "需要举报的帖子", Kind: "rant", Status: "published"}).Error; err != nil {
		t.Fatal(err)
	}
	interactions := &InteractionController{DB: db}
	report := authenticatedRequest(interactions.CreateReport, http.MethodPost, "/api/reports", `{"targetType":"forum_topic","targetId":1,"reason":"违规内容"}`, nil, users[1].ID, "user")
	if report.Code != http.StatusCreated {
		t.Fatalf("forum report returned %d: %s", report.Code, report.Body.String())
	}
}
