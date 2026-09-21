package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"quietsignal/backend/models"
)

func newPostTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "-") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Post{}, &models.Tag{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func performRequest(handler gin.HandlerFunc, method, target, body string, params gin.Params) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(method, target, strings.NewReader(body))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = params
	handler(context)
	return recorder
}

func TestListOmitsContentAndDetailIncludesAdjacentPosts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newPostTestDB(t)
	base := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	posts := []models.Post{
		{Title: "Older", Slug: "older", Content: "older body", Status: "published", PublishedAt: base.Add(-time.Hour)},
		{Title: "Current", Slug: "current", Content: "current body", Status: "published", PublishedAt: base},
		{Title: "Newer", Slug: "newer", Content: "newer body", Status: "published", PublishedAt: base.Add(time.Hour)},
		{Title: "Draft", Slug: "draft", Content: "private body", Status: "draft"},
	}
	if err := db.Create(&posts).Error; err != nil {
		t.Fatal(err)
	}
	controller := &PostController{DB: db}

	listResponse := performRequest(controller.List, http.MethodGet, "/api/posts?pageSize=50", "", nil)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list returned %d: %s", listResponse.Code, listResponse.Body.String())
	}
	var listPayload struct {
		Data struct {
			Items []map[string]json.RawMessage `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listResponse.Body.Bytes(), &listPayload); err != nil {
		t.Fatal(err)
	}
	if len(listPayload.Data.Items) != 3 {
		t.Fatalf("expected 3 public posts, got %d", len(listPayload.Data.Items))
	}
	for _, item := range listPayload.Data.Items {
		if _, exists := item["content"]; exists {
			t.Fatal("list response must not include article content")
		}
	}

	detailResponse := performRequest(controller.Detail, http.MethodGet, "/api/posts/current", "", gin.Params{{Key: "slug", Value: "current"}})
	if detailResponse.Code != http.StatusOK {
		t.Fatalf("detail returned %d: %s", detailResponse.Code, detailResponse.Body.String())
	}
	var detailPayload struct {
		Data PostDetailDTO `json:"data"`
	}
	if err := json.Unmarshal(detailResponse.Body.Bytes(), &detailPayload); err != nil {
		t.Fatal(err)
	}
	if detailPayload.Data.Post.Content != "current body" {
		t.Fatal("detail response omitted article content")
	}
	if detailPayload.Data.Previous == nil || detailPayload.Data.Previous.Slug != "older" {
		t.Fatalf("unexpected previous post: %#v", detailPayload.Data.Previous)
	}
	if detailPayload.Data.Next == nil || detailPayload.Data.Next.Slug != "newer" {
		t.Fatalf("unexpected next post: %#v", detailPayload.Data.Next)
	}
}

func TestRecordViewAndStatusValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newPostTestDB(t)
	publishedAt := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	post := models.Post{Title: "Post", Slug: "post", Content: "body", Status: "published", Views: 3, PublishedAt: publishedAt}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	controller := &PostController{DB: db}

	viewResponse := performRequest(controller.RecordView, http.MethodPost, "/api/posts/post/views", "", gin.Params{{Key: "slug", Value: "post"}})
	if viewResponse.Code != http.StatusOK {
		t.Fatalf("record view returned %d: %s", viewResponse.Code, viewResponse.Body.String())
	}
	if err := db.First(&post, post.ID).Error; err != nil {
		t.Fatal(err)
	}
	if post.Views != 4 {
		t.Fatalf("expected 4 views, got %d", post.Views)
	}

	invalidResponse := performRequest(controller.UpdateStatus, http.MethodPatch, "/api/admin/posts/1/status", `{"status":"publishd"}`, gin.Params{{Key: "id", Value: "1"}})
	if invalidResponse.Code != http.StatusBadRequest {
		t.Fatalf("invalid status returned %d", invalidResponse.Code)
	}

	validResponse := performRequest(controller.UpdateStatus, http.MethodPatch, "/api/admin/posts/1/status", `{"status":"published"}`, gin.Params{{Key: "id", Value: "1"}})
	if validResponse.Code != http.StatusOK {
		t.Fatalf("valid status returned %d: %s", validResponse.Code, validResponse.Body.String())
	}
	if err := db.First(&post, post.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !post.PublishedAt.Equal(publishedAt) {
		t.Fatalf("published date changed from %v to %v", publishedAt, post.PublishedAt)
	}
}
