package controllers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"quietsignal/backend/models"
)

type UploadController struct {
	Dir string
	DB  *gorm.DB
}

var allowedImageExtensions = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}
var allowedImageMIMEs = map[string]bool{"image/jpeg": true, "image/png": true, "image/gif": true, "image/webp": true}
var uploadReferencePattern = regexp.MustCompile(`(?:^|/)uploads/([A-Za-z0-9][A-Za-z0-9._-]*)`)

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
	opened, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无法读取图片文件"})
		return
	}
	header := make([]byte, 512)
	read, readErr := io.ReadFull(opened, header)
	if readErr != nil && readErr != io.ErrUnexpectedEOF {
		opened.Close()
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "图片文件内容无效"})
		return
	}
	contentType := http.DetectContentType(header[:read])
	if err := opened.Close(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "图片文件内容无效"})
		return
	}
	if !allowedImageMIMEs[contentType] || !extensionMatchesMIME(ext, contentType) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "图片格式与文件内容不一致"})
		return
	}
	name := fmt.Sprintf("%d-%d%s", time.Now().UnixNano(), currentUserID(c), ext)
	if err := c.SaveUploadedFile(file, filepath.Join(u.Dir, name)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "保存图片失败"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "success", "data": gin.H{"url": "/uploads/" + name}})
}

func extensionMatchesMIME(ext, contentType string) bool {
	switch contentType {
	case "image/jpeg":
		return ext == ".jpg" || ext == ".jpeg"
	case "image/png":
		return ext == ".png"
	case "image/gif":
		return ext == ".gif"
	case "image/webp":
		return ext == ".webp"
	default:
		return false
	}
}

// CleanupUploads 删除上传目录中不再被用户、文章正文、封面或版本历史引用的文件。
// 本地上传没有单独的资源表，因此以数据库中的引用为准进行安全回收。
func (u *UploadController) CleanupUploads(c *gin.Context) {
	if u.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "文件清理未配置"})
		return
	}
	removed, err := cleanupUnreferencedUploads(u.Dir, u.DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "清理图片失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"removed": removed}})
}

func cleanupUnreferencedUploads(dir string, db *gorm.DB) (int, error) {
	referenced, err := referencedUploadNames(db)
	if err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	removed := 0
	for _, entry := range entries {
		if entry.IsDir() || referenced[entry.Name()] {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

func referencedUploadNames(db *gorm.DB) (map[string]bool, error) {
	referenced := map[string]bool{}
	add := func(value string) {
		for _, match := range uploadReferencePattern.FindAllStringSubmatch(value, -1) {
			if len(match) > 1 {
				referenced[match[1]] = true
			}
		}
	}
	var users []models.User
	if err := db.Select("avatar").Find(&users).Error; err != nil {
		return nil, err
	}
	for _, user := range users {
		add(user.Avatar)
	}
	var posts []models.Post
	if err := db.Select("cover_image", "content").Find(&posts).Error; err != nil {
		return nil, err
	}
	for _, post := range posts {
		add(post.CoverImage)
		add(post.Content)
	}
	var revisions []models.PostRevision
	if err := db.Select("cover_image", "content").Find(&revisions).Error; err != nil {
		return nil, err
	}
	for _, revision := range revisions {
		add(revision.CoverImage)
		add(revision.Content)
	}
	return referenced, nil
}
