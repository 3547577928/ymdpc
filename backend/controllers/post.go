package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"quietsignal/backend/models"
)

type PostController struct{ DB *gorm.DB }

type PostDTO struct {
	ID          uint      `json:"id"`
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	Summary     string    `json:"summary"`
	Content     string    `json:"content"`
	CoverImage  string    `json:"coverImage"`
	Tags        []string  `json:"tags"`
	Status      string    `json:"status"`
	Featured    bool      `json:"featured"`
	Views       int       `json:"views"`
	ReadingTime int       `json:"readingTime"`
	PublishedAt time.Time `json:"publishedAt"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
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
}

func (p *PostController) List(c *gin.Context) {
	p.list(c, "published", false)
}

func (p *PostController) AdminList(c *gin.Context) {
	p.list(c, strings.TrimSpace(c.Query("status")), true)
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
	if keyword := strings.TrimSpace(c.Query("q")); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("title LIKE ? OR summary LIKE ? OR content LIKE ?", like, like, like)
	}
	if tag := strings.TrimSpace(c.Query("tag")); tag != "" {
		query = query.Where("EXISTS (SELECT 1 FROM post_tags pt JOIN tags t ON t.id = pt.tag_id WHERE pt.post_id = posts.id AND (t.name = ? OR t.slug = ?))", tag, tag)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取文章失败"})
		return
	}
	var posts []models.Post
	if err := query.Preload("Tags").Order("published_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取文章失败"})
		return
	}
	items := make([]PostDTO, 0, len(posts))
	for _, post := range posts {
		items = append(items, toPostDTO(post))
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"items": items, "total": total, "page": page, "pageSize": pageSize}})
}

func (p *PostController) Detail(c *gin.Context) {
	var post models.Post
	if err := p.DB.Preload("Tags").Where("slug = ? AND status = ?", c.Param("slug"), "published").First(&post).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文章不存在"})
		return
	}
	if err := p.DB.Model(&models.Post{}).Where("id = ?", post.ID).UpdateColumn("views", gorm.Expr("views + ?", 1)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新阅读量失败"})
		return
	}
	post.Views++
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": toPostDTO(post)})
}

func (p *PostController) AdminDetail(c *gin.Context) {
	var post models.Post
	if err := p.DB.Preload("Tags").First(&post, c.Param("id")).Error; err != nil {
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
	status := normalizeStatus(input.Status)
	slug := input.Slug
	if slug == "" {
		slug = input.Title
	}
	slug = uniqueSlug(p.DB, slugify(slug), 0)
	post := models.Post{Title: strings.TrimSpace(input.Title), Slug: slug, Summary: input.Summary, Content: input.Content, CoverImage: input.CoverImage, Status: status, Featured: input.Featured, ReadingTime: readingTime(input.Content)}
	if status == "published" {
		post.PublishedAt = time.Now()
	}
	if err := p.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&post).Error; err != nil {
			return err
		}
		return replaceTags(tx, &post, input.Tags)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建文章失败"})
		return
	}
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
	status := normalizeStatus(input.Status)
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
	if status == "published" && post.PublishedAt.IsZero() {
		post.PublishedAt = time.Now()
	}
	if err := p.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&post).Error; err != nil {
			return err
		}
		return replaceTags(tx, &post, input.Tags)
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
		return tx.Delete(&post).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除文章失败"})
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
	status := normalizeStatus(input.Status)
	updates := map[string]any{"status": status}
	if status == "published" {
		updates["published_at"] = time.Now()
	}
	result := p.DB.Model(&models.Post{}).Where("id = ?", c.Param("id")).Updates(updates)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新状态失败"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文章不存在"})
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

func normalizeStatus(status string) string {
	switch status {
	case "published", "archived":
		return status
	default:
		return "draft"
	}
}

func toPostDTO(post models.Post) PostDTO {
	tags := make([]string, 0, len(post.Tags))
	for _, tag := range post.Tags {
		tags = append(tags, tag.Name)
	}
	return PostDTO{ID: post.ID, Title: post.Title, Slug: post.Slug, Summary: post.Summary, Content: post.Content, CoverImage: post.CoverImage, Tags: tags, Status: post.Status, Featured: post.Featured, Views: post.Views, ReadingTime: post.ReadingTime, PublishedAt: post.PublishedAt, CreatedAt: post.CreatedAt, UpdatedAt: post.UpdatedAt}
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
