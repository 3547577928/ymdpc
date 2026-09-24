package controllers

import (
	"net/http"
	"time"

	"quietsignal/backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthorDailyMetricDTO struct {
	Date     string `json:"date"`
	Posts    int64  `json:"posts"`
	Likes    int64  `json:"likes"`
	Comments int64  `json:"comments"`
}

type MyStatsDTO struct {
	Posts        int64                  `json:"posts"`
	Published    int64                  `json:"published"`
	Drafts       int64                  `json:"drafts"`
	Views        int64                  `json:"views"`
	Likes        int64                  `json:"likes"`
	Comments     int64                  `json:"comments"`
	Followers    int64                  `json:"followers"`
	DailyMetrics []AuthorDailyMetricDTO `json:"dailyMetrics"`
	TopPosts     []TopPostDTO           `json:"topPosts"`
}

// MyStats 作者数据看板：自己文章的总量指标、近 14 天互动趋势与热门文章。
// 查询模式复用管理端 UserStats（分组统计 + 按天聚合），范围收窄到当前用户
func (ic *InteractionController) MyStats(c *gin.Context) {
	userID := currentUserID(c)
	var stats MyStatsDTO

	type statusCount struct {
		Status string
		Count  int64
	}
	var statusCounts []statusCount
	if err := ic.DB.Model(&models.Post{}).Select("status", "COUNT(*) AS count").Where("author_id = ?", userID).Group("status").Scan(&statusCounts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取统计失败"})
		return
	}
	for _, row := range statusCounts {
		stats.Posts += row.Count
		switch row.Status {
		case "published":
			stats.Published = row.Count
		case "draft":
			stats.Drafts = row.Count
		}
	}
	if err := ic.DB.Model(&models.Post{}).Where("author_id = ?", userID).Select("COALESCE(SUM(views), 0)").Scan(&stats.Views).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取统计失败"})
		return
	}
	if err := ic.DB.Model(&models.PostLike{}).Joins("JOIN posts ON posts.id = post_likes.post_id").Where("posts.author_id = ?", userID).Count(&stats.Likes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取统计失败"})
		return
	}
	if err := ic.DB.Model(&models.Comment{}).Joins("JOIN posts ON posts.id = comments.post_id").Where("posts.author_id = ? AND comments.status = ?", userID, "published").Count(&stats.Comments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取统计失败"})
		return
	}
	if err := ic.DB.Model(&models.Follow{}).Where("following_id = ?", userID).Count(&stats.Followers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取统计失败"})
		return
	}

	// 近 14 天趋势：发布、获赞、被评论，按天聚合后补齐无数据的日期
	startDay := time.Now().AddDate(0, 0, -13).Truncate(24 * time.Hour)
	countByDay := func(query *gorm.DB, dateExpr string) (map[string]int64, error) {
		rows := []struct {
			Day   string
			Count int64
		}{}
		if err := query.Select(dateExpr+" AS day", "COUNT(*) AS count").Group("day").Scan(&rows).Error; err != nil {
			return nil, err
		}
		result := map[string]int64{}
		for _, row := range rows {
			result[row.Day] = row.Count
		}
		return result, nil
	}
	postMetrics, err := countByDay(ic.DB.Model(&models.Post{}).Where("author_id = ? AND status = ? AND published_at >= ?", userID, "published", startDay), "date(published_at)")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取趋势失败"})
		return
	}
	likeMetrics, err := countByDay(ic.DB.Model(&models.PostLike{}).Joins("JOIN posts ON posts.id = post_likes.post_id").Where("posts.author_id = ? AND post_likes.created_at >= ?", userID, startDay), "date(post_likes.created_at)")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取趋势失败"})
		return
	}
	commentMetrics, err := countByDay(ic.DB.Model(&models.Comment{}).Joins("JOIN posts ON posts.id = comments.post_id").Where("posts.author_id = ? AND comments.created_at >= ?", userID, startDay), "date(comments.created_at)")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取趋势失败"})
		return
	}
	stats.DailyMetrics = make([]AuthorDailyMetricDTO, 0, 14)
	for offset := 0; offset < 14; offset++ {
		day := startDay.AddDate(0, 0, offset).Format("2006-01-02")
		stats.DailyMetrics = append(stats.DailyMetrics, AuthorDailyMetricDTO{Date: day, Posts: postMetrics[day], Likes: likeMetrics[day], Comments: commentMetrics[day]})
	}

	var topPosts []models.Post
	if err := ic.DB.Where("author_id = ? AND status = ? AND moderation_status = ?", userID, "published", "normal").Order("views DESC, likes_count DESC, comments_count DESC, id DESC").Limit(5).Find(&topPosts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取热门文章失败"})
		return
	}
	stats.TopPosts = make([]TopPostDTO, 0, len(topPosts))
	for _, post := range topPosts {
		stats.TopPosts = append(stats.TopPosts, TopPostDTO{ID: post.ID, Title: post.Title, Slug: post.Slug, Views: post.Views, LikesCount: post.LikesCount, CommentsCount: post.CommentsCount})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": stats})
}
