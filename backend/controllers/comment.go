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

// 评论域：公开评论的读取与发布、评论删除，以及管理端的评论审核。
// 与帖子域（post.go / community.go）、关注域（follow.go）、
// 管理端用户与统计（admin.go）按职责分文件存放

type CommentDTO struct {
	ID            uint      `json:"id"`
	PostID        uint      `json:"postId"`
	ParentID      *uint     `json:"parentId"`
	ReplyToUserID *uint     `json:"replyToUserId"`
	Content       string    `json:"content"`
	LikesCount    int       `json:"likesCount"`
	Liked         bool      `json:"liked"`
	CreatedAt     time.Time `json:"createdAt"`
	Author        UserDTO   `json:"author"`
}

func toCommentDTO(comment models.Comment, liked bool) CommentDTO {
	return CommentDTO{ID: comment.ID, PostID: comment.PostID, ParentID: comment.ParentID, ReplyToUserID: comment.ReplyToUserID, Content: comment.Content, LikesCount: comment.LikesCount, Liked: liked, CreatedAt: comment.CreatedAt, Author: toUserDTO(comment.Author)}
}

// ListComments 文章的公开评论列表，按时间正序返回
func (cc *CommunityController) ListComments(c *gin.Context) {
	var post models.Post
	if err := cc.DB.Where("slug = ? AND status = ? AND moderation_status = ?", c.Param("slug"), "published", "normal").First(&post).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文章不存在"})
		return
	}
	var comments []models.Comment
	if err := cc.DB.Where("post_id = ? AND status = ?", post.ID, "published").Preload("Author").Order("created_at ASC, id ASC").Find(&comments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取评论失败"})
		return
	}
	// 批量查询当前用户对这批评论的点赞，用于标记 liked
	likedSet := map[uint]bool{}
	if userID := currentUserID(c); userID > 0 {
		commentIDs := make([]uint, 0, len(comments))
		for _, comment := range comments {
			commentIDs = append(commentIDs, comment.ID)
		}
		if len(commentIDs) > 0 {
			var likedIDs []uint
			cc.DB.Model(&models.CommentLike{}).Where("user_id = ? AND comment_id IN ?", userID, commentIDs).Pluck("comment_id", &likedIDs)
			for _, id := range likedIDs {
				likedSet[id] = true
			}
		}
	}
	items := make([]CommentDTO, 0, len(comments))
	for _, comment := range comments {
		items = append(items, toCommentDTO(comment, likedSet[comment.ID]))
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": items})
}

