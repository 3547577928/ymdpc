package controllers

import (
	"net/http"
	"strings"
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
		like := "%" + keyword + "%"
		query = query.Where("username LIKE ? OR nickname LIKE ?", like, like)
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
	items := make([]AdminUserDTO, 0, len(users))
	for _, user := range users {
		items = append(items, cc.adminUserData(user))
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
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": cc.adminUserData(user)})
}

// adminUserData 管理端视角的用户数据：包含草稿在内的全量文章数
func (cc *CommunityController) adminUserData(user models.User) AdminUserDTO {
	profile := cc.profileData(user)
	var postCount, likeCount int64
	cc.DB.Model(&models.Post{}).Where("author_id = ?", user.ID).Count(&postCount)
	cc.DB.Model(&models.PostLike{}).Joins("JOIN posts ON posts.id = post_likes.post_id").Where("posts.author_id = ?", user.ID).Count(&likeCount)
	return AdminUserDTO{UserDTO: profile.User, Status: user.Status, PostCount: postCount, LikeCount: likeCount, Followers: profile.Followers, Following: profile.Following}
}

// ActiveUserDTO 概览中的活跃用户条目
type ActiveUserDTO struct {
	User         UserDTO `json:"user"`
	PostCount    int64   `json:"postCount"`
	CommentCount int64   `json:"commentCount"`
}

// UserStats 全站概览：规模、互动总量、今日新增与近 7 天活跃用户
func (cc *CommunityController) UserStats(c *gin.Context) {
	var stats struct {
		Total         int64           `json:"total"`
		Users         int64           `json:"users"`
		Posts         int64           `json:"posts"`
		Published     int64           `json:"published"`
		Views         int64           `json:"views"`
		Likes         int64           `json:"likes"`
		Comments      int64           `json:"comments"`
		Followers     int64           `json:"follows"`
		TodayUsers    int64           `json:"todayUsers"`
		TodayPosts    int64           `json:"todayPosts"`
		TodayComments int64           `json:"todayComments"`
		ActiveUsers   []ActiveUserDTO `json:"activeUsers"`
	}
	if err := cc.DB.Model(&models.User{}).Count(&stats.Users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取统计失败"})
		return
	}
	cc.DB.Model(&models.Post{}).Count(&stats.Posts)
	stats.Total = stats.Posts
	cc.DB.Model(&models.Post{}).Where("status = ?", "published").Count(&stats.Published)
	cc.DB.Model(&models.Post{}).Select("COALESCE(SUM(views), 0)").Scan(&stats.Views)
	cc.DB.Model(&models.PostLike{}).Count(&stats.Likes)
	cc.DB.Model(&models.Comment{}).Where("status = ?", "published").Count(&stats.Comments)
	cc.DB.Model(&models.Follow{}).Count(&stats.Followers)
	// 今日新增按最近 24 小时统计
	since := time.Now().Add(-24 * time.Hour)
	cc.DB.Model(&models.User{}).Where("created_at >= ?", since).Count(&stats.TodayUsers)
	cc.DB.Model(&models.Post{}).Where("created_at >= ?", since).Count(&stats.TodayPosts)
	cc.DB.Model(&models.Comment{}).Where("created_at >= ?", since).Count(&stats.TodayComments)
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
	cc.DB.Raw(`SELECT u.id, u.username, u.nickname, u.avatar, u.created_at,
		(SELECT COUNT(*) FROM posts p WHERE p.author_id = u.id AND p.created_at >= ?) AS post_count,
		(SELECT COUNT(*) FROM comments cm WHERE cm.author_id = u.id AND cm.created_at >= ?) AS comment_count
		FROM users u ORDER BY post_count + comment_count DESC, u.id ASC LIMIT 5`, week, week).Scan(&rows)
	activeUsers := make([]ActiveUserDTO, 0, len(rows))
	for _, row := range rows {
		activeUsers = append(activeUsers, ActiveUserDTO{
			User:         UserDTO{ID: row.ID, Username: row.Username, Nickname: row.Nickname, Avatar: row.Avatar, CreatedAt: row.CreatedAt},
			PostCount:    row.PostCount,
			CommentCount: row.CommentCount,
		})
	}
	stats.ActiveUsers = activeUsers
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": stats})
}
