package models

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestEnsureIndexes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:indexes-test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &Post{}, &Tag{}, &Comment{}, &PostLike{}, &Follow{}, &Category{}, &Favorite{}, &CommentLike{}, &Notification{}, &Report{}, &AdminLog{}, &Setting{}); err != nil {
		t.Fatal(err)
	}
	if err := EnsureIndexes(db); err != nil {
		t.Fatal(err)
	}

	var indexNames []string
	if err := db.Raw("SELECT name FROM sqlite_master WHERE type = 'index' AND name IN (?, ?, ?, ?)",
		"idx_posts_public_order", "idx_comments_post_status_created", "idx_notifications_user_read_created", "idx_post_tags_tag_post").Scan(&indexNames).Error; err != nil {
		t.Fatal(err)
	}
	if len(indexNames) != 4 {
		t.Fatalf("expected four application indexes, got %v", indexNames)
	}
}
