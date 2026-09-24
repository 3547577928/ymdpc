package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"quietsignal/backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 社区域：信息流（最新/热门/关注）、用户发文与编辑、文章点赞。
// 评论见 comment.go，关注与用户主页见 follow.go，
// 管理端用户与统计见 admin.go，管理端文章操作见 post.go，
// 收藏/通知/举报等互动见 interaction.go

type UserDTO struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Nickname  string    `json:"nickname"`
	Avatar    string    `json:"avatar"`
	Bio       string    `json:"bio"`
	Role      string    `json:"role,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type CommunityController struct {
	DB        *gorm.DB
	UploadDir string
	// StatsCache UserStats 的 60 秒进程内缓存，见 admin.go
	StatsCache SiteStatsCache
}

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

// interestProfile 汇总用户的关注作者、点赞/收藏文章的标签与分类 ID 集合，
// 供推荐排序拼成常量 IN 列表；每个集合带上限，防御极端活跃账号
func (cc *CommunityController) interestProfile(userID uint) (authorIDs, tagIDs, categoryIDs []uint) {
	cc.DB.Model(&models.Follow{}).Where("follower_id = ?", userID).Limit(500).Pluck("following_id", &authorIDs)
	cc.DB.Raw(`SELECT DISTINCT pt.tag_id FROM post_tags pt WHERE pt.post_id IN (
		SELECT post_id FROM post_likes WHERE user_id = ?
		UNION SELECT post_id FROM favorites WHERE user_id = ?
	) LIMIT 500`, userID, userID).Scan(&tagIDs)
	cc.DB.Raw(`SELECT DISTINCT category_id FROM posts WHERE category_id IS NOT NULL AND id IN (
		SELECT post_id FROM post_likes WHERE user_id = ?
		UNION SELECT post_id FROM favorites WHERE user_id = ?
	) LIMIT 500`, userID, userID).Scan(&categoryIDs)
	return authorIDs, tagIDs, categoryIDs
}

// uintInList 把 ID 集合格式化为 SQL IN 列表；ID 来自数据库无注入风险，空集返回 -1 不匹配任何行
func uintInList(ids []uint) string {
	if len(ids) == 0 {
		return "-1"
	}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatUint(uint64(id), 10)
	}
	return strings.Join(parts, ",")
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

func toUserDTO(user models.User) UserDTO {
	return UserDTO{ID: user.ID, Username: user.Username, Nickname: user.Nickname, Avatar: user.Avatar, Bio: user.Bio, Role: user.Role, CreatedAt: user.CreatedAt}
}

// Feed 社区信息流：最新按发布时间、热门按互动量加时间衰减、关注只看已关注作者
func (cc *CommunityController) Feed(c *gin.Context) {
	page, pageSize := pagination(c)
	query := cc.DB.Model(&models.Post{}).Where("posts.status = ? AND posts.moderation_status = ?", "published", "normal")
	if keyword := strings.TrimSpace(c.Query("q")); keyword != "" {
		query = postSearchCondition(query, keyword)
	}
	if tag := strings.TrimSpace(c.Query("tag")); tag != "" {
		query = query.Where("EXISTS (SELECT 1 FROM post_tags pt JOIN tags t ON t.id = pt.tag_id WHERE pt.post_id = posts.id AND (t.name = ? OR t.slug = ?))", tag, tag)
	}
	if category := strings.TrimSpace(c.Query("category")); category != "" {
		query = query.Where("EXISTS (SELECT 1 FROM categories cat WHERE cat.id = posts.category_id AND (cat.name = ? OR cat.slug = ?))", category, category)
	}
	mode := strings.TrimSpace(c.DefaultQuery("mode", "latest"))
	if mode != "latest" && mode != "hot" && mode != "following" && mode != "recommended" {
		mode = "latest"
	}
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
	if mode == "recommended" {
		userID := currentUserID(c)
		if userID == 0 {
			order = "((posts.likes_count * 4 + posts.comments_count * 6 + posts.views / 100.0) / max((julianday('now') - julianday(posts.published_at)) * 24.0, 2.0)) DESC, posts.published_at DESC, posts.id DESC"
		} else {
			// 预取用户兴趣画像（关注作者、点赞/收藏过的标签与分类），拼成常量 IN 列表：
			// 旧实现把 5 个相关子查询塞进每一行的评分表达式，数据量越大越慢；
			// 同时推荐候选收窄到近 90 天，老文章不参与评分，行数也降下来
			authorIDs, tagIDs, categoryIDs := cc.interestProfile(userID)
			query = query.Where("posts.published_at >= ?", time.Now().AddDate(0, 0, -90))
			order = fmt.Sprintf(`(
				CASE WHEN posts.author_id IN (%s) THEN 40 ELSE 0 END +
				(SELECT COUNT(*) * 8 FROM post_tags rec_pt WHERE rec_pt.post_id = posts.id AND rec_pt.tag_id IN (%s)) +
				CASE WHEN posts.category_id IS NOT NULL AND posts.category_id IN (%s) THEN 12 ELSE 0 END +
				posts.likes_count * 2 + posts.comments_count * 3 + posts.favorite_count * 2 + posts.views / 100.0
			) / max((julianday('now') - julianday(posts.published_at)) * 24.0, 2.0) DESC, posts.published_at DESC, posts.id DESC`, uintInList(authorIDs), uintInList(tagIDs), uintInList(categoryIDs))
		}
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
	liked, favorited := interactionSets(cc.DB, currentUserID(c), postIDs(posts))
	for _, post := range posts {
		item := toPostSummaryDTO(post)
		item.Liked = liked[post.ID]
		item.Favorited = favorited[post.ID]
		items = append(items, item)
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"items": items, "total": total, "page": page, "pageSize": pageSize, "mode": mode}})
}

// CreatePost 普通用户发文，标题与正文必填，链接标识为空时由标题生成
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
	status, ok := normalizeStatus(input.Status)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "文章状态无效"})
		return
	}
	validCategory, err := resolveCategory(cc.DB, input.CategoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "检查分类失败"})
		return
	}
	if !validCategory {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "分类不存在"})
		return
	}
	validSeries, err := resolveSeries(cc.DB, input.SeriesID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "检查系列失败"})
		return
	}
	if !validSeries {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "系列不存在或不属于你"})
		return
	}
	slug := input.Slug
	if slug == "" {
		slug = input.Title
	}
	post := models.Post{AuthorID: userID, Title: strings.TrimSpace(input.Title), Summary: strings.TrimSpace(input.Summary), Content: input.Content, CoverImage: strings.TrimSpace(input.CoverImage), Status: status, ReadingTime: readingTime(input.Content), CategoryID: input.CategoryID, SeriesID: input.SeriesID}
	if err := applyPostTiming(&post, status, input.ScheduledAt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := savePostWithSlugRetry(cc.DB, slugify(slug), &post, func(tx *gorm.DB) error {
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
	if err := cc.DB.Preload("Tags").Preload("Author").Preload("Category").First(&post, post.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取新文章失败"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "success", "data": toPostDTO(post)})
}

// UpdatePost 作者或管理员编辑文章，首次发布时通知粉丝
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
	status, ok := normalizeStatus(input.Status)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "文章状态无效"})
		return
	}
	validCategory, err := resolveCategory(cc.DB, input.CategoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "检查分类失败"})
		return
	}
	if !validCategory {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "分类不存在"})
		return
	}
	validSeries, err := resolveSeries(cc.DB, input.SeriesID, post.AuthorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "检查系列失败"})
		return
	}
	if !validSeries {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "系列不存在或不属于文章作者"})
		return
	}
	original := post
	slug := input.Slug
	if slug == "" {
		slug = post.Slug
	}
	post.Title = strings.TrimSpace(input.Title)
	post.Summary = strings.TrimSpace(input.Summary)
	post.Content = input.Content
	post.CoverImage = strings.TrimSpace(input.CoverImage)
	post.Status = status
	post.ReadingTime = readingTime(input.Content)
	post.CategoryID = input.CategoryID
	post.SeriesID = input.SeriesID
	if err := applyPostTiming(&post, status, input.ScheduledAt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := savePostWithSlugRetry(cc.DB, slugify(slug), &post, func(tx *gorm.DB) error {
		if err := savePostRevision(tx, original, userID); err != nil {
			return err
		}
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

// DeletePost 作者或管理员删除文章，级联清理评论、点赞、收藏与相关通知
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
	// 孤儿图片由每日清理任务统一回收（RunDataCleanup）
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// LikePost 点赞文章，重复点赞幂等返回
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

// UnlikePost 取消点赞
func (cc *CommunityController) UnlikePost(c *gin.Context) {
	userID := currentUserID(c)
	var post models.Post
	if err := cc.DB.Where("slug = ? AND status = ? AND moderation_status = ?", c.Param("slug"), "published", "normal").First(&post).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文章不存在"})
		return
	}
	likesCount := post.LikesCount
	if err := cc.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("user_id = ? AND post_id = ?", userID, post.ID).Delete(&models.PostLike{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			if err := tx.Model(&models.Post{}).Where("id = ? AND likes_count > 0", post.ID).UpdateColumn("likes_count", gorm.Expr("likes_count - 1")).Error; err != nil {
				return err
			}
		}
		return tx.Model(&models.Post{}).Select("likes_count").Where("id = ?", post.ID).Scan(&likesCount).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "取消点赞失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"liked": false, "likesCount": likesCount}})
}
