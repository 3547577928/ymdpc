package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"quietsignal/backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PostController struct {
	DB        *gorm.DB
	UploadDir string
}

// interactionSets 批量查询当前用户对一组文章的点赞与收藏状态。
// 列表接口逐篇文章 COUNT 会产生 N+1 查询（一页 12 篇就多 12 次查询），
// 这里合并为两次 IN 查询，由调用方在渲染 DTO 时查表填充
func interactionSets(db *gorm.DB, userID uint, postIDs []uint) (liked, favorited map[uint]bool) {
	liked = map[uint]bool{}
	favorited = map[uint]bool{}
	if userID == 0 || len(postIDs) == 0 {
		return liked, favorited
	}
	var likedIDs, favoritedIDs []uint
	db.Model(&models.PostLike{}).Where("user_id = ? AND post_id IN ?", userID, postIDs).Pluck("post_id", &likedIDs)
	db.Model(&models.Favorite{}).Where("user_id = ? AND post_id IN ?", userID, postIDs).Pluck("post_id", &favoritedIDs)
	for _, id := range likedIDs {
		liked[id] = true
	}
	for _, id := range favoritedIDs {
		favorited[id] = true
	}
	return liked, favorited
}

// postIDs 提取文章 ID 集合，用于 interactionSets 的批量查询
func postIDs(posts []models.Post) []uint {
	ids := make([]uint, 0, len(posts))
	for _, post := range posts {
		ids = append(ids, post.ID)
	}
	return ids
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
	ScheduledAt      *time.Time   `json:"scheduledAt,omitempty"`
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
	Title       string     `json:"title" binding:"required"`
	Slug        string     `json:"slug"`
	Summary     string     `json:"summary"`
	Content     string     `json:"content"`
	CoverImage  string     `json:"coverImage"`
	Tags        []string   `json:"tags"`
	Status      string     `json:"status"`
	Featured    bool       `json:"featured"`
	CategoryID  *uint      `json:"categoryId"`
	ScheduledAt *time.Time `json:"scheduledAt"`
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
	if !includeDrafts || status == "published" || status == "scheduled" || status == "draft" || status == "archived" {
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
		like := likePattern(keyword)
		query = query.Where("title LIKE ? ESCAPE '\\' OR summary LIKE ? ESCAPE '\\' OR content LIKE ? ESCAPE '\\'", like, like, like)
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
	liked, favorited := interactionSets(p.DB, currentUserID(c), postIDs(posts))
	for _, post := range posts {
		item := toPostSummaryDTO(post)
		item.Liked = liked[post.ID]
		item.Favorited = favorited[post.ID]
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
	liked, favorited := interactionSets(p.DB, currentUserID(c), []uint{post.ID})
	detail.Liked = liked[post.ID]
	detail.Favorited = favorited[post.ID]
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
	validCategory, err := resolveCategory(p.DB, input.CategoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "检查分类失败"})
		return
	}
	if !validCategory {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "分类不存在"})
		return
	}
	post := models.Post{AuthorID: currentUserID(c), Title: strings.TrimSpace(input.Title), Summary: input.Summary, Content: input.Content, CoverImage: input.CoverImage, Status: status, Featured: input.Featured, ReadingTime: readingTime(input.Content), CategoryID: input.CategoryID}
	if err := applyPostTiming(&post, status, input.ScheduledAt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := savePostWithSlugRetry(p.DB, slugify(slug), &post, func(tx *gorm.DB) error {
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
	if err := p.DB.Preload("Tags").Preload("Author").First(&post, post.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取新文章失败"})
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
	wasPublished := post.Status == "published"
	status, ok := normalizeStatus(input.Status)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "文章状态无效"})
		return
	}
	validCategory, err := resolveCategory(p.DB, input.CategoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "检查分类失败"})
		return
	}
	if !validCategory {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "分类不存在"})
		return
	}
	original := post
	slug := input.Slug
	if slug == "" {
		slug = post.Slug
	}
	post.Title = strings.TrimSpace(input.Title)
	post.Summary = input.Summary
	post.Content = input.Content
	post.CoverImage = input.CoverImage
	post.Status = status
	post.Featured = input.Featured
	post.ReadingTime = readingTime(input.Content)
	post.CategoryID = input.CategoryID
	if err := applyPostTiming(&post, status, input.ScheduledAt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := savePostWithSlugRetry(p.DB, slugify(slug), &post, func(tx *gorm.DB) error {
		if err := savePostRevision(tx, original, currentUserID(c)); err != nil {
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
	if p.UploadDir != "" {
		if _, err := cleanupUnreferencedUploads(p.UploadDir, p.DB); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "文章已删除，但图片清理失败"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// UpdateModeration 管理员下架/恢复文章（moderation_status: normal 正常 / hidden 下架）
func (p *PostController) UpdateModeration(c *gin.Context) {
	var input struct {
		Status      string     `json:"status" binding:"required"`
		ScheduledAt *time.Time `json:"scheduledAt"`
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
		Status      string     `json:"status" binding:"required"`
		ScheduledAt *time.Time `json:"scheduledAt"`
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
	previousStatus := post.Status
	if err := applyPostTiming(&post, status, input.ScheduledAt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	var scheduledValue any = post.ScheduledAt
	if post.ScheduledAt == nil {
		scheduledValue = gorm.Expr("NULL")
	}
	updates := map[string]any{"status": post.Status, "published_at": post.PublishedAt, "scheduled_at": scheduledValue}
	shouldNotify := previousStatus != "published" && status == "published"
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
	case "published", "scheduled", "archived":
		return strings.TrimSpace(status), true
	case "draft":
		return "draft", true
	}
	return "", false
}

func applyPostTiming(post *models.Post, status string, scheduledAt *time.Time) error {
	post.Status = status
	if status == "scheduled" {
		if scheduledAt == nil || !scheduledAt.After(time.Now()) {
			return errors.New("定时发布时间必须晚于当前时间")
		}
		post.ScheduledAt = scheduledAt
		post.PublishedAt = time.Time{}
		return nil
	}
	post.ScheduledAt = nil
	if status == "published" && post.PublishedAt.IsZero() {
		post.PublishedAt = time.Now()
	}
	return nil
}

func toPostDTO(post models.Post) PostDTO {
	return PostDTO{PostSummaryDTO: toPostSummaryDTO(post), Content: post.Content}
}

func toPostSummaryDTO(post models.Post) PostSummaryDTO {
	tags := make([]string, 0, len(post.Tags))
	for _, tag := range post.Tags {
		tags = append(tags, tag.Name)
	}
	return PostSummaryDTO{ID: post.ID, Author: toUserDTO(post.Author), Title: post.Title, Slug: post.Slug, Summary: post.Summary, CoverImage: post.CoverImage, Tags: tags, Status: post.Status, ModerationStatus: post.ModerationStatus, Category: toCategoryDTO(post.Category), Featured: post.Featured, Views: post.Views, LikesCount: post.LikesCount, FavoriteCount: post.FavoriteCount, CommentsCount: post.CommentsCount, ReadingTime: post.ReadingTime, PublishedAt: post.PublishedAt, ScheduledAt: post.ScheduledAt, CreatedAt: post.CreatedAt, UpdatedAt: post.UpdatedAt}
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
			slug, err := uniqueTagSlug(tx, slugify(name))
			if err != nil {
				return err
			}
			tag = models.Tag{Name: name, Slug: slug}
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

func uniqueTagSlug(db *gorm.DB, base string) (string, error) {
	if base == "" {
		base = "tag"
	}
	candidate := base
	for i := 2; ; i++ {
		var count int64
		if err := db.Model(&models.Tag{}).Where("slug = ?", candidate).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}

func uniqueSlug(db *gorm.DB, base string, id uint) (string, error) {
	if base == "" {
		base = "untitled-post"
	}
	candidate := base
	for i := 2; ; i++ {
		var count int64
		if err := db.Model(&models.Post{}).Where("slug = ? AND id <> ?", candidate, id).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}

// savePostWithSlugRetry 在事务中保存文章。slug 唯一索引冲突只会来自并发写入
// （相同标题/链接标识同时创建），此时基于基础 slug 重新生成后缀再重试一次，
// 把并发冲突收敛为重试而不是向用户暴露「创建文章失败」；save 必须只是写库，
// 不能包含面向客户端的响应，否则重试会写出两份响应
func savePostWithSlugRetry(db *gorm.DB, base string, post *models.Post, save func(tx *gorm.DB) error) error {
	slug, err := uniqueSlug(db, base, post.ID)
	if err != nil {
		return err
	}
	post.Slug = slug
	err = db.Transaction(save)
	if !errors.Is(err, gorm.ErrDuplicatedKey) {
		return err
	}
	slug, err = uniqueSlug(db, base, post.ID)
	if err != nil {
		return err
	}
	post.Slug = slug
	return db.Transaction(save)
}

// slugify 把标题转换为 URL 安全的文章标识：仅保留字母、数字和中日韩字符，
// 其余字符（含 ?、#、% 等保留字符）统一折叠为连字符，避免生成的链接被前端
// 或浏览器解析成 query string 而打不开文章；同时限制长度防止超出字段上限
func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	pendingDash := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			// 连字符只在一串非法字符的首个位置写入，后续连续非法字符不再重复写
			builder.WriteRune(r)
			pendingDash = false
		case !pendingDash:
			builder.WriteByte('-')
			pendingDash = true
		}
	}
	slug := strings.Trim(builder.String(), "-")
	// slug 字段上限 180，这里按字符数截断（避免按字节切坏多字节字符），余量留给 uniqueSlug 追加序号
	if utf8.RuneCountInString(slug) > 80 {
		slug = strings.Trim(string([]rune(slug)[:80]), "-")
	}
	return slug
}

// likePattern 构造 LIKE 匹配式并转义通配符：用户输入的 % 和 _ 有特殊语义，
// 不转义时搜索 "%" 会匹配全表、"_" 会匹配任意单字符；转义符用反斜杠，
// 对应 SQL 需要加 ESCAPE '\' 子句
func likePattern(keyword string) string {
	escaped := strings.ReplaceAll(keyword, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, "%", `\%`)
	escaped = strings.ReplaceAll(escaped, "_", `\_`)
	return "%" + escaped + "%"
}

func readingTime(content string) int {
	count := utf8.RuneCountInString(strings.TrimSpace(content))
	minutes := count / 400
	if minutes < 1 {
		return 1
	}
	return minutes + 1
}
