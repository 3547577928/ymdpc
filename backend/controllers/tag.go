package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"quietsignal/backend/models"
)

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
