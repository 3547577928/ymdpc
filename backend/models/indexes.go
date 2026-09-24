package models

import "gorm.io/gorm"

// EnsureIndexes 补充高频列表、计数和关联查询使用的组合索引。
// 使用 IF NOT EXISTS 兼容已有数据库，应用启动时可重复执行。
func EnsureIndexes(db *gorm.DB) error {
	statements := []string{
		`CREATE INDEX IF NOT EXISTS idx_posts_public_order ON posts(status, moderation_status, published_at, id)`,
		`CREATE INDEX IF NOT EXISTS idx_posts_scheduled_publish ON posts(status, scheduled_at, id)`,
		`CREATE INDEX IF NOT EXISTS idx_comments_post_status_created ON comments(post_id, status, created_at, id)`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_user_read_created ON notifications(user_id, read_at, created_at, id)`,
		`CREATE INDEX IF NOT EXISTS idx_follows_following_follower ON follows(following_id, follower_id)`,
		`CREATE INDEX IF NOT EXISTS idx_post_likes_post_user ON post_likes(post_id, user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_favorites_post_user ON favorites(post_id, user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_comment_likes_comment_user ON comment_likes(comment_id, user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_post_tags_tag_post ON post_tags(tag_id, post_id)`,
		`CREATE INDEX IF NOT EXISTS idx_forum_topics_public_order ON forum_topics(status, created_at, id)`,
		`CREATE INDEX IF NOT EXISTS idx_forum_topics_kind_order ON forum_topics(kind, status, created_at, id)`,
		`CREATE INDEX IF NOT EXISTS idx_forum_replies_topic_status_created ON forum_replies(topic_id, status, created_at, id)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
