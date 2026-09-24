package controllers

import (
	"context"
	"log"
	"time"

	"quietsignal/backend/models"

	"gorm.io/gorm"
)

// 数据保留策略：验证码 15 分钟过期，留 24 小时兜底排查即可；
// 已读通知保留 90 天（未读不删）；管理审计日志保留 1 年。
const (
	emailCodeRetention    = 24 * time.Hour
	notificationRetention = 90 * 24 * time.Hour
	adminLogRetention     = 365 * 24 * time.Hour
	cleanupInterval       = 24 * time.Hour
)

// CleanupOldData 清理过期数据：这些表只增不减，长期运行会拖累相关查询
func CleanupOldData(db *gorm.DB, now time.Time) error {
	if err := db.Where("created_at < ?", now.Add(-emailCodeRetention)).Delete(&models.EmailLoginCode{}).Error; err != nil {
		return err
	}
	if err := db.Where("read_at IS NOT NULL AND read_at < ?", now.Add(-notificationRetention)).Delete(&models.Notification{}).Error; err != nil {
		return err
	}
	return db.Where("created_at < ?", now.Add(-adminLogRetention)).Delete(&models.AdminLog{}).Error
}

// RunDataCleanup 启动时先跑一次，之后每天清理一次过期数据与孤儿图片；
// 孤儿图片的清理由此统一承担，删文章路径不再同步扫描全部内容
func RunDataCleanup(ctx context.Context, db *gorm.DB, uploadDir string) {
	run := func() {
		if err := CleanupOldData(db, time.Now()); err != nil {
			log.Printf("cleanup old data: %v", err)
		}
		if uploadDir != "" {
			if _, err := cleanupUnreferencedUploads(uploadDir, db); err != nil {
				log.Printf("cleanup uploads: %v", err)
			}
		}
	}
	run()
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}
