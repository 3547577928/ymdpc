package controllers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"quietsignal/backend/models"

	"gorm.io/gorm"
)

// ViewCounter 缓冲文章浏览量。浏览量是全站最热的写路径，逐次 UPDATE 在
// SQLite 单写者模型下会拖累其他写入；改为内存累计、定期批量刷盘。
// 同时按 文章+IP 做 6 小时去重，作为前端 sessionStorage 防刷之外的第二道防线，
// 防止直接调用接口刷量。
type ViewCounter struct {
	mu      sync.Mutex
	pending map[uint]int
	seen    map[string]time.Time
	db      *gorm.DB
}

// viewDedupWindow 同一 IP 对同一文章的重复浏览只计一次的窗口
const viewDedupWindow = 6 * time.Hour

// viewFlushInterval 浏览量增量批量刷盘的间隔
const viewFlushInterval = 30 * time.Second

func NewViewCounter(db *gorm.DB) *ViewCounter {
	return &ViewCounter{pending: map[uint]int{}, seen: map[string]time.Time{}, db: db}
}

// Add 记录一次浏览，返回本次是否计入（去重窗口内的重复浏览不计）
func (v *ViewCounter) Add(postID uint, ip string) bool {
	key := fmt.Sprintf("%d|%s", postID, ip)
	now := time.Now()
	v.mu.Lock()
	defer v.mu.Unlock()
	if last, ok := v.seen[key]; ok && now.Sub(last) < viewDedupWindow {
		return false
	}
	v.seen[key] = now
	v.pending[postID]++
	// 兜底清理：正常由 Flush 周期清理，这里防止海量 文章+IP 组合把 map 撑爆
	if len(v.seen) > 100_000 {
		for entry, at := range v.seen {
			if now.Sub(at) >= viewDedupWindow {
				delete(v.seen, entry)
			}
		}
	}
	return true
}

// Pending 返回某文章尚未刷盘的浏览增量，用于响应中估算当前阅读量
func (v *ViewCounter) Pending(postID uint) int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.pending[postID]
}

// Flush 把累计增量批量写回数据库；失败的增量放回待刷队列，下次重试
func (v *ViewCounter) Flush() error {
	v.mu.Lock()
	batch := v.pending
	v.pending = map[uint]int{}
	now := time.Now()
	for entry, at := range v.seen {
		if now.Sub(at) >= viewDedupWindow {
			delete(v.seen, entry)
		}
	}
	v.mu.Unlock()
	for postID, delta := range batch {
		if err := v.db.Model(&models.Post{}).Where("id = ?", postID).UpdateColumn("views", gorm.Expr("views + ?", delta)).Error; err != nil {
			v.mu.Lock()
			v.pending[postID] += delta
			v.mu.Unlock()
			return err
		}
	}
	return nil
}

// Run 周期刷盘直到 ctx 取消，退出前最后刷一次，避免关停丢失增量
func (v *ViewCounter) Run(ctx context.Context) {
	ticker := time.NewTicker(viewFlushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			_ = v.Flush()
			return
		case <-ticker.C:
			_ = v.Flush()
		}
	}
}
