package models

import "time"

type Post struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Title       string    `json:"title" gorm:"size:180;not null"`
	Slug        string    `json:"slug" gorm:"size:180;uniqueIndex;not null"`
	Summary     string    `json:"summary" gorm:"type:text"`
	Content     string    `json:"content" gorm:"type:longtext;not null"`
	CoverImage  string    `json:"coverImage" gorm:"size:500"`
	Status      string    `json:"status" gorm:"size:20;index;not null;default:published"`
	Featured    bool      `json:"featured"`
	Views       int       `json:"views"`
	ReadingTime int       `json:"readingTime"`
	PublishedAt time.Time `json:"publishedAt"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Tags        []Tag     `json:"tags" gorm:"many2many:post_tags;"`
}

type Tag struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"size:80;uniqueIndex;not null"`
	Slug      string    `json:"slug" gorm:"size:100;uniqueIndex;not null"`
	CreatedAt time.Time `json:"createdAt"`
}

type User struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Username     string    `json:"username" gorm:"size:60;uniqueIndex;not null"`
	PasswordHash string    `json:"-" gorm:"size:255;not null"`
	Nickname     string    `json:"nickname" gorm:"size:100"`
	Avatar       string    `json:"avatar" gorm:"size:500"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}
