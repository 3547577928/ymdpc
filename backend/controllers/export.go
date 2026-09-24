package controllers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"quietsignal/backend/models"

	"github.com/gin-gonic/gin"
)

// AdminExport 将管理端常用数据导出为 UTF-8 CSV，避免把大批数据塞入 JSON 页面。
func (ic *InteractionController) AdminExport(c *gin.Context) {
	kind := c.DefaultQuery("type", "posts")
	if kind != "posts" && kind != "users" && kind != "reports" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "导出类型无效"})
		return
	}
	filename := fmt.Sprintf("quietsig-%s-%s.csv", kind, time.Now().Format("20060102-150405"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)

	writer := csv.NewWriter(c.Writer)
	// Excel 和部分国产表格软件需要 BOM 才能正确识别中文列名。
	_, _ = c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})
	var err error
	switch kind {
	case "posts":
		err = exportPosts(ic, writer)
	case "users":
		err = exportUsers(ic, writer)
	case "reports":
		err = exportReports(ic, writer)
	}
	if err != nil {
		// 导出响应已经开始写出，无法再改成 JSON；记录为服务端错误并结束 CSV。
		c.Error(err)
		return
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		c.Error(err)
	}
}

func exportPosts(ic *InteractionController, writer *csv.Writer) error {
	if err := writer.Write([]string{"id", "title", "slug", "author", "status", "moderationStatus", "views", "likes", "comments", "createdAt", "publishedAt"}); err != nil {
		return err
	}
	var posts []models.Post
	if err := ic.DB.Preload("Author").Order("id ASC").Find(&posts).Error; err != nil {
		return err
	}
	for _, post := range posts {
		if err := writer.Write([]string{strconv.FormatUint(uint64(post.ID), 10), post.Title, post.Slug, post.Author.Username, post.Status, post.ModerationStatus, strconv.Itoa(post.Views), strconv.Itoa(post.LikesCount), strconv.Itoa(post.CommentsCount), post.CreatedAt.Format(time.RFC3339), post.PublishedAt.Format(time.RFC3339)}); err != nil {
			return err
		}
	}
	return nil
}

func exportUsers(ic *InteractionController, writer *csv.Writer) error {
	if err := writer.Write([]string{"id", "username", "nickname", "email", "role", "status", "createdAt"}); err != nil {
		return err
	}
	var users []models.User
	if err := ic.DB.Order("id ASC").Find(&users).Error; err != nil {
		return err
	}
	for _, user := range users {
		if err := writer.Write([]string{strconv.FormatUint(uint64(user.ID), 10), user.Username, user.Nickname, user.Email, user.Role, user.Status, user.CreatedAt.Format(time.RFC3339)}); err != nil {
			return err
		}
	}
	return nil
}

func exportReports(ic *InteractionController, writer *csv.Writer) error {
	if err := writer.Write([]string{"id", "targetType", "targetId", "reason", "status", "resolution", "reporter", "handler", "createdAt", "handledAt"}); err != nil {
		return err
	}
	var reports []models.Report
	if err := ic.DB.Preload("Reporter").Preload("Handler").Order("id ASC").Find(&reports).Error; err != nil {
		return err
	}
	for _, report := range reports {
		handler := ""
		if report.Handler != nil {
			handler = report.Handler.Username
		}
		handledAt := ""
		if report.HandledAt != nil {
			handledAt = report.HandledAt.Format(time.RFC3339)
		}
		if err := writer.Write([]string{strconv.FormatUint(uint64(report.ID), 10), report.TargetType, strconv.FormatUint(uint64(report.TargetID), 10), report.Reason, report.Status, report.Resolution, report.Reporter.Username, handler, report.CreatedAt.Format(time.RFC3339), handledAt}); err != nil {
			return err
		}
	}
	return nil
}
