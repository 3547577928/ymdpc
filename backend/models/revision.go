package models

import "time"

// PostRevision 保存文章每次更新前的内容快照，便于作者回看和恢复。
type PostRevision struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	PostID     uint      `json:"postId" gorm:"index;not null"`
	EditorID   uint      `json:"editorId" gorm:"index;not null"`
	Title      string    `json:"title" gorm:"size:180;not null"`
	Slug       string    `json:"slug" gorm:"size:180;not null"`
	Summary    string    `json:"summary" gorm:"type:text"`
	Content    string    `json:"content" gorm:"type:longtext;not null"`
	CoverImage string    `json:"coverImage" gorm:"size:500"`
	CategoryID *uint     `json:"categoryId" gorm:"index"`
	Featured   bool      `json:"featured"`
	TagsJSON   string    `json:"-" gorm:"type:text;not null"`
	CreatedAt  time.Time `json:"createdAt" gorm:"index"`
}
