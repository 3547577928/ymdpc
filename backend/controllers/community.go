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

type UserDTO struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Nickname  string    `json:"nickname"`
	Avatar    string    `json:"avatar"`
	Bio       string    `json:"bio"`
	Role      string    `json:"role,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

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

type CommunityController struct{ DB *gorm.DB }

func currentUserID(c *gin.Context) uint {
	value, ok := c.Get("userID")
	if !ok {
		return 0
	}
	switch id := value.(type) {
	case uint:
		return id
	case int:
		return uint(id)
	case float64:
		return uint(id)
	default:
		return 0
	}
}

func currentUserIsAdmin(c *gin.Context) bool {
	role, ok := c.Get("userRole")
	return ok && role == "admin"
}

// ensureActiveUser 校验当前登录用户存在且未被禁言/封禁，方案要求禁言用户不能发布文章或评论
func (cc *CommunityController) ensureActiveUser(c *gin.Context, userID uint) bool {
	var user models.User
	if err := cc.DB.Select("status").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "unauthorized"})
		return false
	}
	if user.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "账号已被禁言或封禁，暂时无法发布内容"})
		return false
	}
	return true
}

func (cc *CommunityController) Feed(c *gin.Context) {
	page, pageSize := pagination(c)
	query := cc.DB.Model(&models.Post{}).Where("posts.status = ? AND posts.moderation_status = ?", "published", "normal")
	if keyword := strings.TrimSpace(c.Query("q")); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("posts.title LIKE ? OR posts.summary LIKE ? OR posts.content LIKE ?", like, like, like)
	}
	if tag := strings.TrimSpace(c.Query("tag")); tag != "" {
		query = query.Where("EXISTS (SELECT 1 FROM post_tags pt JOIN tags t ON t.id = pt.tag_id WHERE pt.post_id = posts.id AND (t.name = ? OR t.slug = ?))", tag, tag)
	}
	mode := strings.TrimSpace(c.DefaultQuery("mode", "latest"))
	if mode == "following" {
		userID := currentUserID(c)
		if userID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "登录后查看关注动态"})
			return
		}
		query = query.Where("posts.author_id IN (SELECT following_id FROM follows WHERE follower_id = ?)", userID)
	}
	order := "posts.published_at DESC, posts.id DESC"
	if mode == "hot" {
		// 热度按阅读、点赞、评论综合计算，并按发布时长做时间衰减
		order = "((posts.likes_count * 4 + posts.comments_count * 6 + posts.views / 100.0) / max((julianday('now') - julianday(posts.published_at)) * 24.0, 2.0)) DESC, posts.published_at DESC, posts.id DESC"
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取信息流失败"})
		return
	}
	var posts []models.Post
	if err := query.Omit("content").Preload("Tags").Preload("Author").Preload("Category").Order(order).Offset((page - 1) * pageSize).Limit(pageSize).Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取信息流失败"})
		return
	}
	items := make([]PostSummaryDTO, 0, len(posts))
	for _, post := range posts {
		item := toPostSummaryDTO(post)
		item.Liked = cc.isPostLiked(c, post.ID)
		items = append(items, item)
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"items": items, "total": total, "page": page, "pageSize": pageSize, "mode": mode}})
}

func (cc *CommunityController) CreatePost(c *gin.Context) {
	userID := currentUserID(c)
	if !cc.ensureActiveUser(c, userID) {
		return
	}
	var input PostRequest
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.Content) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "标题和正文不能为空"})
		return
	}
	status := strings.TrimSpace(input.Status)
	if status != "published" {
		status = "draft"
	}
	if !resolveCategory(cc.DB, input.CategoryID) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "分类不存在"})
		return
	}
	slug := input.Slug
	if slug == "" {
		slug = input.Title
	}
	post := models.Post{AuthorID: userID, Title: strings.TrimSpace(input.Title), Slug: uniqueSlug(cc.DB, slugify(slug), 0), Summary: strings.TrimSpace(input.Summary), Content: input.Content, CoverImage: strings.TrimSpace(input.CoverImage), Status: status, ReadingTime: readingTime(input.Content), CategoryID: input.CategoryID}
	if status == "published" {
		post.PublishedAt = time.Now()
	}
	if err := cc.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&post).Error; err != nil {
			return err
		}
		if err := replaceTags(tx, &post, input.Tags); err != nil {
			return err
		}
		if status == "published" {
			return notifyFollowersOfPost(tx, post.AuthorID, post.ID)
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建文章失败"})
		return
	}
	cc.DB.Preload("Tags").Preload("Author").Preload("Category").First(&post, post.ID)
	c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "success", "data": toPostDTO(post)})
}

func (cc *CommunityController) UpdatePost(c *gin.Context) {
	userID := currentUserID(c)
	var post models.Post
	if err := cc.DB.Preload("Tags").Preload("Author").Preload("Category").First(&post, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文章不存在"})
		return
	}
	if post.AuthorID != userID && !currentUserIsAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只能编辑自己的文章"})
		return
	}
	var input PostRequest
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.Content) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "标题和正文不能为空"})
		return
	}
	wasPublished := post.Status == "published"
	status := strings.TrimSpace(input.Status)
	if status != "published" && status != "archived" {
		status = "draft"
	}
	if !resolveCategory(cc.DB, input.CategoryID) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "分类不存在"})
		return
	}
	slug := input.Slug
	if slug == "" {
		slug = post.Slug
	}
	post.Title = strings.TrimSpace(input.Title)
	post.Slug = uniqueSlug(cc.DB, slugify(slug), post.ID)
	post.Summary = strings.TrimSpace(input.Summary)
	post.Content = input.Content
	post.CoverImage = strings.TrimSpace(input.CoverImage)
	post.Status = status
	post.ReadingTime = readingTime(input.Content)
	post.CategoryID = input.CategoryID
	if status == "published" && post.PublishedAt.IsZero() {
		post.PublishedAt = time.Now()
	}
	if err := cc.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&post).Error; err != nil {
			return err
		}
		if err := replaceTags(tx, &post, input.Tags); err != nil {
			return err
		}
		if !wasPublished && status == "published" {
			return notifyFollowersOfPost(tx, post.AuthorID, post.ID)
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新文章失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": toPostDTO(post)})
}

func (cc *CommunityController) DeletePost(c *gin.Context) {
	userID := currentUserID(c)
	var post models.Post
	if err := cc.DB.First(&post, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文章不存在"})
		return
	}
	if post.AuthorID != userID && !currentUserIsAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只能删除自己的文章"})
		return
	}
	if err := cc.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&post).Association("Tags").Clear(); err != nil {
			return err
		}
		if err := deletePostRelations(tx, post.ID); err != nil {
			return err
		}
		return tx.Delete(&post).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除文章失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

func (cc *CommunityController) LikePost(c *gin.Context) {
	userID := currentUserID(c)
	var post models.Post
	if err := cc.DB.Where("slug = ? AND status = ? AND moderation_status = ?", c.Param("slug"), "published", "normal").First(&post).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文章不存在"})
		return
	}
	var like models.PostLike
	if err := cc.DB.Where("user_id = ? AND post_id = ?", userID, post.ID).First(&like).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"liked": true, "likesCount": post.LikesCount}})
		return
	}
	if err := cc.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&models.PostLike{UserID: userID, PostID: post.ID}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Post{}).Where("id = ?", post.ID).UpdateColumn("likes_count", gorm.Expr("likes_count + 1")).Error; err != nil {
			return err
		}
		// 通知文章作者收到点赞，自己给自己点赞不通知
		if post.AuthorID != userID {
			return createNotification(tx, post.AuthorID, userID, "like", post.ID)
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusConflict, gin.H{"code": 409, "message": "点赞失败，请重试"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"liked": true, "likesCount": post.LikesCount + 1}})
}

func (cc *CommunityController) UnlikePost(c *gin.Context) {
	userID := currentUserID(c)
	var post models.Post
	if err := cc.DB.Where("slug = ? AND status = ? AND moderation_status = ?", c.Param("slug"), "published", "normal").First(&post).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文章不存在"})
		return
	}
	result := cc.DB.Where("user_id = ? AND post_id = ?", userID, post.ID).Delete(&models.PostLike{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "取消点赞失败"})
		return
	}
	if result.RowsAffected > 0 {
		cc.DB.Model(&models.Post{}).Where("id = ? AND likes_count > 0", post.ID).UpdateColumn("likes_count", gorm.Expr("likes_count - 1"))
	}
	cc.DB.Select("likes_count").First(&post, post.ID)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"liked": false, "likesCount": post.LikesCount}})
}

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

func (cc *CommunityController) FollowUser(c *gin.Context) {
	userID := currentUserID(c)
	targetID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || uint(targetID) == userID {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不能关注自己"})
		return
	}
	var target models.User
	if err := cc.DB.Where("id = ? AND status = ?", targetID, "active").First(&target).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}
	var follow models.Follow
	if err := cc.DB.Where("follower_id = ? AND following_id = ?", userID, targetID).First(&follow).Error; err == nil {
		cc.followResponse(c, userID, uint(targetID), true)
		return
	}
	if err := cc.DB.Create(&models.Follow{FollowerID: userID, FollowingID: uint(targetID)}).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"code": 409, "message": "关注失败，请重试"})
		return
	}
	// 通知被关注者，resourceId 指向关注者，便于跳转主页
	if err := createNotification(cc.DB, uint(targetID), userID, "follow", userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "关注失败，请重试"})
		return
	}
	cc.followResponse(c, userID, uint(targetID), true)
}

func (cc *CommunityController) UnfollowUser(c *gin.Context) {
	targetID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "用户不存在"})
		return
	}
	cc.DB.Where("follower_id = ? AND following_id = ?", currentUserID(c), targetID).Delete(&models.Follow{})
	cc.followResponse(c, currentUserID(c), uint(targetID), false)
}

func (cc *CommunityController) Profile(c *gin.Context) {
	var user models.User
	if err := cc.DB.Where("username = ? AND status = ?", c.Param("username"), "active").First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}
	profile := cc.profileData(user)
	var following int64
	if currentUserID(c) > 0 {
		cc.DB.Model(&models.Follow{}).Where("follower_id = ? AND following_id = ?", currentUserID(c), user.ID).Count(&following)
	}
	profile.FollowingMe = following > 0
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": profile})
}

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

type UserProfileDTO struct {
	User        UserDTO          `json:"user"`
	Posts       []PostSummaryDTO `json:"posts"`
	PostCount   int64            `json:"postCount"`
	LikeCount   int64            `json:"likeCount"`
	Followers   int64            `json:"followers"`
	Following   int64            `json:"following"`
	FollowingMe bool             `json:"followingMe"`
}

type AdminUserDTO struct {
	UserDTO
	Status    string `json:"status"`
	PostCount int64  `json:"postCount"`
	LikeCount int64  `json:"likeCount"`
	Followers int64  `json:"followers"`
	Following int64  `json:"following"`
}

func (cc *CommunityController) profileData(user models.User) UserProfileDTO {
	var posts []models.Post
	cc.DB.Where("author_id = ? AND status = ? AND moderation_status = ?", user.ID, "published", "normal").Preload("Tags").Preload("Author").Preload("Category").Order("published_at DESC, id DESC").Limit(30).Find(&posts)
	var postCount, likeCount, followers, following int64
	cc.DB.Model(&models.Post{}).Where("author_id = ? AND status = ? AND moderation_status = ?", user.ID, "published", "normal").Count(&postCount)
	cc.DB.Model(&models.PostLike{}).Joins("JOIN posts ON posts.id = post_likes.post_id").Where("posts.author_id = ?", user.ID).Count(&likeCount)
	cc.DB.Model(&models.Follow{}).Where("following_id = ?", user.ID).Count(&followers)
	cc.DB.Model(&models.Follow{}).Where("follower_id = ?", user.ID).Count(&following)
	items := make([]PostSummaryDTO, 0, len(posts))
	for _, post := range posts {
		items = append(items, toPostSummaryDTO(post))
	}
	return UserProfileDTO{User: toUserDTO(user), Posts: items, PostCount: postCount, LikeCount: likeCount, Followers: followers, Following: following}
}

func (cc *CommunityController) adminUserData(user models.User) AdminUserDTO {
	profile := cc.profileData(user)
	var postCount, likeCount int64
	cc.DB.Model(&models.Post{}).Where("author_id = ?", user.ID).Count(&postCount)
	cc.DB.Model(&models.PostLike{}).Joins("JOIN posts ON posts.id = post_likes.post_id").Where("posts.author_id = ?", user.ID).Count(&likeCount)
	return AdminUserDTO{UserDTO: profile.User, Status: user.Status, PostCount: postCount, LikeCount: likeCount, Followers: profile.Followers, Following: profile.Following}
}

func (cc *CommunityController) followResponse(c *gin.Context, followerID, followingID uint, following bool) error {
	var followers int64
	cc.DB.Model(&models.Follow{}).Where("following_id = ?", followingID).Count(&followers)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"following": following, "followers": followers, "followerId": followerID}})
	return nil
}

func (cc *CommunityController) isPostLiked(c *gin.Context, postID uint) bool {
	userID := currentUserID(c)
	if userID == 0 {
		return false
	}
	var count int64
	cc.DB.Model(&models.PostLike{}).Where("user_id = ? AND post_id = ?", userID, postID).Count(&count)
	return count > 0
}

func toCommentDTO(comment models.Comment, liked bool) CommentDTO {
	return CommentDTO{ID: comment.ID, PostID: comment.PostID, ParentID: comment.ParentID, ReplyToUserID: comment.ReplyToUserID, Content: comment.Content, LikesCount: comment.LikesCount, Liked: liked, CreatedAt: comment.CreatedAt, Author: toUserDTO(comment.Author)}
}

func toUserDTO(user models.User) UserDTO {
	return UserDTO{ID: user.ID, Username: user.Username, Nickname: user.Nickname, Avatar: user.Avatar, Bio: user.Bio, Role: user.Role, CreatedAt: user.CreatedAt}
}

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

// ActiveUserDTO 概览中的活跃用户条目
type ActiveUserDTO struct {
	User         UserDTO `json:"user"`
	PostCount    int64   `json:"postCount"`
	CommentCount int64   `json:"commentCount"`
}
