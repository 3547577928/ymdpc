package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"quietsignal/backend/models"
)

// TagInfoDTO 标签落地页头部信息
type TagInfoDTO struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	PostCount  int64  `json:"postCount"`
	Subscribed bool   `json:"subscribed"`
}

// TagInfo 标签信息与当前用户订阅状态
func (t *TagController) TagInfo(c *gin.Context) {
	var tag models.Tag
	if err := t.DB.Where("slug = ? OR name = ?", c.Param("slug"), c.Param("slug")).First(&tag).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "标签不存在"})
		return
	}
	var postCount int64
	if err := t.DB.Model(&models.Post{}).Joins("JOIN post_tags ON post_tags.post_id = posts.id").Where("post_tags.tag_id = ? AND posts.status = ? AND posts.moderation_status = ?", tag.ID, "published", "normal").Count(&postCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取标签失败"})
		return
	}
	subscribed := false
	if userID := currentUserID(c); userID > 0 {
		var count int64
		t.DB.Model(&models.TagSubscription{}).Where("user_id = ? AND tag_id = ?", userID, tag.ID).Count(&count)
		subscribed = count > 0
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": TagInfoDTO{ID: tag.ID, Name: tag.Name, Slug: tag.Slug, PostCount: postCount, Subscribed: subscribed}})
}

// SubscribeTag 订阅标签：新公开文章带该标签时收到站内通知；幂等
func (t *TagController) SubscribeTag(c *gin.Context) {
	userID := currentUserID(c)
	var tag models.Tag
	if err := t.DB.First(&tag, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "标签不存在"})
		return
	}
	subscription := models.TagSubscription{UserID: userID, TagID: tag.ID}
	if err := t.DB.Where(models.TagSubscription{UserID: userID, TagID: tag.ID}).FirstOrCreate(&subscription).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "订阅失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"subscribed": true}})
}

// UnsubscribeTag 取消订阅；幂等
func (t *TagController) UnsubscribeTag(c *gin.Context) {
	var tag models.Tag
	if err := t.DB.First(&tag, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "标签不存在"})
		return
	}
	if err := t.DB.Where("user_id = ? AND tag_id = ?", currentUserID(c), tag.ID).Delete(&models.TagSubscription{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "取消订阅失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"subscribed": false}})
}

type TagController struct{ DB *gorm.DB }

func (t *TagController) List(c *gin.Context) {
	var tags []models.Tag
	if err := t.DB.
		Joins("JOIN post_tags ON post_tags.tag_id = tags.id").
		Joins("JOIN posts ON posts.id = post_tags.post_id AND posts.status = ?", "published").
		Group("tags.id").
		Order("tags.name ASC").
		Find(&tags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取标签失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": tags})
}