func (cc *CommunityController) CreateComment(c *gin.Context) {
	userID := currentUserID(c)
	if !cc.ensureActiveUser(c, userID) {
		return
	}
	// 管理员可在后台关闭全站评论
	if !models.BoolSetting(cc.DB, "comments_enabled", true) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "评论功能已关闭"})
		return
	}
	var input struct {
		Content       string `json:"content" binding:"required"`
		ParentID      *uint  `json:"parentId"`
		ReplyToUserID *uint  `json:"replyToUserId"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Content) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "评论内容不能为空"})
		return
	}
	var post models.Post
	if err := cc.DB.Where("slug = ? AND status = ? AND moderation_status = ?", c.Param("slug"), "published", "normal").First(&post).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文章不存在"})
		return
	}
	if input.ParentID != nil {
		var parent models.Comment
		if err := cc.DB.Where("id = ? AND post_id = ? AND status = ?", *input.ParentID, post.ID, "published").First(&parent).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "回复目标不存在"})
			return
		}
		// 方案要求评论最多两层：对回复的回复，统一挂到其所属的顶层评论下
		if parent.ParentID != nil {
			rootID := *parent.ParentID
			input.ParentID = &rootID
		}
	}
	comment := models.Comment{PostID: post.ID, AuthorID: userID, ParentID: input.ParentID, ReplyToUserID: input.ReplyToUserID, Content: strings.TrimSpace(input.Content), Status: "published"}
	if err := cc.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&comment).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Post{}).Where("id = ?", post.ID).UpdateColumn("comments_count", gorm.Expr("comments_count + 1")).Error; err != nil {
			return err
		}
		// 通知文章作者与被回复者，自己操作不通知；被回复者是作者时只发一条 reply
		if post.AuthorID != userID {
			if err := createNotification(tx, post.AuthorID, userID, "comment", post.ID); err != nil {
				return err
			}
		}
		if input.ReplyToUserID != nil && *input.ReplyToUserID != userID && *input.ReplyToUserID != post.AuthorID {
			if err := createNotification(tx, *input.ReplyToUserID, userID, "reply", comment.ID); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "发表评论失败"})
		return
	}
	cc.DB.Preload("Author").First(&comment, comment.ID)
	c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "success", "data": toCommentDTO(comment, false)})
}

func (cc *CommunityController) DeleteComment(c *gin.Context) {
	var comment models.Comment
	if err := cc.DB.First(&comment, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "评论不存在"})
		return
	}
	if comment.AuthorID != currentUserID(c) && !currentUserIsAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只能删除自己的评论"})
		return
	}
	if err := cc.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("comment_id = ?", comment.ID).Delete(&models.CommentLike{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&comment).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Post{}).Where("id = ? AND comments_count > 0", comment.PostID).UpdateColumn("comments_count", gorm.Expr("comments_count - 1")).Error; err != nil {
			return err
		}
		// 管理员删除评论保留操作日志
		if currentUserIsAdmin(c) {
			return writeAdminLog(tx, currentUserID(c), "comment.delete", "comment", comment.ID, comment.Content)
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除评论失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

type AdminCommentDTO struct {
	ID        uint      `json:"id"`
	PostID    uint      `json:"postId"`
	PostTitle string    `json:"postTitle"`
	PostSlug  string    `json:"postSlug"`
	Content   string    `json:"content"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	Author    UserDTO   `json:"author"`
}

// AdminComments 管理端评论列表，可按状态或所属文章筛选，展示所属文章与作者
func (cc *CommunityController) AdminComments(c *gin.Context) {
	page, pageSize := pagination(c)
	query := cc.DB.Model(&models.Comment{})
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	// 按文章筛选用于查看评论上下文
	if postID, err := strconv.ParseUint(c.Query("postId"), 10, 64); err == nil && postID > 0 {
		query = query.Where("post_id = ?", postID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取评论失败"})
		return
	}
	var comments []models.Comment
	if err := query.Preload("Author").Preload("Post").Order("created_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&comments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取评论失败"})
		return
	}
	items := make([]AdminCommentDTO, 0, len(comments))
	for _, comment := range comments {
		items = append(items, AdminCommentDTO{ID: comment.ID, PostID: comment.PostID, PostTitle: comment.Post.Title, PostSlug: comment.Post.Slug, Content: comment.Content, Status: comment.Status, CreatedAt: comment.CreatedAt, Author: toUserDTO(comment.Author)})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"items": items, "total": total, "page": page, "pageSize": pageSize}})
}

// AdminUpdateCommentStatus 管理员隐藏/恢复评论，并同步文章的评论计数
func (cc *CommunityController) AdminUpdateCommentStatus(c *gin.Context) {
	var input struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || (input.Status != "published" && input.Status != "hidden") {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "评论状态无效"})
		return
	}
	var comment models.Comment
	if err := cc.DB.First(&comment, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "评论不存在"})
		return
	}
	if comment.Status == input.Status {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
		return
	}
	if err := cc.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&comment).Update("status", input.Status).Error; err != nil {
			return err
		}
		// 隐藏减少计数、恢复增加计数，保持评论数与公开评论一致
		if input.Status == "hidden" {
			return tx.Model(&models.Post{}).Where("id = ? AND comments_count > 0", comment.PostID).UpdateColumn("comments_count", gorm.Expr("comments_count - 1")).Error
		}
		return tx.Model(&models.Post{}).Where("id = ?", comment.PostID).UpdateColumn("comments_count", gorm.Expr("comments_count + 1")).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新评论状态失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}
