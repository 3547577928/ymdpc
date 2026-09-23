package models

import "time"

// MagicLinkToken 保存邮箱魔法链接的哈希，不保存可直接登录的原始 token。
// UserID 为空时表示该邮箱尚未注册，验证成功后可按公开注册设置创建账号。
type MagicLinkToken struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	Email     string     `json:"email" gorm:"size:254;index;not null"`
	TokenHash string     `json:"-" gorm:"size:64;uniqueIndex;not null"`
	UserID    *uint      `json:"userId" gorm:"index"`
	ExpiresAt time.Time  `json:"expiresAt" gorm:"index;not null"`
	UsedAt    *time.Time `json:"usedAt" gorm:"index"`
	RequestIP string     `json:"requestIp" gorm:"size:64"`
	CreatedAt time.Time  `json:"createdAt"`
}
