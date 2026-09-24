package controllers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"quietsignal/backend/models"

	"github.com/gin-gonic/gin"
)

type TrendingTagDTO struct {
	ID        uint    `json:"id"`
	Name      string  `json:"name"`
	Slug      string  `json:"slug"`
	PostCount int64   `json:"postCount"`
	Score     float64 `json:"score"`
}

// TrendingTags 返回近一段时间内真实公开文章使用的热门标签。
func (ic *InteractionController) TrendingTags(c *gin.Context) {
	days := parseBoundedInt(c.Query("days"), 30, 1, 90)
	limit := parseBoundedInt(c.Query("limit"), 12, 1, 30)
	since := time.Now().AddDate(0, 0, -days)
	var rows []TrendingTagDTO
	query := ic.DB.Model(&models.Tag{}).
		Select(`tags.id, tags.name, tags.slug,
			COUNT(DISTINCT posts.id) AS post_count,
			COALESCE(SUM(posts.likes_count * 3 + posts.comments_count * 5 + posts.favorite_count * 2 + posts.views / 100.0), 0) AS score`).
		Joins("JOIN post_tags ON post_tags.tag_id = tags.id").
		Joins("JOIN posts ON posts.id = post_tags.post_id AND posts.status = ? AND posts.moderation_status = ? AND posts.published_at >= ?", "published", "normal", since).
		Group("tags.id, tags.name, tags.slug").
		Order("score DESC, post_count DESC, tags.name ASC").
		Limit(limit)
	if err := query.Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取热门标签失败"})
		return
	}
	if rows == nil {
		rows = make([]TrendingTagDTO, 0)
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": rows})
}

func parseBoundedInt(value string, fallback, min, max int) int {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < min {
		return fallback
	}
	if parsed > max {
		return max
	}
	return parsed
}
