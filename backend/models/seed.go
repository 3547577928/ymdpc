package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Seed(db *gorm.DB, username, password string) error {
	if err := db.Model(&Post{}).Where("reading_time = 0").Update("reading_time", 1).Error; err != nil {
		return err
	}
	var count int64
	if err := db.Model(&Post{}).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		posts := []Post{
			{Title: "把复杂系统写成安静的界面", Slug: "quiet-interface", Summary: "从信息密度、层次与反馈开始，记录我对工具型产品界面设计的几条判断。", Content: "# 把复杂系统写成安静的界面\n\n好的界面并不是把所有信息都藏起来，而是让用户在需要它的时候，能够准确地找到它。", Status: "published", Featured: true, Views: 1842, ReadingTime: 6, PublishedAt: time.Now().AddDate(0, 0, -6)},
			{Title: "用 Go 构建一个不急着扩张的 API", Slug: "go-api-without-overengineering", Summary: "从路由、服务层到数据访问，分享一个个人项目里更容易维护的 Go API 组织方式。", Content: "# 用 Go 构建一个不急着扩张的 API\n\n个人项目最容易遇到的问题不是代码不够多，而是抽象提前了。", Status: "published", Views: 986, ReadingTime: 8, PublishedAt: time.Now().AddDate(0, 0, -14)},
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

	var user User
	if err := db.Where("username = ?", username).First(&user).Error; err == gorm.ErrRecordNotFound {
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if hashErr != nil {
			return hashErr
		}
		return db.Create(&User{Username: username, PasswordHash: string(hash), Nickname: "Yiming"}).Error
	}
	return nil
}
