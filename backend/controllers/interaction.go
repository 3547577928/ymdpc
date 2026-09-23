package controllers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"quietsignal/backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// InteractionController 收藏、评论点赞、通知、举报、我的内容与系统设置
type InteractionController struct{ DB *gorm.DB }

type NotificationDTO struct {
	ID            uint      `json:"id"`
	Type          string    `json:"type"`
	ResourceID    uint      `json:"resourceId"`
	ResourceSlug  string    `json:"resourceSlug,omitempty"`
	ResourceTitle string    `json:"resourceTitle,omitempty"`
	Read          bool      `json:"read"`
	CreatedAt     time.Time `json:"createdAt"`
	Actor         UserDTO   `json:"actor"`
}

type AdminReportDTO struct {
	ID            uint      `json:"id"`
	TargetType    string    `json:"targetType"`
	TargetID      uint      `json:"targetId"`
	TargetSummary string    `json:"targetSummary"`
	Reason        string    `json:"reason"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
	Reporter      UserDTO   `json:"reporter"`
}

type AdminLogDTO struct {
	ID         uint      `json:"id"`
	Action     string    `json:"action"`
	TargetType string    `json:"targetType"`
	TargetID   uint      `json:"targetId"`
	Detail     string    `json:"detail"`
	CreatedAt  time.Time `json:"createdAt"`
	Admin      UserDTO   `json:"admin"`
}

// createNotification 写入一条通知，调用方需保证接收者不是操作者本人
func createNotification(db *gorm.DB, userID, actorID uint, notificationType string, resourceID uint) error {
	return db.Create(&models.Notification{UserID: userID, ActorID: actorID, Type: notificationType, ResourceID: resourceID}).Error
}

// notifyFollowersOfPost 在文章首次发布时通知作者的全部粉丝。
func notifyFollowersOfPost(db *gorm.DB, authorID, postID uint) error {
	var followerIDs []uint
	if err := db.Model(&models.Follow{}).Where("following_id = ?", authorID).Pluck("follower_id", &followerIDs).Error; err != nil {
		return err
	}
	if len(followerIDs) == 0 {
		return nil
	}
	notifications := make([]models.Notification, 0, len(followerIDs))
	for _, followerID := range followerIDs {
		if followerID != authorID {
			notifications = append(notifications, models.Notification{UserID: followerID, ActorID: authorID, Type: "post", ResourceID: postID})
		}
	}
	if len(notifications) == 0 {
		return nil
	}
	return db.Create(&notifications).Error
}

// writeAdminLog 写入管理操作日志，保留封禁、删除、下架等操作的审核记录
func writeAdminLog(db *gorm.DB, adminID uint, action, targetType string, targetID uint, detail string) error {
	return db.Create(&models.AdminLog{AdminID: adminID, Action: action, TargetType: targetType, TargetID: targetID, Detail: detail}).Error
}

// deletePostRelations 删除文章时清理关联数据：评论点赞、回复通知、评论、文章点赞、收藏与站内通知
func deletePostRelations(tx *gorm.DB, postID uint) error {
	if err := tx.Where("post_id = ?", postID).Delete(&models.PostRevision{}).Error; err != nil {
		return err
	}
	// 顺序敏感：reply 通知的资源 id 指向评论，必须在删除评论之前清理，
	// 否则评论已经不存在，这里的子查询永远查不到行，通知会残留在数据库里
	if err := tx.Where("comment_id IN (SELECT id FROM comments WHERE post_id = ?)", postID).Delete(&models.CommentLike{}).Error; err != nil {
		return err
	}
	if err := tx.Where("type = ? AND resource_id IN (SELECT id FROM comments WHERE post_id = ?)", "reply", postID).Delete(&models.Notification{}).Error; err != nil {
		return err
	}
	if err := tx.Where("post_id = ?", postID).Delete(&models.Comment{}).Error; err != nil {
		return err
	}
	if err := tx.Where("post_id = ?", postID).Delete(&models.PostLike{}).Error; err != nil {
		return err
	}
	if err := tx.Where("post_id = ?", postID).Delete(&models.Favorite{}).Error; err != nil {
		return err
	}
	// like/comment/post 三类通知的 resource_id 直接指向文章
	return tx.Where("type IN ? AND resource_id = ?", []string{"like", "comment", "post"}, postID).Delete(&models.Notification{}).Error
}

// resolveCategory 校验文章关联的分类存在
func resolveCategory(tx *gorm.DB, categoryID *uint) (bool, error) {
	if categoryID == nil {
		return true, nil
	}
	var count int64
	if err := tx.Model(&models.Category{}).Where("id = ?", *categoryID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// FavoritePost 收藏文章，重复收藏幂等返回
func (ic *InteractionController) FavoritePost(c *gin.Context) {
	userID := currentUserID(c)
	var post models.Post
	if err := ic.DB.Where("slug = ? AND status = ? AND moderation_status = ?", c.Param("slug"), "published", "normal").First(&post).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文章不存在"})
		return
	}
	var favorite models.Favorite
	if err := ic.DB.Where("user_id = ? AND post_id = ?", userID, post.ID).First(&favorite).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"favorited": true, "favoriteCount": post.FavoriteCount}})
		return
	}
	if err := ic.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&models.Favorite{UserID: userID, PostID: post.ID}).Error; err != nil {
			return err
		}
		return tx.Model(&models.Post{}).Where("id = ?", post.ID).UpdateColumn("favorite_count", gorm.Expr("favorite_count + 1")).Error
	}); err != nil {
		c.JSON(http.StatusConflict, gin.H{"code": 409, "message": "收藏失败，请重试"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"favorited": true, "favoriteCount": post.FavoriteCount + 1}})
}

// UnfavoritePost 取消收藏
func (ic *InteractionController) UnfavoritePost(c *gin.Context) {
	userID := currentUserID(c)
	var post models.Post
	if err := ic.DB.Where("slug = ?", c.Param("slug")).First(&post).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文章不存在"})
		return
	}
	favoriteCount := post.FavoriteCount
	if err := ic.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("user_id = ? AND post_id = ?", userID, post.ID).Delete(&models.Favorite{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			if err := tx.Model(&models.Post{}).Where("id = ? AND favorite_count > 0", post.ID).UpdateColumn("favorite_count", gorm.Expr("favorite_count - 1")).Error; err != nil {
				return err
			}
		}
		return tx.Model(&models.Post{}).Select("favorite_count").Where("id = ?", post.ID).Scan(&favoriteCount).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "取消收藏失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"favorited": false, "favoriteCount": favoriteCount}})
}

// MyPosts 当前用户的文章列表，包含草稿与归档，可按状态筛选
func (ic *InteractionController) MyPosts(c *gin.Context) {
	userID := currentUserID(c)
	page, pageSize := pagination(c)
	query := ic.DB.Model(&models.Post{}).Where("author_id = ?", userID)
	switch strings.TrimSpace(c.Query("status")) {
	case "published", "scheduled", "draft", "archived":
		query = query.Where("status = ?", strings.TrimSpace(c.Query("status")))
	}
	if keyword := strings.TrimSpace(c.Query("q")); keyword != "" {
		like := likePattern(keyword)
		query = query.Where("title LIKE ? ESCAPE '\\' OR summary LIKE ? ESCAPE '\\'", like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取文章失败"})
		return
	}
	var posts []models.Post
	if err := query.Omit("content").Preload("Tags").Preload("Category").Order("updated_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取文章失败"})
		return
	}
	items := make([]PostSummaryDTO, 0, len(posts))
	liked, favorited := interactionSets(ic.DB, currentUserID(c), postIDs(posts))
	for _, post := range posts {
		item := toPostSummaryDTO(post)
		item.Liked = liked[post.ID]
		item.Favorited = favorited[post.ID]
		items = append(items, item)
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"items": items, "total": total, "page": page, "pageSize": pageSize}})
}

// MyFavorites 当前用户收藏的文章列表。收藏后文章可能被删除或下架，
// 通过 join 文章表过滤，避免收藏页出现点进去 404 的条目
func (ic *InteractionController) MyFavorites(c *gin.Context) {
	userID := currentUserID(c)
	page, pageSize := pagination(c)
	query := ic.DB.Model(&models.Favorite{}).
		Joins("JOIN posts ON posts.id = favorites.post_id").
		Where("favorites.user_id = ? AND posts.status = ? AND posts.moderation_status = ?", userID, "published", "normal")
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取收藏失败"})
		return
	}
	var favorites []models.Favorite
	if err := query.Preload("Post").Preload("Post.Tags").Preload("Post.Author").Preload("Post.Category").Order("favorites.created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&favorites).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取收藏失败"})
		return
	}
	items := make([]PostSummaryDTO, 0, len(favorites))
	collected := make([]models.Post, 0, len(favorites))
	for _, favorite := range favorites {
		collected = append(collected, favorite.Post)
	}
	liked, favorited := interactionSets(ic.DB, userID, postIDs(collected))
	for _, post := range collected {
		item := toPostSummaryDTO(post)
		item.Liked = liked[post.ID]
		item.Favorited = favorited[post.ID]
		items = append(items, item)
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"items": items, "total": total, "page": page, "pageSize": pageSize}})
}

// LikeComment 点赞评论，重复点赞幂等返回
func (ic *InteractionController) LikeComment(c *gin.Context) {
	userID := currentUserID(c)
	var comment models.Comment
	if err := ic.DB.Where("id = ? AND status = ?", c.Param("id"), "published").First(&comment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "评论不存在"})
		return
	}
	var like models.CommentLike
	if err := ic.DB.Where("user_id = ? AND comment_id = ?", userID, comment.ID).First(&like).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"liked": true, "likesCount": comment.LikesCount}})
		return
	}
	if err := ic.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&models.CommentLike{UserID: userID, CommentID: comment.ID}).Error; err != nil {
			return err
		}
		return tx.Model(&models.Comment{}).Where("id = ?", comment.ID).UpdateColumn("likes_count", gorm.Expr("likes_count + 1")).Error
	}); err != nil {
		c.JSON(http.StatusConflict, gin.H{"code": 409, "message": "点赞失败，请重试"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"liked": true, "likesCount": comment.LikesCount + 1}})
}

// UnlikeComment 取消评论点赞
func (ic *InteractionController) UnlikeComment(c *gin.Context) {
	userID := currentUserID(c)
	var comment models.Comment
	if err := ic.DB.First(&comment, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "评论不存在"})
		return
	}
	likesCount := comment.LikesCount
	if err := ic.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("user_id = ? AND comment_id = ?", userID, comment.ID).Delete(&models.CommentLike{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			if err := tx.Model(&models.Comment{}).Where("id = ? AND likes_count > 0", comment.ID).UpdateColumn("likes_count", gorm.Expr("likes_count - 1")).Error; err != nil {
				return err
			}
		}
		return tx.Model(&models.Comment{}).Select("likes_count").Where("id = ?", comment.ID).Scan(&likesCount).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "取消点赞失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"liked": false, "likesCount": likesCount}})
}

// Notifications 当前用户的通知列表，附带未读数
func (ic *InteractionController) Notifications(c *gin.Context) {
	userID := currentUserID(c)
	page, pageSize := pagination(c)
	query := ic.DB.Model(&models.Notification{}).Where("user_id = ?", userID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取通知失败"})
		return
	}
	var unread int64
	if err := ic.DB.Model(&models.Notification{}).Where("user_id = ? AND read_at IS NULL", userID).Count(&unread).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取通知失败"})
		return
	}
	var notifications []models.Notification
	if err := query.Preload("Actor").Order("created_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&notifications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取通知失败"})
		return
	}
	// 通知的目标资源可能是文章或被回复的评论，逐条查询会产生 N+1，
	// 这里按通知 ID 批量预取文章元信息后查表填充
	metaByNotification, err := notificationPostMeta(ic.DB, notifications)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取通知目标失败"})
		return
	}
	items := make([]NotificationDTO, 0, len(notifications))
	for _, notification := range notifications {
		item := NotificationDTO{ID: notification.ID, Type: notification.Type, ResourceID: notification.ResourceID, Read: notification.ReadAt != nil, CreatedAt: notification.CreatedAt, Actor: toUserDTO(notification.Actor)}
		// 从预取的文章元信息中取标题与 slug，便于前端生成跳转链接
		if meta, ok := metaByNotification[notification.ID]; ok {
			item.ResourceSlug = meta.Slug
			item.ResourceTitle = meta.Title
		}
		items = append(items, item)
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"items": items, "total": total, "unread": unread, "page": page, "pageSize": pageSize}})
}

// postMeta 通知目标文章的标题与 slug
type postMeta struct {
	Title string
	Slug  string
}

// notificationPostMeta 按通知 ID 批量预取关联文章的标题与 slug：
// comment/like/post 类通知的 resource_id 直接指向文章；reply 类指向评论，
// 需先按评论找到所属文章。整个列表最多三次查询（文章、评论、回复关联文章），
// 与通知条数无关；文章已被删除时对应通知不填充，前端降级为不可跳转
func notificationPostMeta(db *gorm.DB, notifications []models.Notification) (map[uint]postMeta, error) {
	postIDs := make([]uint, 0, len(notifications))
	replyCommentIDs := make([]uint, 0, len(notifications))
	for _, notification := range notifications {
		switch notification.Type {
		case "comment", "like", "post":
			postIDs = append(postIDs, notification.ResourceID)
		case "reply":
			replyCommentIDs = append(replyCommentIDs, notification.ResourceID)
		}
	}
	// 回复类通知先按评论批量取 post_id，并入文章查询集合
	commentPostID := map[uint]uint{}
	if len(replyCommentIDs) > 0 {
		var comments []models.Comment
		if err := db.Select("id", "post_id").Where("id IN ?", replyCommentIDs).Find(&comments).Error; err != nil {
			return nil, err
		}
		for _, comment := range comments {
			commentPostID[comment.ID] = comment.PostID
			postIDs = append(postIDs, comment.PostID)
		}
	}
	posts := map[uint]postMeta{}
	if len(postIDs) > 0 {
		var rows []models.Post
		if err := db.Select("id", "title", "slug").Where("id IN ?", postIDs).Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, row := range rows {
			posts[row.ID] = postMeta{Title: row.Title, Slug: row.Slug}
		}
	}
	meta := map[uint]postMeta{}
	for _, notification := range notifications {
		// follow 类通知的 resource_id 是关注者用户 ID，与文章 ID 不同空间，不能顺便填充
		if notification.Type == "follow" {
			continue
		}
		postID := notification.ResourceID
		if notification.Type == "reply" {
			postID = commentPostID[notification.ResourceID]
		}
		if post, ok := posts[postID]; ok {
			meta[notification.ID] = post
		}
	}
	return meta, nil
}

// MarkNotificationRead 标记通知为已读
func (ic *InteractionController) MarkNotificationRead(c *gin.Context) {
	userID := currentUserID(c)
	result := ic.DB.Model(&models.Notification{}).Where("id = ? AND user_id = ?", c.Param("id"), userID).Update("read_at", time.Now())
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新通知失败"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "通知不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// CreateReport 举报文章或评论
func (ic *InteractionController) CreateReport(c *gin.Context) {
	userID := currentUserID(c)
	var input struct {
		TargetType string `json:"targetType" binding:"required"`
		TargetID   uint   `json:"targetId" binding:"required"`
		Reason     string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Reason) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "举报理由不能为空"})
		return
	}
	input.TargetType = strings.TrimSpace(input.TargetType)
	input.Reason = strings.TrimSpace(input.Reason)
	if input.TargetType != "post" && input.TargetType != "comment" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "举报目标类型无效"})
		return
	}
	if len(input.Reason) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "举报理由不能超过 500 字"})
		return
	}
	// 校验举报目标存在
	switch input.TargetType {
	case "post":
		var count int64
		if err := ic.DB.Model(&models.Post{}).Where("id = ?", input.TargetID).Count(&count).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "检查举报目标失败"})
			return
		}
		if count == 0 {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "举报的文章不存在"})
			return
		}
	case "comment":
		var count int64
		if err := ic.DB.Model(&models.Comment{}).Where("id = ?", input.TargetID).Count(&count).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "检查举报目标失败"})
			return
		}
		if count == 0 {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "举报的评论不存在"})
			return
		}
	}
	report := models.Report{ReporterID: userID, TargetType: input.TargetType, TargetID: input.TargetID, Reason: input.Reason, Status: "pending"}
	if err := ic.DB.Create(&report).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "提交举报失败"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "success"})
}

// AdminReports 管理端举报列表，可按状态筛选，附带举报目标摘要
func (ic *InteractionController) AdminReports(c *gin.Context) {
	page, pageSize := pagination(c)
	query := ic.DB.Model(&models.Report{})
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取举报失败"})
		return
	}
	var reports []models.Report
	if err := query.Preload("Reporter").Order("created_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&reports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取举报失败"})
		return
	}
	items := make([]AdminReportDTO, 0, len(reports))
	for _, report := range reports {
		items = append(items, AdminReportDTO{ID: report.ID, TargetType: report.TargetType, TargetID: report.TargetID, TargetSummary: ic.reportTargetSummary(report), Reason: report.Reason, Status: report.Status, CreatedAt: report.CreatedAt, Reporter: toUserDTO(report.Reporter)})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"items": items, "total": total, "page": page, "pageSize": pageSize}})
}

// reportTargetSummary 生成举报目标的文字摘要，便于管理员判断
func (ic *InteractionController) reportTargetSummary(report models.Report) string {
	switch report.TargetType {
	case "post":
		var post models.Post
		if err := ic.DB.Select("title", "slug").First(&post, report.TargetID).Error; err != nil {
			return "文章已删除"
		}
		return "文章：" + post.Title
	case "comment":
		var comment models.Comment
		if err := ic.DB.Select("content").First(&comment, report.TargetID).Error; err != nil {
			return "评论已删除"
		}
		if len(comment.Content) > 80 {
			return "评论：" + comment.Content[:80] + "…"
		}
		return "评论：" + comment.Content
	}
	return ""
}

// AdminHandleReport 管理员处理举报：handled 标记已处理，dismissed 驳回
func (ic *InteractionController) AdminHandleReport(c *gin.Context) {
	adminID := currentUserID(c)
	var input struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || (input.Status != "handled" && input.Status != "dismissed") {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "举报处理状态无效"})
		return
	}
	var report models.Report
	if err := ic.DB.First(&report, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "举报不存在"})
		return
	}
	if err := ic.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&report).Updates(map[string]any{"status": input.Status, "handled_by": adminID}).Error; err != nil {
			return err
		}
		return writeAdminLog(tx, adminID, "report."+input.Status, report.TargetType, report.TargetID, report.Reason)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "处理举报失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// AdminLogs 管理操作日志列表
func (ic *InteractionController) AdminLogs(c *gin.Context) {
	page, pageSize := pagination(c)
	query := ic.DB.Model(&models.AdminLog{})
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取日志失败"})
		return
	}
	var logs []models.AdminLog
	if err := query.Preload("Admin").Order("created_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取日志失败"})
		return
	}
	items := make([]AdminLogDTO, 0, len(logs))
	for _, entry := range logs {
		items = append(items, AdminLogDTO{ID: entry.ID, Action: entry.Action, TargetType: entry.TargetType, TargetID: entry.TargetID, Detail: entry.Detail, CreatedAt: entry.CreatedAt, Admin: toUserDTO(entry.Admin)})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"items": items, "total": total, "page": page, "pageSize": pageSize}})
}

// AdminSettings 读取系统设置
func (ic *InteractionController) AdminSettings(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{
		"openRegistration": models.BoolSetting(ic.DB, "open_registration", true),
		"commentsEnabled":  models.BoolSetting(ic.DB, "comments_enabled", true),
	}})
}

// AdminUpdateSettings 更新系统设置并记录管理日志
func (ic *InteractionController) AdminUpdateSettings(c *gin.Context) {
	adminID := currentUserID(c)
	var input struct {
		OpenRegistration *bool `json:"openRegistration"`
		CommentsEnabled  *bool `json:"commentsEnabled"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || (input.OpenRegistration == nil && input.CommentsEnabled == nil) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "设置内容无效"})
		return
	}
	if err := ic.DB.Transaction(func(tx *gorm.DB) error {
		if input.OpenRegistration != nil {
			if err := models.SaveSetting(tx, "open_registration", strconv.FormatBool(*input.OpenRegistration)); err != nil {
				return err
			}
		}
		if input.CommentsEnabled != nil {
			if err := models.SaveSetting(tx, "comments_enabled", strconv.FormatBool(*input.CommentsEnabled)); err != nil {
				return err
			}
		}
		return writeAdminLog(tx, adminID, "settings.update", "settings", 0, "更新系统设置")
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "保存设置失败"})
		return
	}
	ic.AdminSettings(c)
}

