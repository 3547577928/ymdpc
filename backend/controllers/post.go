package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"quietsignal/backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PostController struct{ DB *gorm.DB }

func (p *PostController) isPostLiked(c *gin.Context, postID uint) bool {
	userID := currentUserID(c)
	if userID == 0 {
		return false
	}
	var count int64
	p.DB.Model(&models.PostLike{}).Where("user_id = ? AND post_id = ?", userID, postID).Count(&count)
	return count > 0
}

// isPostFavorited 判断当前用户是否收藏了文章
func (p *PostController) isPostFavorited(c *gin.Context, postID uint) bool {
	userID := currentUserID(c)
	if userID == 0 {
		return false
	}
	var count int64
	p.DB.Model(&models.Favorite{}).Where("user_id = ? AND post_id = ?", userID, postID).Count(&count)
	return count > 0
}

type CategoryDTO struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type PostSummaryDTO struct {
	ID               uint         `json:"id"`
	Author           UserDTO      `json:"author"`
	Title            string       `json:"title"`
	Slug             string       `json:"slug"`
	Summary          string       `json:"summary"`
	CoverImage       string       `json:"coverImage"`
	Tags             []string     `json:"tags"`
	Status           string       `json:"status"`
	ModerationStatus string       `json:"moderationStatus"`
	Category         *CategoryDTO `json:"category,omitempty"`
	Featured         bool         `json:"featured"`
	Views            int          `json:"views"`
	LikesCount       int          `json:"likesCount"`
	FavoriteCount    int          `json:"favoriteCount"`
	CommentsCount    int          `json:"commentsCount"`
	Liked            bool         `json:"liked"`
	Favorited        bool         `json:"favorited"`
	ReadingTime      int          `json:"readingTime"`
	PublishedAt      time.Time    `json:"publishedAt"`
	CreatedAt        time.Time    `json:"createdAt"`
	UpdatedAt        time.Time    `json:"updatedAt"`
}

type PostDTO struct {
	PostSummaryDTO
	Content string `json:"content"`
}

type AdjacentPostDTO struct {
	Title string `json:"title"`
	Slug  string `json:"slug"`
}

type PostDetailDTO struct {
	Post     PostDTO          `json:"post"`
	Previous *AdjacentPostDTO `json:"previous"`
	Next     *AdjacentPostDTO `json:"next"`
}

type PostRequest struct {
	Title      string   `json:"title" binding:"required"`
	Slug       string   `json:"slug"`
	Summary    string   `json:"summary"`
	Content    string   `json:"content"`
	CoverImage string   `json:"coverImage"`
	Tags       []string `json:"tags"`
	Status     string   `json:"status"`
	Featured   bool     `json:"featured"`
	CategoryID *uint    `json:"categoryId"`
}

func (p *PostController) List(c *gin.Context) {
	p.list(c, "published", false)
}

func (p *PostController) AdminList(c *gin.Context) {
	status := strings.TrimSpace(c.Query("status"))
	if status == "" {
		status = "all"
	}
	p.list(c, status, true)
}

func (p *PostController) list(c *gin.Context, status string, includeDrafts bool) {
	page, pageSize := pagination(c)
	query := p.DB.Model(&models.Post{})
	if !includeDrafts || status == "published" || status == "draft" || status == "archived" {
		if status == "" {
			status = "published"
		}
		query = query.Where("status = ?", status)
	}
	if !includeDrafts {
		// 公开列表只展示未被下架的文章
		query = query.Where("moderation_status = ?", "normal")
	}
	if keyword := strings.TrimSpace(c.Query("q")); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("title LIKE ? OR summary LIKE ? OR content LIKE ?", like, like, like)
	}
	if tag := strings.TrimSpace(c.Query("tag")); tag != "" {
		query = query.Where("EXISTS (SELECT 1 FROM post_tags pt JOIN tags t ON t.id = pt.tag_id WHERE pt.post_id = posts.id AND (t.name = ? OR t.slug = ?))", tag, tag)
	}
	if category := strings.TrimSpace(c.Query("category")); category != "" {
		query = query.Where("EXISTS (SELECT 1 FROM categories cat WHERE cat.id = posts.category_id AND (cat.name = ? OR cat.slug = ?))", category, category)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取文章失败"})
		return
	}
	var posts []models.Post
	if err := query.Omit("content").Preload("Tags").Preload("Author").Preload("Category").Order("published_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取文章失败"})
		return
	}
	items := make([]PostSummaryDTO, 0, len(posts))
	for _, post := range posts {
		item := toPostSummaryDTO(post)
		item.Liked = p.isPostLiked(c, post.ID)
		items = append(items, item)
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"items": items, "total": total, "page": page, "pageSize": pageSize}})
}

