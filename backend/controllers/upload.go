package controllers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type UploadController struct {
	Dir string
}

var allowedImageExtensions = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}

func (u *UploadController) Image(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请选择图片文件"})
		return
	}
	if file.Size <= 0 || file.Size > 8<<20 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "图片大小不能超过 8MB"})
		return
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedImageExtensions[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "仅支持 JPG、PNG、GIF 或 WebP 图片"})
		return
	}
	name := fmt.Sprintf("%d-%d%s", time.Now().UnixNano(), currentUserID(c), ext)
	if err := c.SaveUploadedFile(file, filepath.Join(u.Dir, name)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "保存图片失败"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "success", "data": gin.H{"url": "/uploads/" + name}})
}
