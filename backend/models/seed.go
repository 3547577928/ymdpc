package models

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Seed(db *gorm.DB, username, password string) error {
	admin, err := ensureAdmin(db, username, password)
	if err != nil {
		return err
	}
	if err := db.Model(&User{}).Where("id = ?", admin.ID).Updates(map[string]any{"role": "admin", "status": "active"}).Error; err != nil {
		return err
	}
	// 新增 session_version 列时，已有用户的值为 0（default 只作用于新行），
	// 修正为 1 与签发逻辑使用的初始版本一致
	if err := db.Model(&User{}).Where("session_version = 0").Update("session_version", 1).Error; err != nil {
		return err
	}
	if err := removeLegacyDemoPosts(db, admin.ID); err != nil {
		return err
	}
	if err := db.Model(&Post{}).Where("reading_time IS NULL OR reading_time = 0").Update("reading_time", 1).Error; err != nil {
		return err
	}
	if err := db.Model(&Post{}).Where("likes_count IS NULL").Update("likes_count", 0).Error; err != nil {
		return err
	}
	if err := db.Model(&Post{}).Where("comments_count IS NULL").Update("comments_count", 0).Error; err != nil {
		return err
	}
	if err := db.Model(&Post{}).Where("author_id = 0").Update("author_id", admin.ID).Error; err != nil {
		return err
	}
	return nil
}

// removeLegacyDemoPosts 清理旧版本曾自动创建的演示文章。
// 只匹配固定 slug、标题和管理员作者，避免误删用户后来创建的同名真实文章。
func removeLegacyDemoPosts(db *gorm.DB, adminID uint) error {
	legacy := []struct{ slug, title string }{
		{slug: "quiet-interface", title: "把复杂系统写成安静的界面"},
		{slug: "go-api-without-overengineering", title: "用 Go 构建一个不急着扩张的 API"},
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, item := range legacy {
			var post Post
			if err := tx.Where("author_id = ? AND slug = ? AND title = ?", adminID, item.slug, item.title).First(&post).Error; err == gorm.ErrRecordNotFound {
				continue
			} else if err != nil {
				return err
			}

			var commentIDs []uint
			if err := tx.Model(&Comment{}).Where("post_id = ?", post.ID).Pluck("id", &commentIDs).Error; err != nil {
				return err
			}
			if len(commentIDs) > 0 {
				if err := tx.Where("comment_id IN ?", commentIDs).Delete(&CommentLike{}).Error; err != nil {
					return err
				}
				if err := tx.Where("target_type = ? AND target_id IN ?", "comment", commentIDs).Delete(&Report{}).Error; err != nil {
					return err
				}
				if err := tx.Where("type = ? AND resource_id IN ?", "reply", commentIDs).Delete(&Notification{}).Error; err != nil {
					return err
				}
			}
			if err := tx.Exec("DELETE FROM post_tags WHERE post_id = ?", post.ID).Error; err != nil {
				return err
			}
			if err := tx.Where("post_id = ?", post.ID).Delete(&Comment{}).Error; err != nil {
				return err
			}
			if err := tx.Where("post_id = ?", post.ID).Delete(&PostLike{}).Error; err != nil {
				return err
			}
			if err := tx.Where("post_id = ?", post.ID).Delete(&Favorite{}).Error; err != nil {
				return err
			}
			if err := tx.Where("post_id = ?", post.ID).Delete(&PostRevision{}).Error; err != nil {
				return err
			}
			if err := tx.Where("target_type = ? AND target_id = ?", "post", post.ID).Delete(&Report{}).Error; err != nil {
				return err
			}
			if err := tx.Where("type IN ? AND resource_id = ?", []string{"like", "comment", "post"}, post.ID).Delete(&Notification{}).Error; err != nil {
				return err
			}
			if err := tx.Delete(&post).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func ensureAdmin(db *gorm.DB, username, password string) (User, error) {
	var user User
	err := db.Where("username = ?", username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if hashErr != nil {
			return user, hashErr
		}
		user = User{Username: username, PasswordHash: string(hash), Nickname: "Yiming", Role: "admin", Status: "active"}
		return user, db.Create(&user).Error
	}
	if err != nil {
		return user, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) == nil {
		return user, nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return user, err
	}
	return user, db.Model(&user).Updates(map[string]any{"password_hash": string(hash), "role": "admin", "status": "active"}).Error
}
