package models

import (
	"errors"
	"time"

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
	if err := db.Model(&Post{}).Where("reading_time IS NULL OR reading_time = 0").Update("reading_time", 1).Error; err != nil {
		return err
	}
	if err := db.Model(&Post{}).Where("likes_count IS NULL").Update("likes_count", 0).Error; err != nil {
		return err
	}
	if err := db.Model(&Post{}).Where("comments_count IS NULL").Update("comments_count", 0).Error; err != nil {
		return err
	}
	var count int64
	if err := db.Model(&Post{}).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		posts := []Post{
			{AuthorID: admin.ID, Title: "把复杂系统写成安静的界面", Slug: "quiet-interface", Summary: "从信息密度、层次与反馈开始，记录我对工具型产品界面设计的几条判断。", Content: "# 把复杂系统写成安静的界面\n\n好的界面并不是把所有信息都藏起来，而是让用户在需要它的时候，能够准确地找到它。", Status: "published", Featured: true, Views: 1842, ReadingTime: 6, PublishedAt: time.Now().AddDate(0, 0, -6)},
			{AuthorID: admin.ID, Title: "用 Go 构建一个不急着扩张的 API", Slug: "go-api-without-overengineering", Summary: "从路由、服务层到数据访问，分享一个个人项目里更容易维护的 Go API 组织方式。", Content: "# 用 Go 构建一个不急着扩张的 API\n\n个人项目最容易遇到的问题不是代码不够多，而是抽象提前了。", Status: "published", Views: 986, ReadingTime: 8, PublishedAt: time.Now().AddDate(0, 0, -14)},
		}
		if err := db.Create(&posts).Error; err != nil {
			return err
		}
		design := Tag{Name: "设计系统", Slug: "设计系统"}
		backend := Tag{Name: "后端工程", Slug: "后端工程"}
		if err := db.Where("name = ?", design.Name).FirstOrCreate(&design).Error; err != nil {
			return err
		}
		if err := db.Where("name = ?", backend.Name).FirstOrCreate(&backend).Error; err != nil {
			return err
		}
		if err := db.Model(&posts[0]).Association("Tags").Append(&design); err != nil {
			return err
		}
		if err := db.Model(&posts[1]).Association("Tags").Append(&backend); err != nil {
			return err
		}
	}

	if err := db.Model(&Post{}).Where("author_id = 0").Update("author_id", admin.ID).Error; err != nil {
		return err
	}
	return nil
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
