package models

import "time"

// EmailLoginCode 保存邮箱验证码的 HMAC，不保存可直接登录的明文验证码。
// UserID 为空时表示该邮箱尚未注册，验证成功后可按公开注册设置创建账号。
type EmailLoginCode struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	Email     string     `json:"email" gorm:"size:254;index;not null"`
	CodeHash  string     `json:"-" gorm:"size:64;not null"`
	UserID    *uint      `json:"userId" gorm:"index"`
	ExpiresAt time.Time  `json:"expiresAt" gorm:"index;not null"`
	UsedAt    *time.Time `json:"usedAt" gorm:"index"`
	Attempts  int        `json:"attempts" gorm:"not null;default:0"`
	RequestIP string     `json:"requestIp" gorm:"size:64"`
	CreatedAt time.Time  `json:"createdAt"`
}
