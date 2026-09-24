package models

import "time"

// ForumTopic 是独立于长文章的社区短帖，Kind 为 discuss/share/help/rant。
type ForumTopic struct {
	ID           uint              `json:"id" gorm:"primaryKey"`
	AuthorID     uint              `json:"authorId" gorm:"index;not null"`
	Author       User              `json:"author" gorm:"foreignKey:AuthorID"`
	Content      string            `json:"content" gorm:"type:text;not null"`
	Kind         string            `json:"kind" gorm:"size:20;index;not null;default:discuss"`
	Status       string            `json:"status" gorm:"size:20;index;not null;default:published"`
	LikesCount   int               `json:"likesCount"`
	RepliesCount int               `json:"repliesCount"`
	Images       []ForumTopicImage `json:"images" gorm:"foreignKey:TopicID;constraint:OnDelete:CASCADE"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
	EditedAt     *time.Time        `json:"editedAt,omitempty"`
}

type ForumTopicImage struct {
	ID       uint   `json:"id" gorm:"primaryKey"`
	TopicID  uint   `json:"topicId" gorm:"index;not null"`
	URL      string `json:"url" gorm:"size:500;not null"`
	Position int    `json:"position" gorm:"not null;default:0"`
}

type ForumReply struct {
	ID            uint       `json:"id" gorm:"primaryKey"`
	TopicID       uint       `json:"topicId" gorm:"index;not null"`
	Topic         ForumTopic `json:"-" gorm:"foreignKey:TopicID"`
	AuthorID      uint       `json:"authorId" gorm:"index;not null"`
	Author        User       `json:"author" gorm:"foreignKey:AuthorID"`
	ParentID      *uint      `json:"parentId" gorm:"index"`
	ReplyToUserID *uint      `json:"replyToUserId" gorm:"index"`
	Content       string     `json:"content" gorm:"type:text;not null"`
	Status        string     `json:"status" gorm:"size:20;index;not null;default:published"`
	LikesCount    int        `json:"likesCount"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	EditedAt      *time.Time `json:"editedAt,omitempty"`
}

type ForumTopicLike struct {
	UserID    uint      `json:"userId" gorm:"uniqueIndex:idx_forum_topic_like"`
	TopicID   uint      `json:"topicId" gorm:"uniqueIndex:idx_forum_topic_like"`
	CreatedAt time.Time `json:"createdAt"`
}

type ForumReplyLike struct {
	UserID    uint      `json:"userId" gorm:"uniqueIndex:idx_forum_reply_like"`
	ReplyID   uint      `json:"replyId" gorm:"uniqueIndex:idx_forum_reply_like"`
	CreatedAt time.Time `json:"createdAt"`
}