// ListCategories 公开分类列表
func (ic *InteractionController) ListCategories(c *gin.Context) {
	var categories []models.Category
	if err := ic.DB.Order("name ASC").Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取分类失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": categories})
}

// AdminTags 管理端标签与分类列表，带使用计数
func (ic *InteractionController) AdminTags(c *gin.Context) {
	var tags []models.Tag
	if err := ic.DB.Order("name ASC").Find(&tags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取标签失败"})
		return
	}
	var categories []models.Category
	if err := ic.DB.Order("name ASC").Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取分类失败"})
		return
	}
	type TagUsage struct {
		ID    uint   `json:"id"`
		Name  string `json:"name"`
		Slug  string `json:"slug"`
		Usage int64  `json:"usage"`
	}
	// 逐个标签/分类 COUNT 是 N+1，改为按关联字段分组统计一次查完
	tagUsage := map[uint]int64{}
	var tagRows []struct {
		TagID uint
		Count int64
	}
	if err := ic.DB.Model(&models.Post{}).Select("pt.tag_id AS tag_id, COUNT(*) AS count").
		Joins("JOIN post_tags pt ON pt.post_id = posts.id").Group("pt.tag_id").Scan(&tagRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取标签统计失败"})
		return
	}
	for _, row := range tagRows {
		tagUsage[row.TagID] = row.Count
	}
	categoryUsage := map[uint]int64{}
	var categoryRows []struct {
		CategoryID uint
		Count      int64
	}
	if err := ic.DB.Model(&models.Post{}).Select("category_id, COUNT(*) AS count").
		Where("category_id IS NOT NULL").Group("category_id").Scan(&categoryRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取分类统计失败"})
		return
	}
	for _, row := range categoryRows {
		categoryUsage[row.CategoryID] = row.Count
	}
	tagItems := make([]TagUsage, 0, len(tags))
	for _, tag := range tags {
		tagItems = append(tagItems, TagUsage{ID: tag.ID, Name: tag.Name, Slug: tag.Slug, Usage: tagUsage[tag.ID]})
	}
	type CategoryUsage struct {
		ID    uint   `json:"id"`
		Name  string `json:"name"`
		Slug  string `json:"slug"`
		Usage int64  `json:"usage"`
	}
	categoryItems := make([]CategoryUsage, 0, len(categories))
	for _, category := range categories {
		categoryItems = append(categoryItems, CategoryUsage{ID: category.ID, Name: category.Name, Slug: category.Slug, Usage: categoryUsage[category.ID]})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"tags": tagItems, "categories": categoryItems}})
}

