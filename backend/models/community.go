package models

import (
	"time"

	"gorm.io/gorm"
)

// Category 文章分类
type Category struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"size:80;uniqueIndex;not null"`
	Slug      string    `json:"slug" gorm:"size:100;uniqueIndex;not null"`
	CreatedAt time.Time `json:"createdAt"`
}

// Favorite 文章收藏，唯一索引防止重复收藏
type Favorite struct {
	UserID    uint      `json:"userId" gorm:"uniqueIndex:idx_favorite_pair"`
	PostID    uint      `json:"postId" gorm:"uniqueIndex:idx_favorite_pair"`
	Post      Post      `json:"-" gorm:"foreignKey:PostID"`
	CreatedAt time.Time `json:"createdAt"`
}

// CommentLike 评论点赞，唯一索引防止重复点赞
type CommentLike struct {
	UserID    uint      `json:"userId" gorm:"uniqueIndex:idx_comment_like"`
	CommentID uint      `json:"commentId" gorm:"uniqueIndex:idx_comment_like"`
	CreatedAt time.Time `json:"createdAt"`
}

// Notification 站内通知，Type 为 comment/reply/like/follow/post
type Notification struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	UserID     uint       `json:"userId" gorm:"index;not null"`
	ActorID    uint       `json:"actorId" gorm:"index;not null"`
	Actor      User       `json:"actor" gorm:"foreignKey:ActorID"`
	Type       string     `json:"type" gorm:"size:20;index;not null"`
	ResourceID uint       `json:"resourceId"`
	ReadAt     *time.Time `json:"readAt"`
	CreatedAt  time.Time  `json:"createdAt"`
}

// Report 举报，TargetType 为 post 或 comment，Status 为 pending/handled/dismissed
type Report struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	ReporterID uint       `json:"reporterId" gorm:"index;not null"`
	Reporter   User       `json:"reporter" gorm:"foreignKey:ReporterID"`
	TargetType string     `json:"targetType" gorm:"size:20;index;not null"`
	TargetID   uint       `json:"targetId" gorm:"index;not null"`
	Reason     string     `json:"reason" gorm:"size:500;not null"`
	Status     string     `json:"status" gorm:"size:20;index;not null;default:pending"`
	HandledBy  *uint      `json:"handledBy" gorm:"index"`
	Handler    *User      `json:"handler,omitempty" gorm:"foreignKey:HandledBy"`
	HandledAt  *time.Time `json:"handledAt" gorm:"index"`
	Resolution string     `json:"resolution" gorm:"size:500"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

// AdminLog 管理操作日志，记录封禁、删除、下架等操作
type AdminLog struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	AdminID    uint      `json:"adminId" gorm:"index;not null"`
	Admin      User      `json:"admin" gorm:"foreignKey:AdminID"`
	Action     string    `json:"action" gorm:"size:40;not null"`
	TargetType string    `json:"targetType" gorm:"size:20;not null"`
	TargetID   uint      `json:"targetId"`
	Detail     string    `json:"detail" gorm:"size:500"`
	CreatedAt  time.Time `json:"createdAt"`
}

// Setting 系统设置键值对：open_registration、comments_enabled
type Setting struct {
	Key   string `json:"key" gorm:"size:50;primaryKey"`
	Value string `json:"value" gorm:"size:20;not null"`
}

// GetSetting 读取设置值，未设置时返回默认值
func GetSetting(db *gorm.DB, key, fallback string) string {
	var setting Setting
	if err := db.Where("key = ?", key).First(&setting).Error; err != nil {
		return fallback
	}
	return setting.Value
}

// BoolSetting 以布尔语义读取设置值
func BoolSetting(db *gorm.DB, key string, fallback bool) bool {
	value := GetSetting(db, key, "")
	if value == "" {
		return fallback
	}
	return value == "true"
}

// SaveSetting 保存设置值（不存在则创建）
func SaveSetting(db *gorm.DB, key, value string) error {
	return db.Save(&Setting{Key: key, Value: value}).Error
}