func (p *PostController) Detail(c *gin.Context) {
	var post models.Post
	if err := p.DB.Preload("Tags").Preload("Author").Preload("Category").Where("slug = ? AND status = ? AND moderation_status = ?", c.Param("slug"), "published", "normal").First(&post).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文章不存在"})
		return
	}
	previous, err := adjacentPost(p.DB, post, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取相邻文章失败"})
		return
	}
	next, err := adjacentPost(p.DB, post, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取相邻文章失败"})
		return
	}
	detail := toPostDTO(post)
	detail.Liked = p.isPostLiked(c, post.ID)
	detail.Favorited = p.isPostFavorited(c, post.ID)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": PostDetailDTO{Post: detail, Previous: previous, Next: next}})
}

func (p *PostController) RecordView(c *gin.Context) {
	result := p.DB.Model(&models.Post{}).Where("slug = ? AND status = ? AND moderation_status = ?", c.Param("slug"), "published", "normal").UpdateColumn("views", gorm.Expr("views + ?", 1))
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新阅读量失败"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文章不存在"})
		return
	}
	var post models.Post
	if err := p.DB.Select("views").Where("slug = ?", c.Param("slug")).First(&post).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取阅读量失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"views": post.Views}})
}

func (p *PostController) AdminStats(c *gin.Context) {
	var stats struct {
		Total     int64 `json:"total"`
		Published int64 `json:"published"`
		Views     int64 `json:"views"`
	}
	if err := p.DB.Model(&models.Post{}).
		Select("COUNT(*) AS total, COALESCE(SUM(CASE WHEN status = 'published' THEN 1 ELSE 0 END), 0) AS published, COALESCE(SUM(views), 0) AS views").
		Scan(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取统计失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": stats})
}

func (p *PostController) AdminDetail(c *gin.Context) {
	var post models.Post
	if err := p.DB.Preload("Tags").Preload("Author").Preload("Category").First(&post, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文章不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": toPostDTO(post)})
}

func (p *PostController) Create(c *gin.Context) {
	var input PostRequest
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Title) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "文章标题不能为空"})
		return
	}
	status, ok := normalizeStatus(input.Status)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "文章状态无效"})
		return
	}
	slug := input.Slug
	if slug == "" {
		slug = input.Title
	}
	if !resolveCategory(p.DB, input.CategoryID) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "分类不存在"})
		return
	}
	slug = uniqueSlug(p.DB, slugify(slug), 0)
	post := models.Post{AuthorID: currentUserID(c), Title: strings.TrimSpace(input.Title), Slug: slug, Summary: input.Summary, Content: input.Content, CoverImage: input.CoverImage, Status: status, Featured: input.Featured, ReadingTime: readingTime(input.Content), CategoryID: input.CategoryID}
	if status == "published" {
		post.PublishedAt = time.Now()
	}
	if err := p.DB.Transaction(func(tx *gorm.DB) error {
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
	p.DB.Preload("Tags").Preload("Author").First(&post, post.ID)
	c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "success", "data": toPostDTO(post)})
}

func (p *PostController) Update(c *gin.Context) {
	var input PostRequest
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Title) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "文章标题不能为空"})
		return
	}
	var post models.Post
	if err := p.DB.Preload("Tags").First(&post, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文章不存在"})
		return
	}
	wasPublished := post.Status == "published"
	status, ok := normalizeStatus(input.Status)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "文章状态无效"})
		return
	}
	if !resolveCategory(p.DB, input.CategoryID) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "分类不存在"})
		return
	}
	slug := input.Slug
	if slug == "" {
		slug = post.Slug
	}
	post.Title = strings.TrimSpace(input.Title)
	post.Slug = uniqueSlug(p.DB, slugify(slug), post.ID)
	post.Summary = input.Summary
	post.Content = input.Content
	post.CoverImage = input.CoverImage
	post.Status = status
	post.Featured = input.Featured
	post.ReadingTime = readingTime(input.Content)
	post.CategoryID = input.CategoryID
	if status == "published" && post.PublishedAt.IsZero() {
		post.PublishedAt = time.Now()
	}
	if err := p.DB.Transaction(func(tx *gorm.DB) error {
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

func (p *PostController) Delete(c *gin.Context) {
	var post models.Post
	if err := p.DB.First(&post, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文章不存在"})
		return
	}
	if err := p.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&post).Association("Tags").Clear(); err != nil {
			return err
		}
		if err := deletePostRelations(tx, post.ID); err != nil {
			return err
		}
		if err := tx.Delete(&post).Error; err != nil {
			return err
		}
		return writeAdminLog(tx, currentUserID(c), "post.delete", "post", post.ID, post.Title)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除文章失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// UpdateModeration 管理员下架/恢复文章（moderation_status: normal 正常 / hidden 下架）
func (p *PostController) UpdateModeration(c *gin.Context) {
	var input struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || (input.Status != "normal" && input.Status != "hidden") {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "审核状态无效"})
		return
	}
	var post models.Post
	if err := p.DB.First(&post, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文章不存在"})
		return
	}
	if err := p.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&post).Update("moderation_status", input.Status).Error; err != nil {
			return err
		}
		// 下架/恢复操作保留管理日志
		if input.Status == "hidden" {
			return writeAdminLog(tx, currentUserID(c), "post.takedown", "post", post.ID, post.Title)
		}
		return writeAdminLog(tx, currentUserID(c), "post.restore", "post", post.ID, post.Title)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新审核状态失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

