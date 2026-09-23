package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"quietsignal/backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PostRevisionDTO struct {
	ID         uint      `json:"id"`
	Title      string    `json:"title"`
	Slug       string    `json:"slug"`
	Summary    string    `json:"summary"`
	Content    string    `json:"content"`
	CoverImage string    `json:"coverImage"`
	CategoryID *uint     `json:"categoryId"`
	Featured   bool      `json:"featured"`
	Tags       []string  `json:"tags"`
	CreatedAt  time.Time `json:"createdAt"`
}

func savePostRevision(tx *gorm.DB, post models.Post, editorID uint) error {
	tags := make([]string, 0, len(post.Tags))
	for _, tag := range post.Tags {
		tags = append(tags, tag.Name)
	}
	encodedTags, err := json.Marshal(tags)
	if err != nil {
		return err
	}
	revision := models.PostRevision{PostID: post.ID, EditorID: editorID, Title: post.Title, Slug: post.Slug, Summary: post.Summary, Content: post.Content, CoverImage: post.CoverImage, CategoryID: post.CategoryID, Featured: post.Featured, TagsJSON: string(encodedTags)}
	return tx.Create(&revision).Error
}

func toPostRevisionDTO(revision models.PostRevision) (PostRevisionDTO, error) {
	var tags []string
	if err := json.Unmarshal([]byte(revision.TagsJSON), &tags); err != nil {
		return PostRevisionDTO{}, err
	}
	return PostRevisionDTO{ID: revision.ID, Title: revision.Title, Slug: revision.Slug, Summary: revision.Summary, Content: revision.Content, CoverImage: revision.CoverImage, CategoryID: revision.CategoryID, Featured: revision.Featured, Tags: tags, CreatedAt: revision.CreatedAt}, nil
}

func canEditPost(c *gin.Context, post models.Post) bool {
	return post.AuthorID == currentUserID(c) || currentUserIsAdmin(c)
}

func (cc *CommunityController) PostRevisions(c *gin.Context) {
	var post models.Post
	if err := cc.DB.First(&post, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文章不存在"})
		return
	}
	if !canEditPost(c, post) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只能查看自己的文章版本"})
		return
	}
	var revisions []models.PostRevision
	if err := cc.DB.Where("post_id = ?", post.ID).Order("created_at DESC, id DESC").Limit(30).Find(&revisions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取版本历史失败"})
		return
	}
	items := make([]PostRevisionDTO, 0, len(revisions))
	for _, revision := range revisions {
		item, err := toPostRevisionDTO(revision)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取版本历史失败"})
			return
		}
		items = append(items, item)
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": items})
}

func (cc *CommunityController) RestorePostRevision(c *gin.Context) {
	var post models.Post
	if err := cc.DB.Preload("Tags").Preload("Author").Preload("Category").First(&post, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文章不存在"})
		return
	}
	if !canEditPost(c, post) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只能恢复自己的文章版本"})
		return
	}
	var revision models.PostRevision
	if err := cc.DB.Where("id = ? AND post_id = ?", c.Param("revisionId"), post.ID).First(&revision).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文章版本不存在"})
		return
	}
	revisionDTO, err := toPostRevisionDTO(revision)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取文章版本失败"})
		return
	}
	if revision.CategoryID != nil {
		valid, err := resolveCategory(cc.DB, revision.CategoryID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "检查历史分类失败"})
			return
		}
		if !valid {
			revision.CategoryID = nil
		}
	}
	if err := cc.DB.Transaction(func(tx *gorm.DB) error {
		if err := savePostRevision(tx, post, currentUserID(c)); err != nil {
			return err
		}
		slug, err := uniqueSlug(tx, revision.Slug, post.ID)
		if err != nil {
			return err
		}
		post.Title = revision.Title
		post.Slug = slug
		post.Summary = revision.Summary
		post.Content = revision.Content
		post.CoverImage = revision.CoverImage
		post.CategoryID = revision.CategoryID
		post.Featured = revision.Featured
		post.ReadingTime = readingTime(revision.Content)
		if err := tx.Save(&post).Error; err != nil {
			return err
		}
		return replaceTags(tx, &post, revisionDTO.Tags)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "恢复文章版本失败"})
		return
	}
	if err := cc.DB.Preload("Tags").Preload("Author").Preload("Category").First(&post, post.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取恢复后的文章失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": toPostDTO(post)})
}