// CreateCategory 登录用户创建分类，作者可以在编辑文章时即时创建并选择分类
func (ic *InteractionController) CreateCategory(c *gin.Context) {
	var input struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "分类名称不能为空"})
		return
	}
	name := strings.TrimSpace(input.Name)
	if len(name) > 80 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "分类名称不能超过 80 个字符"})
		return
	}
	var user models.User
	if err := ic.DB.Select("status").First(&user, currentUserID(c)).Error; err != nil || user.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "当前账号不能创建分类"})
		return
	}
	var count int64
	if err := ic.DB.Model(&models.Category{}).Where("name = ?", name).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "检查分类失败"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"code": 409, "message": "分类已存在"})
		return
	}
	slug, err := uniqueCategorySlug(ic.DB, slugify(name))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "生成分类标识失败"})
		return
	}
	category := models.Category{Name: name, Slug: slug}
	if err := ic.DB.Create(&category).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建分类失败"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "success", "data": category})
}

// DeleteCategory 管理员删除分类，文章的分类引用置空
func (ic *InteractionController) DeleteCategory(c *gin.Context) {
	adminID := currentUserID(c)
	var category models.Category
	if err := ic.DB.First(&category, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "分类不存在"})
		return
	}
	if err := ic.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Post{}).Where("category_id = ?", category.ID).Update("category_id", nil).Error; err != nil {
			return err
		}
		if err := tx.Delete(&category).Error; err != nil {
			return err
		}
		return writeAdminLog(tx, adminID, "category.delete", "category", category.ID, category.Name)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除分类失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// DeleteTag 管理员删除标签，同时清空文章关联
