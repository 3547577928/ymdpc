package controllers

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"quietsignal/backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 管理端用户与统计域：用户列表、封禁/禁言/角色调整、全站概览数据

type AdminUserDTO struct {
	UserDTO
	Status    string `json:"status"`
	PostCount int64  `json:"postCount"`
	LikeCount int64  `json:"likeCount"`
	Followers int64  `json:"followers"`
	Following int64  `json:"following"`
}

// AdminUsers 管理端用户列表，支持按用户名/昵称搜索
func (cc *CommunityController) AdminUsers(c *gin.Context) {
	page, pageSize := pagination(c)
	query := cc.DB.Model(&models.User{})
	if keyword := strings.TrimSpace(c.Query("q")); keyword != "" {
		like := likePattern(keyword)
		query = query.Where("username LIKE ? ESCAPE '\\' OR nickname LIKE ? ESCAPE '\\'", like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取用户失败"})
		return
	}
	var users []models.User
	if err := query.Order("created_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取用户失败"})
		return
	}
	if len(users) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"items": make([]AdminUserDTO, 0), "total": total, "page": page, "pageSize": pageSize}})
		return
	}
	// 逐用户调用 profileData 是每用户 6 次查询的 N+1（20 个用户上百次查询），
	// 这里按 ID 集合做四次分组统计，查询次数与用户数无关
	ids := make([]uint, 0, len(users))
	for _, user := range users {
		ids = append(ids, user.ID)
	}
	postCounts, err := countPostsByAuthor(cc.DB, ids)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取用户统计失败"})
		return
	}
	likeCounts, err := countLikesByAuthor(cc.DB, ids)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取用户统计失败"})
		return
	}
	followers, following, err := countFollowsByUser(cc.DB, ids)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取用户统计失败"})
		return
	}
	items := make([]AdminUserDTO, 0, len(users))
	for _, user := range users {
		items = append(items, AdminUserDTO{
			UserDTO:   toUserDTO(user),
			Status:    user.Status,
			PostCount: postCounts[user.ID],
			LikeCount: likeCounts[user.ID],
			Followers: followers[user.ID],
			Following: following[user.ID],
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"items": items, "total": total, "page": page, "pageSize": pageSize}})
}

