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

// 文章系列/专栏：同一作者把一组文章编成有序目录
type SeriesController struct{ DB *gorm.DB }

type SeriesBriefDTO struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
	Slug  string `json:"slug"`
}

type SeriesListItemDTO struct {
	SeriesBriefDTO
	Description string    `json:"description"`
	PostCount   int64     `json:"postCount"`
	CreatedAt   time.Time `json:"createdAt"`
}

type SeriesDetailDTO struct {
	SeriesBriefDTO
	Description string            `json:"description"`
	Author      UserDTO           `json:"author"`
	Posts       []AdjacentPostDTO `json:"posts"`
}

// resolveSeries 校验系列存在且属于指定作者；seriesID 为空时合法（不属于任何系列）
func resolveSeries(db *gorm.DB, seriesID *uint, authorID uint) (bool, error) {
	if seriesID == nil {
		return true, nil
	}
	var count int64
	err := db.Model(&models.Series{}).Where("id = ? AND author_id = ?", *seriesID, authorID).Count(&count).Error
	return count > 0, err
}

// Create 创建系列（作者本人），slug 从标题生成并保证唯一
func (sc *SeriesController) Create(c *gin.Context) {
	userID := currentUserID(c)
	var input struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Title) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "系列标题不能为空"})
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	if len(input.Title) > 120 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "系列标题不能超过 120 字"})
		return
	}
	slug := slugify(input.Title)
	if slug == "" {
		slug = "series"
	}
	// 系列 slug 全站唯一：冲突时追加序号
	candidate := slug
	for suffix := 2; ; suffix++ {
		var count int64
		if err := sc.DB.Model(&models.Series{}).Where("slug = ?", candidate).Count(&count).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建系列失败"})
			return
		}
		if count == 0 {
			break
		}
		candidate = slug + "-" + strconv.Itoa(suffix)
	}
	series := models.Series{AuthorID: userID, Title: input.Title, Slug: candidate, Description: strings.TrimSpace(input.Description)}
	if err := sc.DB.Create(&series).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建系列失败"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "success", "data": SeriesListItemDTO{SeriesBriefDTO: SeriesBriefDTO{ID: series.ID, Title: series.Title, Slug: series.Slug}, Description: series.Description, CreatedAt: series.CreatedAt}})
}

// Detail 系列目录页：系列信息 + 公开文章（按发布时间正序，即系列阅读顺序）
func (sc *SeriesController) Detail(c *gin.Context) {
	var series models.Series
	if err := sc.DB.Preload("Author").Where("slug = ?", c.Param("slug")).First(&series).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "系列不存在"})
		return
	}
	var posts []models.Post
	if err := sc.DB.Select("id", "title", "slug", "published_at").Where("series_id = ? AND status = ? AND moderation_status = ?", series.ID, "published", "normal").Order("published_at ASC, id ASC").Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取系列文章失败"})
		return
	}
	items := make([]AdjacentPostDTO, 0, len(posts))
	for _, post := range posts {
		items = append(items, AdjacentPostDTO{Title: post.Title, Slug: post.Slug})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": SeriesDetailDTO{
		SeriesBriefDTO: SeriesBriefDTO{ID: series.ID, Title: series.Title, Slug: series.Slug},
		Description:    series.Description,
		Author:         toUserDTO(series.Author),
		Posts:          items,
	}})
}

// AuthorSeries 作者的系列列表（含公开文章数），用于写作页选择与个人主页
func (sc *SeriesController) AuthorSeries(c *gin.Context) {
	var user models.User
	if err := sc.DB.Where("username = ? AND status = ?", c.Param("username"), "active").First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}
	var seriesList []models.Series
	if err := sc.DB.Where("author_id = ?", user.ID).Order("created_at DESC, id DESC").Find(&seriesList).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取系列失败"})
		return
	}
	counts := map[uint]int64{}
	if len(seriesList) > 0 {
		ids := make([]uint, 0, len(seriesList))
		for _, series := range seriesList {
			ids = append(ids, series.ID)
		}
		rows := []struct {
			SeriesID uint
			Count    int64
		}{}
		if err := sc.DB.Model(&models.Post{}).Select("series_id", "COUNT(*) AS count").Where("series_id IN ? AND status = ? AND moderation_status = ?", ids, "published", "normal").Group("series_id").Scan(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取系列失败"})
			return
		}
		for _, row := range rows {
			counts[row.SeriesID] = row.Count
		}
	}
	items := make([]SeriesListItemDTO, 0, len(seriesList))
	for _, series := range seriesList {
		items = append(items, SeriesListItemDTO{SeriesBriefDTO: SeriesBriefDTO{ID: series.ID, Title: series.Title, Slug: series.Slug}, Description: series.Description, PostCount: counts[series.ID], CreatedAt: series.CreatedAt})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": items})
}