func (ic *InteractionController) DeleteTag(c *gin.Context) {
	adminID := currentUserID(c)
	var tag models.Tag
	if err := ic.DB.First(&tag, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "标签不存在"})
		return
	}
	if err := ic.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM post_tags WHERE tag_id = ?", tag.ID).Error; err != nil {
			return err
		}
		if err := tx.Delete(&tag).Error; err != nil {
			return err
		}
		return writeAdminLog(tx, adminID, "tag.delete", "tag", tag.ID, tag.Name)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除标签失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

func uniqueCategorySlug(db *gorm.DB, base string) (string, error) {
	if base == "" {
		base = "category"
	}
	candidate := base
	for i := 2; ; i++ {
		var count int64
		if err := db.Model(&models.Category{}).Where("slug = ?", candidate).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
		candidate = strconv.Itoa(i) + "-" + base
	}
}

// OwnerPostDetail 作者或管理员获取单篇文章（含草稿），用于编辑
func (ic *InteractionController) OwnerPostDetail(c *gin.Context) {
	userID := currentUserID(c)
	var post models.Post
	if err := ic.DB.Preload("Tags").Preload("Author").Preload("Category").First(&post, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文章不存在"})
		return
	}
	if post.AuthorID != userID && !currentUserIsAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只能查看自己的文章"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": toPostDTO(post)})
}
