package controllers

import (
	"testing"
	"time"

	"quietsignal/backend/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestPublishScheduledPosts(t *testing.T) {
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Post{}, &models.Follow{}, &models.Notification{}); err != nil {
		t.Fatal(err)
	}
	users := []models.User{{Username: "author", Nickname: "Author", Status: "active"}, {Username: "follower", Nickname: "Follower", Status: "active"}}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Follow{FollowerID: users[1].ID, FollowingID: users[0].ID}).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	scheduledAt := now.Add(-time.Minute)
	post := models.Post{AuthorID: users[0].ID, Title: "Scheduled", Slug: "scheduled", Content: "body", Status: "scheduled", ScheduledAt: &scheduledAt}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	if err := PublishScheduledPosts(db, now); err != nil {
		t.Fatal(err)
	}
	var stored models.Post
	if err := db.First(&stored, post.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != "published" || stored.ScheduledAt != nil || !stored.PublishedAt.Equal(scheduledAt) {
		t.Fatalf("scheduled post was not published correctly: %#v", stored)
	}
	var notifications int64
	if err := db.Model(&models.Notification{}).Where("user_id = ? AND type = ?", users[1].ID, "post").Count(&notifications).Error; err != nil {
		t.Fatal(err)
	}
	if notifications != 1 {
		t.Fatalf("expected one follower notification, got %d", notifications)
	}
	if err := PublishScheduledPosts(db, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.Notification{}).Where("user_id = ? AND type = ?", users[1].ID, "post").Count(&notifications).Error; err != nil {
		t.Fatal(err)
	}
	if notifications != 1 {
		t.Fatalf("scheduled publish must notify only once, got %d", notifications)
	}
}