func (p *PostController) UpdateStatus(c *gin.Context) {
	var input struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "状态不能为空"})
		return
	}
	status, ok := normalizeStatus(input.Status)
	if !ok || strings.TrimSpace(input.Status) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "文章状态无效"})
		return
	}
	var post models.Post
	if err := p.DB.First(&post, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文章不存在"})
		return
	}
	updates := map[string]any{"status": status}
	if status == "published" && post.PublishedAt.IsZero() {
		updates["published_at"] = time.Now()
	}
	shouldNotify := post.Status != "published" && status == "published"
	if err := p.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&post).Updates(updates).Error; err != nil {
			return err
		}
		if shouldNotify {
			return notifyFollowersOfPost(tx, post.AuthorID, post.ID)
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新状态失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

func pagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	return page, pageSize
}

func normalizeStatus(status string) (string, bool) {
	switch strings.TrimSpace(status) {
	case "":
		return "draft", true
	case "published", "archived":
		return strings.TrimSpace(status), true
	case "draft":
		return "draft", true
	}
	return "", false
}

func toPostDTO(post models.Post) PostDTO {
	return PostDTO{PostSummaryDTO: toPostSummaryDTO(post), Content: post.Content}
}

func toPostSummaryDTO(post models.Post) PostSummaryDTO {
	tags := make([]string, 0, len(post.Tags))
	for _, tag := range post.Tags {
		tags = append(tags, tag.Name)
	}
	return PostSummaryDTO{ID: post.ID, Author: toUserDTO(post.Author), Title: post.Title, Slug: post.Slug, Summary: post.Summary, CoverImage: post.CoverImage, Tags: tags, Status: post.Status, ModerationStatus: post.ModerationStatus, Category: toCategoryDTO(post.Category), Featured: post.Featured, Views: post.Views, LikesCount: post.LikesCount, FavoriteCount: post.FavoriteCount, CommentsCount: post.CommentsCount, ReadingTime: post.ReadingTime, PublishedAt: post.PublishedAt, CreatedAt: post.CreatedAt, UpdatedAt: post.UpdatedAt}
}

func toCategoryDTO(category *models.Category) *CategoryDTO {
	if category == nil {
		return nil
	}
	return &CategoryDTO{ID: category.ID, Name: category.Name, Slug: category.Slug}
}

func adjacentPost(db *gorm.DB, current models.Post, newer bool) (*AdjacentPostDTO, error) {
	comparison, order := "<", "published_at DESC, id DESC"
	if newer {
		comparison, order = ">", "published_at ASC, id ASC"
	}
	var post models.Post
	err := db.Select("title", "slug").
		Where("status = ? AND moderation_status = ? AND (published_at "+comparison+" ? OR (published_at = ? AND id "+comparison+" ?))", "published", "normal", current.PublishedAt, current.PublishedAt, current.ID).
		Order(order).
		First(&post).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &AdjacentPostDTO{Title: post.Title, Slug: post.Slug}, nil
}

func replaceTags(tx *gorm.DB, post *models.Post, names []string) error {
	seen := map[string]bool{}
	tags := make([]models.Tag, 0, len(names))
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		var tag models.Tag
		err := tx.Where("name = ?", name).First(&tag).Error
		if err == gorm.ErrRecordNotFound {
			tag = models.Tag{Name: name, Slug: uniqueTagSlug(tx, slugify(name))}
			if err := tx.Create(&tag).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		tags = append(tags, tag)
	}
	if err := tx.Model(post).Association("Tags").Replace(tags); err != nil {
		return err
	}
	post.Tags = tags
	return nil
}

func uniqueTagSlug(db *gorm.DB, base string) string {
	if base == "" {
		base = "tag"
	}
	candidate := base
	for i := 2; ; i++ {
		var count int64
		db.Model(&models.Tag{}).Where("slug = ?", candidate).Count(&count)
		if count == 0 {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}

func uniqueSlug(db *gorm.DB, base string, id uint) string {
	if base == "" {
		base = "untitled-post"
	}
	candidate := base
	for i := 2; ; i++ {
		var count int64
		db.Model(&models.Post{}).Where("slug = ? AND id <> ?", candidate, id).Count(&count)
		if count == 0 {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.NewReplacer(" ", "-", "/", "-", "_", "-").Replace(value)
	return value
}

func readingTime(content string) int {
	count := utf8.RuneCountInString(strings.TrimSpace(content))
	minutes := count / 400
	if minutes < 1 {
		return 1
	}
	return minutes + 1
}