func (cc *CommunityController) UpdateUser(c *gin.Context) {
	var input struct {
		Status string `json:"status"`
		Role   string `json:"role"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "用户状态格式错误"})
		return
	}
	var user models.User
	if err := cc.DB.First(&user, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}
	if user.ID == currentUserID(c) && input.Status == "banned" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不能封禁当前管理员"})
		return
	}
	updates := map[string]any{}
	if input.Status == "active" || input.Status == "muted" || input.Status == "banned" {
		updates["status"] = input.Status
	}
	if input.Role == "user" || input.Role == "admin" {
		updates["role"] = input.Role
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "用户状态无效"})
		return
	}
	adminID := currentUserID(c)
	if err := cc.DB.Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			updates["session_version"] = gorm.Expr("session_version + 1")
		}
		if err := tx.Model(&user).Updates(updates).Error; err != nil {
			return err
		}
		// 封禁、禁言、角色调整保留管理日志
		if status, ok := updates["status"]; ok {
			if err := writeAdminLog(tx, adminID, "user."+status.(string), "user", user.ID, user.Username); err != nil {
				return err
			}
		}
		if role, ok := updates["role"]; ok {
			return writeAdminLog(tx, adminID, "user.role."+role.(string), "user", user.ID, user.Username)
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新用户失败"})
		return
	}
	// 返回更新后的用户数据，供管理端列表原地刷新；重新读取确保 status/role 是最新值
	if err := cc.DB.First(&user, user.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取用户失败"})
		return
	}
	ids := []uint{user.ID}
	postCounts, err := countPostsByAuthor(cc.DB, ids)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取用户统计失败"})
		return
	}
	likeCounts, err := countLikesByAuthor(cc.DB, ids)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取用户统计失败"})
		return
	}
	followers, following, err := countFollowsByUser(cc.DB, ids)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取用户统计失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": AdminUserDTO{
		UserDTO:   toUserDTO(user),
		Status:    user.Status,
		PostCount: postCounts[user.ID],
		LikeCount: likeCounts[user.ID],
		Followers: followers[user.ID],
		Following: following[user.ID],
	}})
}

// countPostsByAuthor 批量统计每个作者的文章数（含草稿等全部状态），管理端列表使用
func countPostsByAuthor(db *gorm.DB, ids []uint) (map[uint]int64, error) {
	counts := map[uint]int64{}
	if len(ids) == 0 {
		return counts, nil
	}
	var rows []struct {
		AuthorID uint
		Count    int64
	}
	if err := db.Model(&models.Post{}).Select("author_id, COUNT(*) AS count").Where("author_id IN ?", ids).Group("author_id").Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		counts[row.AuthorID] = row.Count
	}
	return counts, nil
}

// countLikesByAuthor 批量统计每个作者的全部文章获赞数
func countLikesByAuthor(db *gorm.DB, ids []uint) (map[uint]int64, error) {
	counts := map[uint]int64{}
	if len(ids) == 0 {
		return counts, nil
	}
	var rows []struct {
		AuthorID uint
		Count    int64
	}
	if err := db.Model(&models.PostLike{}).
		Select("posts.author_id AS author_id, COUNT(*) AS count").
		Joins("JOIN posts ON posts.id = post_likes.post_id").
		Where("posts.author_id IN ?", ids).Group("posts.author_id").Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		counts[row.AuthorID] = row.Count
	}
	return counts, nil
}

// countFollowsByUser 批量统计每个用户的粉丝数与关注数
func countFollowsByUser(db *gorm.DB, ids []uint) (followers, following map[uint]int64, err error) {
	followers = map[uint]int64{}
	following = map[uint]int64{}
	if len(ids) == 0 {
		return followers, following, nil
	}
	var byFollowing []struct {
		FollowingID uint
		Count       int64
	}
	if err := db.Model(&models.Follow{}).Select("following_id, COUNT(*) AS count").Where("following_id IN ?", ids).Group("following_id").Scan(&byFollowing).Error; err != nil {
		return nil, nil, err
	}
	for _, row := range byFollowing {
		followers[row.FollowingID] = row.Count
	}
	var byFollower []struct {
		FollowerID uint
		Count      int64
	}
	if err := db.Model(&models.Follow{}).Select("follower_id, COUNT(*) AS count").Where("follower_id IN ?", ids).Group("follower_id").Scan(&byFollower).Error; err != nil {
		return nil, nil, err
	}
	for _, row := range byFollower {
		following[row.FollowerID] = row.Count
	}
	return followers, following, nil
}

// ActiveUserDTO 概览中的活跃用户条目
type ActiveUserDTO struct {
	User         UserDTO `json:"user"`
	PostCount    int64   `json:"postCount"`
	CommentCount int64   `json:"commentCount"`
}

type DailyMetricDTO struct {
	Date     string `json:"date"`
	Users    int64  `json:"users"`
	Posts    int64  `json:"posts"`
	Comments int64  `json:"comments"`
}

type TopPostDTO struct {
	ID            uint    `json:"id"`
	Title         string  `json:"title"`
	Slug          string  `json:"slug"`
	Views         int     `json:"views"`
	LikesCount    int     `json:"likesCount"`
	CommentsCount int     `json:"commentsCount"`
	Author        UserDTO `json:"author"`
}

// SiteStats 全站概览数据：规模、互动总量、今日新增与近 7 天活跃用户
type SiteStats struct {
	Total          int64            `json:"total"`
	Users          int64            `json:"users"`
	Posts          int64            `json:"posts"`
	Published      int64            `json:"published"`
	Views          int64            `json:"views"`
	Likes          int64            `json:"likes"`
	Comments       int64            `json:"comments"`
	Followers      int64            `json:"follows"`
	TodayUsers     int64            `json:"todayUsers"`
	TodayPosts     int64            `json:"todayPosts"`
	TodayComments  int64            `json:"todayComments"`
	PendingReports int64            `json:"pendingReports"`
	ActiveUsers    []ActiveUserDTO  `json:"activeUsers"`
	DailyMetrics   []DailyMetricDTO `json:"dailyMetrics"`
	TopPosts       []TopPostDTO     `json:"topPosts"`
}

// SiteStatsCache 统计结果进程内缓存：接口要串行执行十余条 COUNT/SUM，
// 后台每次进入/翻页都全量重算不划算，60 秒内直接返回缓存
type SiteStatsCache struct {
	mu      sync.Mutex
	data    *SiteStats
	expires time.Time
}

const siteStatsTTL = 60 * time.Second

func (cache *SiteStatsCache) Get() *SiteStats {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if cache.data != nil && time.Now().Before(cache.expires) {
		return cache.data
	}
	return nil
}

func (cache *SiteStatsCache) Store(stats *SiteStats) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.data = stats
	cache.expires = time.Now().Add(siteStatsTTL)
}

// UserStats 全站概览：规模、互动总量、今日新增与近 7 天活跃用户
func (cc *CommunityController) UserStats(c *gin.Context) {
	if cached := cc.StatsCache.Get(); cached != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": cached})
		return
	}
	var stats SiteStats
	defer func() {
		// 仅在成功响应时写入缓存
		if c.Writer.Status() == http.StatusOK {
			cc.StatsCache.Store(&stats)
		}
	}()
	if err := cc.DB.Model(&models.User{}).Count(&stats.Users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取统计失败"})
		return
	}
	if err := cc.DB.Model(&models.Post{}).Count(&stats.Posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取统计失败"})
		return
	}
	stats.Total = stats.Posts
	if err := cc.DB.Model(&models.Post{}).Where("status = ?", "published").Count(&stats.Published).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取统计失败"})
		return
	}
	if err := cc.DB.Model(&models.Post{}).Select("COALESCE(SUM(views), 0)").Scan(&stats.Views).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取统计失败"})
		return
	}
	if err := cc.DB.Model(&models.PostLike{}).Count(&stats.Likes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取统计失败"})
		return
	}
	if err := cc.DB.Model(&models.Comment{}).Where("status = ?", "published").Count(&stats.Comments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取统计失败"})
		return
	}
	if err := cc.DB.Model(&models.Follow{}).Count(&stats.Followers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取统计失败"})
		return
	}
	if err := cc.DB.Model(&models.Report{}).Where("status = ?", "pending").Count(&stats.PendingReports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取统计失败"})
		return
	}
	// 今日新增按最近 24 小时统计
	since := time.Now().Add(-24 * time.Hour)
	if err := cc.DB.Model(&models.User{}).Where("created_at >= ?", since).Count(&stats.TodayUsers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取统计失败"})
		return
	}
	if err := cc.DB.Model(&models.Post{}).Where("created_at >= ?", since).Count(&stats.TodayPosts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取统计失败"})
		return
	}
	if err := cc.DB.Model(&models.Comment{}).Where("created_at >= ?", since).Count(&stats.TodayComments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取统计失败"})
		return
	}
	// 活跃用户：近 7 天发文与评论合计最多的前 5 位
	week := time.Now().Add(-7 * 24 * time.Hour)
	var rows []struct {
		ID           uint
		Username     string
		Nickname     string
		Avatar       string
		CreatedAt    time.Time
		PostCount    int64
		CommentCount int64
	}
	if err := cc.DB.Raw(`SELECT u.id, u.username, u.nickname, u.avatar, u.created_at,
		(SELECT COUNT(*) FROM posts p WHERE p.author_id = u.id AND p.created_at >= ?) AS post_count,
		(SELECT COUNT(*) FROM comments cm WHERE cm.author_id = u.id AND cm.created_at >= ?) AS comment_count
		FROM users u ORDER BY post_count + comment_count DESC, u.id ASC LIMIT 5`, week, week).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取统计失败"})
		return
	}
	activeUsers := make([]ActiveUserDTO, 0, len(rows))
	for _, row := range rows {
		activeUsers = append(activeUsers, ActiveUserDTO{
			User:         UserDTO{ID: row.ID, Username: row.Username, Nickname: row.Nickname, Avatar: row.Avatar, CreatedAt: row.CreatedAt},
			PostCount:    row.PostCount,
			CommentCount: row.CommentCount,
		})
	}
	stats.ActiveUsers = activeUsers

	startDay := time.Now().AddDate(0, 0, -13)
	startDay = time.Date(startDay.Year(), startDay.Month(), startDay.Day(), 0, 0, 0, 0, startDay.Location())
	type metricRow struct {
		Date  string
		Count int64
	}
	loadMetric := func(model any) (map[string]int64, error) {
		var metricRows []metricRow
		if err := cc.DB.Model(model).Select("date(created_at) AS date, COUNT(*) AS count").Where("created_at >= ?", startDay).Group("date(created_at)").Scan(&metricRows).Error; err != nil {
			return nil, err
		}
		result := make(map[string]int64, len(metricRows))
		for _, row := range metricRows {
			result[row.Date] = row.Count
		}
		return result, nil
	}
	userMetrics, err := loadMetric(&models.User{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取趋势统计失败"})
		return
	}
	postMetrics, err := loadMetric(&models.Post{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取趋势统计失败"})
		return
	}
	commentMetrics, err := loadMetric(&models.Comment{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取趋势统计失败"})
		return
	}
	stats.DailyMetrics = make([]DailyMetricDTO, 0, 14)
	for offset := 0; offset < 14; offset++ {
		day := startDay.AddDate(0, 0, offset).Format("2006-01-02")
		stats.DailyMetrics = append(stats.DailyMetrics, DailyMetricDTO{Date: day, Users: userMetrics[day], Posts: postMetrics[day], Comments: commentMetrics[day]})
	}

	var topPosts []models.Post
	if err := cc.DB.Where("status = ? AND moderation_status = ?", "published", "normal").Preload("Author").Order("views DESC, likes_count DESC, comments_count DESC, id DESC").Limit(5).Find(&topPosts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取热门文章失败"})
		return
	}
	stats.TopPosts = make([]TopPostDTO, 0, len(topPosts))
	for _, post := range topPosts {
		stats.TopPosts = append(stats.TopPosts, TopPostDTO{ID: post.ID, Title: post.Title, Slug: post.Slug, Views: post.Views, LikesCount: post.LikesCount, CommentsCount: post.CommentsCount, Author: toUserDTO(post.Author)})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": stats})
}
