package controllers

import (
	"context"
	"time"

	"quietsignal/backend/models"

	"gorm.io/gorm"
)

// PublishScheduledPosts 发布所有已到计划时间的文章。状态更新带 scheduled 条件，
// 即使多实例同时扫描，也只有成功抢到状态更新的实例会发送关注通知。
func PublishScheduledPosts(db *gorm.DB, now time.Time) error {
	var posts []models.Post
	if err := db.Where("status = ? AND scheduled_at IS NOT NULL AND scheduled_at <= ?", "scheduled", now).Order("scheduled_at ASC, id ASC").Find(&posts).Error; err != nil {
		return err
	}
	for _, post := range posts {
		if err := db.Transaction(func(tx *gorm.DB) error {
			publishedAt := now
			if post.ScheduledAt != nil {
				publishedAt = *post.ScheduledAt
			}
			result := tx.Exec("UPDATE posts SET status = ?, published_at = ?, scheduled_at = NULL WHERE id = ? AND status = ?", "published", publishedAt, post.ID, "scheduled")
			if result.Error != nil || result.RowsAffected == 0 {
				return result.Error
			}
			return notifyFollowersOfPost(tx, post.AuthorID, post.ID)
		}); err != nil {
			return err
		}
	}
	return nil
}

func RunScheduledPublisher(ctx context.Context, db *gorm.DB) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			_ = PublishScheduledPosts(db, now)
		}
	}
}
