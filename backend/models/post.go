package models

import "time"

type Post struct {
	ID         uint   `json:"id" gorm:"primaryKey"`
	AuthorID   uint   `json:"authorId" gorm:"index;not null;default:1"`
	Author     User   `json:"author" gorm:"foreignKey:AuthorID"`
	Title      string `json:"title" gorm:"size:180;not null"`
	Slug       string `json:"slug" gorm:"size:180;uniqueIndex;not null"`
	Summary    string `json:"summary" gorm:"type:text"`
	Content    string `json:"content" gorm:"type:longtext;not null"`
	CoverImage string `json:"coverImage" gorm:"size:500"`
	Status     string `json:"status" gorm:"size:20;index;not null;default:published"`
	// ModerationStatus 管理员审核状态：normal 正常，hidden 下架隐藏
	ModerationStatus string     `json:"moderationStatus" gorm:"size:20;index;not null;default:normal"`
	CategoryID       *uint      `json:"categoryId" gorm:"index"`
	Category         *Category  `json:"category" gorm:"foreignKey:CategoryID"`
	// SeriesID 文章所属系列；空表示不属于任何系列
	SeriesID *uint  `json:"seriesId" gorm:"index"`
	Series   *Series `json:"series,omitempty" gorm:"foreignKey:SeriesID"`
	Featured         bool       `json:"featured"`
	Views            int        `json:"views"`
	LikesCount       int        `json:"likesCount"`
	FavoriteCount    int        `json:"favoriteCount"`
	CommentsCount    int        `json:"commentsCount"`
	ReadingTime      int        `json:"readingTime"`
	PublishedAt      time.Time  `json:"publishedAt"`
	ScheduledAt      *time.Time `json:"scheduledAt" gorm:"index"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
	Tags             []Tag      `json:"tags" gorm:"many2many:post_tags;"`
}

type Tag struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"size:80;uniqueIndex;not null"`
	Slug      string    `json:"slug" gorm:"size:100;uniqueIndex;not null"`
	CreatedAt time.Time `json:"createdAt"`
}

// Series 文章系列/专栏：同一作者的一组有序文章
type Series struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	AuthorID    uint      `json:"authorId" gorm:"index;not null"`
	Author      User      `json:"author" gorm:"foreignKey:AuthorID"`
	Title       string    `json:"title" gorm:"size:120;not null"`
	Slug        string    `json:"slug" gorm:"size:140;uniqueIndex;not null"`
	Description string    `json:"description" gorm:"size:500"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// TagSubscription 标签订阅：订阅的标签有新公开文章时收到站内通知
type TagSubscription struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"userId" gorm:"not null;uniqueIndex:idx_tag_subscriptions_user_tag"`
	TagID     uint      `json:"tagId" gorm:"not null;uniqueIndex:idx_tag_subscriptions_user_tag"`
	CreatedAt time.Time `json:"createdAt"`
}

type User struct {
	ID           uint   `json:"id" gorm:"primaryKey"`
	Username     string `json:"username" gorm:"size:60;uniqueIndex;not null"`
	Email        string `json:"email" gorm:"size:254;index"`
	PasswordHash string `json:"-" gorm:"size:255;not null"`
	Nickname     string `json:"nickname" gorm:"size:100"`
	Avatar       string `json:"avatar" gorm:"size:500"`
	Bio          string `json:"bio" gorm:"size:500"`
	Role         string `json:"role" gorm:"size:20;index;not null;default:user"`
	Status       string `json:"status" gorm:"size:20;index;not null;default:active"`
	// SessionVersion 会话版本号：签发 JWT 时写入 claims，改密码或退出登录时递增，
	// 使该操作之前签发的 token 立即失效，弥补 JWT 24 小时有效期内无法作废的问题
	SessionVersion int       `json:"-" gorm:"not null;default:1"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type Comment struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	PostID        uint      `json:"postId" gorm:"index;not null"`
	Post          Post      `json:"-" gorm:"foreignKey:PostID"`
	AuthorID      uint      `json:"authorId" gorm:"index;not null"`
	Author        User      `json:"author" gorm:"foreignKey:AuthorID"`
	ParentID      *uint     `json:"parentId" gorm:"index"`
	ReplyToUserID *uint     `json:"replyToUserId"`
	Content       string    `json:"content" gorm:"type:text;not null"`
	Status        string    `json:"status" gorm:"size:20;index;not null;default:published"`
	Pinned        bool      `json:"pinned" gorm:"index"`
	LikesCount    int       `json:"likesCount"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	// EditedAt 内容最后一次被编辑的时间；仅编辑内容时更新，置顶/审核不影响
	EditedAt *time.Time `json:"editedAt,omitempty"`
}

type PostLike struct {
	UserID    uint      `json:"userId" gorm:"uniqueIndex:idx_post_like"`
	PostID    uint      `json:"postId" gorm:"uniqueIndex:idx_post_like"`
	CreatedAt time.Time `json:"createdAt"`
}

type Follow struct {
	FollowerID  uint      `json:"followerId" gorm:"uniqueIndex:idx_follow_pair"`
	FollowingID uint      `json:"followingId" gorm:"uniqueIndex:idx_follow_pair"`
	CreatedAt   time.Time `json:"createdAt"`
}
