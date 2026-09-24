package controllers

import (
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"quietsignal/backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ForumController struct{ DB *gorm.DB }

type ForumTopicDTO struct {
	ID           uint      `json:"id"`
	Content      string    `json:"content"`
	Kind         string    `json:"kind"`
	Images       []string  `json:"images"`
	LikesCount   int       `json:"likesCount"`
	RepliesCount int       `json:"repliesCount"`
	Liked        bool      `json:"liked"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	Author       UserDTO   `json:"author"`
}

type ForumReplyDTO struct {
	ID            uint      `json:"id"`
	TopicID       uint      `json:"topicId"`
	ParentID      *uint     `json:"parentId"`
	ReplyToUserID *uint     `json:"replyToUserId"`
	Content       string    `json:"content"`
	LikesCount    int       `json:"likesCount"`
	Liked         bool      `json:"liked"`
	CreatedAt     time.Time `json:"createdAt"`
	Author        UserDTO   `json:"author"`
}

func (fc *ForumController) ensureActiveUser(c *gin.Context, userID uint) bool {
	var user models.User
	if err := fc.DB.Select("status").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "unauthorized"})
		return false
	}
	if user.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "账号已被禁言或封禁，暂时无法发布内容"})
		return false
	}
	return true
}

func normalizeForumKind(kind string) (string, bool) {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		return "discuss", true
	}
	switch kind {
	case "discuss", "share", "help", "rant":
		return kind, true
	default:
		return "", false
	}
}

func normalizeForumImages(images []string) ([]string, bool) {
	if len(images) > 9 {
		return nil, false
	}
	result := make([]string, 0, len(images))
	seen := map[string]bool{}
	for _, image := range images {
		image = strings.TrimSpace(image)
		if image == "" || seen[image] {
			continue
		}
		if len(image) > 500 || !strings.HasPrefix(image, "/uploads/") {
			return nil, false
		}
		seen[image] = true
		result = append(result, image)
	}
	return result, true
}

func forumTopicDTO(topic models.ForumTopic, liked bool) ForumTopicDTO {
	images := make([]string, 0, len(topic.Images))
	for _, image := range topic.Images {
		images = append(images, image.URL)
	}
	return ForumTopicDTO{ID: topic.ID, Content: topic.Content, Kind: topic.Kind, Images: images, LikesCount: topic.LikesCount, RepliesCount: topic.RepliesCount, Liked: liked, CreatedAt: topic.CreatedAt, UpdatedAt: topic.UpdatedAt, Author: toUserDTO(topic.Author)}
}

func forumReplyDTO(reply models.ForumReply, liked bool) ForumReplyDTO {
	return ForumReplyDTO{ID: reply.ID, TopicID: reply.TopicID, ParentID: reply.ParentID, ReplyToUserID: reply.ReplyToUserID, Content: reply.Content, LikesCount: reply.LikesCount, Liked: liked, CreatedAt: reply.CreatedAt, Author: toUserDTO(reply.Author)}
}

func (fc *ForumController) ListTopics(c *gin.Context) {
	page, pageSize := pagination(c)
	query := fc.DB.Model(&models.ForumTopic{}).Where("status = ?", "published")
	if kind := strings.TrimSpace(c.Query("kind")); kind != "" && kind != "all" {
		if normalized, ok := normalizeForumKind(kind); ok {
			query = query.Where("kind = ?", normalized)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "帖子分类无效"})
			return
		}
	}
	if keyword := strings.TrimSpace(c.Query("q")); keyword != "" {
		query = query.Where("content LIKE ? ESCAPE '\\'", likePattern(keyword))
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取帖子失败"})
		return
	}
	order := "created_at DESC, id DESC"
	if c.Query("mode") == "hot" {
		order = "((likes_count * 3 + replies_count * 5 + 1.0) / max((julianday('now') - julianday(created_at)) * 24.0, 2.0)) DESC, created_at DESC, id DESC"
	}
	var topics []models.ForumTopic
	if err := query.Preload("Author").Preload("Images", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC, id ASC") }).Order(order).Offset((page - 1) * pageSize).Limit(pageSize).Find(&topics).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取帖子失败"})
		return
	}
	liked := forumTopicLikedSet(fc.DB, currentUserID(c), forumTopicIDs(topics))
	items := make([]ForumTopicDTO, 0, len(topics))
	for _, topic := range topics {
		items = append(items, forumTopicDTO(topic, liked[topic.ID]))
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"items": items, "total": total, "page": page, "pageSize": pageSize}})
}

func (fc *ForumController) CreateTopic(c *gin.Context) {
	userID := currentUserID(c)
	if !fc.ensureActiveUser(c, userID) {
		return
	}
	var input struct {
		Content string   `json:"content"`
		Kind    string   `json:"kind"`
		Images  []string `json:"images"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "帖子格式错误"})
		return
	}
	input.Content = strings.TrimSpace(input.Content)
	if utf8.RuneCountInString(input.Content) == 0 || utf8.RuneCountInString(input.Content) > 2000 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "帖子内容需为 1 到 2000 字"})
		return
	}
	kind, ok := normalizeForumKind(input.Kind)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "帖子分类无效"})
		return
	}
	images, ok := normalizeForumImages(input.Images)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "帖子图片无效或超过 9 张"})
		return
	}
	topic := models.ForumTopic{AuthorID: userID, Content: input.Content, Kind: kind, Status: "published"}
	if err := fc.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&topic).Error; err != nil {
			return err
		}
		for index, image := range images {
			if err := tx.Create(&models.ForumTopicImage{TopicID: topic.ID, URL: image, Position: index}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "发布帖子失败"})
		return
	}
	if err := fc.DB.Preload("Author").Preload("Images", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC, id ASC") }).First(&topic, topic.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取新帖子失败"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "success", "data": forumTopicDTO(topic, false)})
}

func (fc *ForumController) TopicDetail(c *gin.Context) {
	topicID, ok := forumParamID(c)
	if !ok {
		return
	}
	var topic models.ForumTopic
	if err := fc.DB.Where("id = ? AND status = ?", topicID, "published").Preload("Author").Preload("Images", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC, id ASC") }).First(&topic).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "帖子不存在"})
		return
	}
	// 带 replyPage 参数时按顶层回复分页（子回复随父回复一并带出），repliesTotal
	// 只计顶层回复；不带参数时保持完整列表行为
	replyPage, _ := strconv.Atoi(c.DefaultQuery("replyPage", "0"))
	replyPageSize, _ := strconv.Atoi(c.DefaultQuery("replyPageSize", "20"))
	if replyPageSize < 1 {
		replyPageSize = 20
	}
	if replyPageSize > 50 {
		replyPageSize = 50
	}
	var repliesTotal int64
	if err := fc.DB.Model(&models.ForumReply{}).Where("topic_id = ? AND status = ? AND parent_id IS NULL", topic.ID, "published").Count(&repliesTotal).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取回复失败"})
		return
	}
	var replies []models.ForumReply
	if replyPage > 0 {
		var roots []models.ForumReply
		if err := fc.DB.Where("topic_id = ? AND status = ? AND parent_id IS NULL", topic.ID, "published").Preload("Author").Order("created_at ASC, id ASC").Offset((replyPage - 1) * replyPageSize).Limit(replyPageSize).Find(&roots).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取回复失败"})
			return
		}
		rootIDs := make([]uint, 0, len(roots))
		for _, root := range roots {
			rootIDs = append(rootIDs, root.ID)
		}
		var children []models.ForumReply
		if len(rootIDs) > 0 {
			if err := fc.DB.Where("parent_id IN ? AND status = ?", rootIDs, "published").Preload("Author").Order("created_at ASC, id ASC").Find(&children).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取回复失败"})
				return
			}
		}
		replies = append(roots, children...)
	} else if err := fc.DB.Where("topic_id = ? AND status = ?", topic.ID, "published").Preload("Author").Order("created_at ASC, id ASC").Find(&replies).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取回复失败"})
		return
	}
	userID := currentUserID(c)
	likedTopics := forumTopicLikedSet(fc.DB, userID, []uint{topic.ID})
	likedReplies := forumReplyLikedSet(fc.DB, userID, forumReplyIDs(replies))
	replyItems := make([]ForumReplyDTO, 0, len(replies))
	for _, reply := range replies {
		replyItems = append(replyItems, forumReplyDTO(reply, likedReplies[reply.ID]))
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"topic": forumTopicDTO(topic, likedTopics[topic.ID]), "replies": replyItems, "repliesTotal": repliesTotal}})
}

func (fc *ForumController) DeleteTopic(c *gin.Context) {
	topicID, ok := forumParamID(c)
	if !ok {
		return
	}
	var topic models.ForumTopic
	if err := fc.DB.First(&topic, topicID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "帖子不存在"})
		return
	}
	if topic.AuthorID != currentUserID(c) && !currentUserIsAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只能删除自己的帖子"})
		return
	}
	if err := fc.DB.Transaction(func(tx *gorm.DB) error {
		var replyIDs []uint
		if err := tx.Model(&models.ForumReply{}).Where("topic_id = ?", topic.ID).Pluck("id", &replyIDs).Error; err != nil {
			return err
		}
		if len(replyIDs) > 0 {
			if err := tx.Where("reply_id IN ?", replyIDs).Delete(&models.ForumReplyLike{}).Error; err != nil {
				return err
			}
			if err := tx.Where("target_type = ? AND target_id IN ?", "forum_reply", replyIDs).Delete(&models.Report{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("topic_id = ?", topic.ID).Delete(&models.ForumReply{}).Error; err != nil {
			return err
		}
		if err := tx.Where("topic_id = ?", topic.ID).Delete(&models.ForumTopicLike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("topic_id = ?", topic.ID).Delete(&models.ForumTopicImage{}).Error; err != nil {
			return err
		}
		if err := tx.Where("type IN ? AND resource_id = ?", []string{"forum_reply", "forum_like"}, topic.ID).Delete(&models.Notification{}).Error; err != nil {
			return err
		}
		if err := tx.Where("target_type IN ? AND target_id = ?", []string{"forum_topic"}, topic.ID).Delete(&models.Report{}).Error; err != nil {
			return err
		}
		return tx.Delete(&topic).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除帖子失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

func (fc *ForumController) LikeTopic(c *gin.Context) {
	userID := currentUserID(c)
	topicID, ok := forumParamID(c)
	if !ok {
		return
	}
	var topic models.ForumTopic
	if err := fc.DB.Where("id = ? AND status = ?", topicID, "published").First(&topic).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "帖子不存在"})
		return
	}
	likesCount := topic.LikesCount
	if err := fc.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("user_id = ? AND topic_id = ?", userID, topic.ID).FirstOrCreate(&models.ForumTopicLike{UserID: userID, TopicID: topic.ID})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			if err := tx.Model(&models.ForumTopic{}).Where("id = ?", topic.ID).UpdateColumn("likes_count", gorm.Expr("likes_count + 1")).Error; err != nil {
				return err
			}
			if topic.AuthorID != userID {
				if err := createNotification(tx, topic.AuthorID, userID, "forum_like", topic.ID); err != nil {
					return err
				}
			}
		}
		return tx.Model(&models.ForumTopic{}).Select("likes_count").Where("id = ?", topic.ID).Scan(&likesCount).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "点赞失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"liked": true, "likesCount": likesCount}})
}

func (fc *ForumController) UnlikeTopic(c *gin.Context) {
	userID := currentUserID(c)
	topicID, ok := forumParamID(c)
	if !ok {
		return
	}
	var topic models.ForumTopic
	if err := fc.DB.First(&topic, topicID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "帖子不存在"})
		return
	}
	likesCount := topic.LikesCount
	if err := fc.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("user_id = ? AND topic_id = ?", userID, topic.ID).Delete(&models.ForumTopicLike{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			if err := tx.Model(&models.ForumTopic{}).Where("id = ? AND likes_count > 0", topic.ID).UpdateColumn("likes_count", gorm.Expr("likes_count - 1")).Error; err != nil {
				return err
			}
		}
		return tx.Model(&models.ForumTopic{}).Select("likes_count").Where("id = ?", topic.ID).Scan(&likesCount).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "取消点赞失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"liked": false, "likesCount": likesCount}})
}

func (fc *ForumController) CreateReply(c *gin.Context) {
	userID := currentUserID(c)
	if !fc.ensureActiveUser(c, userID) {
		return
	}
	if !models.BoolSetting(fc.DB, "comments_enabled", true) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "回复功能已关闭"})
		return
	}
	topicID, ok := forumParamID(c)
	if !ok {
		return
	}
	var topic models.ForumTopic
	if err := fc.DB.Where("id = ? AND status = ?", topicID, "published").First(&topic).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "帖子不存在"})
		return
	}
	var input struct {
		Content       string `json:"content"`
		ParentID      *uint  `json:"parentId"`
		ReplyToUserID *uint  `json:"replyToUserId"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "回复格式错误"})
		return
	}
	input.Content = strings.TrimSpace(input.Content)
	if utf8.RuneCountInString(input.Content) == 0 || utf8.RuneCountInString(input.Content) > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "回复内容需为 1 到 1000 字"})
		return
	}
	if input.ParentID != nil {
		var parent models.ForumReply
		if err := fc.DB.Where("id = ? AND topic_id = ? AND status = ?", *input.ParentID, topic.ID, "published").First(&parent).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "回复目标不存在"})
			return
		}
		if parent.ParentID != nil {
			rootID := *parent.ParentID
			input.ParentID = &rootID
		}
	}
	reply := models.ForumReply{TopicID: topic.ID, AuthorID: userID, ParentID: input.ParentID, ReplyToUserID: input.ReplyToUserID, Content: input.Content, Status: "published"}
	if err := fc.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&reply).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.ForumTopic{}).Where("id = ?", topic.ID).UpdateColumn("replies_count", gorm.Expr("replies_count + 1")).Error; err != nil {
			return err
		}
		if topic.AuthorID != userID {
			if err := createNotification(tx, topic.AuthorID, userID, "forum_reply", topic.ID); err != nil {
				return err
			}
		}
		if input.ReplyToUserID != nil && *input.ReplyToUserID != userID && *input.ReplyToUserID != topic.AuthorID {
			if err := createNotification(tx, *input.ReplyToUserID, userID, "forum_reply", topic.ID); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "发表回复失败"})
		return
	}
	if err := fc.DB.Preload("Author").First(&reply, reply.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取新回复失败"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "success", "data": forumReplyDTO(reply, false)})
}

func (fc *ForumController) DeleteReply(c *gin.Context) {
	replyID, ok := forumParamID(c)
	if !ok {
		return
	}
	var reply models.ForumReply
	if err := fc.DB.First(&reply, replyID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "回复不存在"})
		return
	}
	if reply.AuthorID != currentUserID(c) && !currentUserIsAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只能删除自己的回复"})
		return
	}
	if err := fc.DB.Transaction(func(tx *gorm.DB) error {
		ids, err := forumReplyDescendantIDs(tx, reply.ID)
		if err != nil {
			return err
		}
		if err := tx.Where("reply_id IN ?", ids).Delete(&models.ForumReplyLike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", ids).Delete(&models.ForumReply{}).Error; err != nil {
			return err
		}
		return refreshForumReplyCount(tx, reply.TopicID)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除回复失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

func (fc *ForumController) LikeReply(c *gin.Context) {
	fc.changeReplyLike(c, true)
}

func (fc *ForumController) UnlikeReply(c *gin.Context) {
	fc.changeReplyLike(c, false)
}

func (fc *ForumController) changeReplyLike(c *gin.Context, liked bool) {
	userID := currentUserID(c)
	replyID, ok := forumParamID(c)
	if !ok {
		return
	}
	var reply models.ForumReply
	if err := fc.DB.Where("id = ? AND status = ?", replyID, "published").First(&reply).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "回复不存在"})
		return
	}
	likesCount := reply.LikesCount
	if err := fc.DB.Transaction(func(tx *gorm.DB) error {
		if liked {
			result := tx.Where("user_id = ? AND reply_id = ?", userID, reply.ID).FirstOrCreate(&models.ForumReplyLike{UserID: userID, ReplyID: reply.ID})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected > 0 {
				if err := tx.Model(&models.ForumReply{}).Where("id = ?", reply.ID).UpdateColumn("likes_count", gorm.Expr("likes_count + 1")).Error; err != nil {
					return err
				}
				if reply.AuthorID != userID {
					if err := createNotification(tx, reply.AuthorID, userID, "forum_like", reply.TopicID); err != nil {
						return err
					}
				}
			}
		} else {
			result := tx.Where("user_id = ? AND reply_id = ?", userID, reply.ID).Delete(&models.ForumReplyLike{})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected > 0 {
				if err := tx.Model(&models.ForumReply{}).Where("id = ? AND likes_count > 0", reply.ID).UpdateColumn("likes_count", gorm.Expr("likes_count - 1")).Error; err != nil {
					return err
				}
			}
		}
		return tx.Model(&models.ForumReply{}).Select("likes_count").Where("id = ?", reply.ID).Scan(&likesCount).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新点赞失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"liked": liked, "likesCount": likesCount}})
}

func forumParamID(c *gin.Context) (uint, bool) {
	value, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || value == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "资源编号无效"})
		return 0, false
	}
	return uint(value), true
}

func forumTopicIDs(topics []models.ForumTopic) []uint {
	ids := make([]uint, 0, len(topics))
	for _, topic := range topics {
		ids = append(ids, topic.ID)
	}
	return ids
}

func forumReplyIDs(replies []models.ForumReply) []uint {
	ids := make([]uint, 0, len(replies))
	for _, reply := range replies {
		ids = append(ids, reply.ID)
	}
	return ids
}

func forumTopicLikedSet(db *gorm.DB, userID uint, ids []uint) map[uint]bool {
	result := map[uint]bool{}
	if userID == 0 || len(ids) == 0 {
		return result
	}
	var likedIDs []uint
	if err := db.Model(&models.ForumTopicLike{}).Where("user_id = ? AND topic_id IN ?", userID, ids).Pluck("topic_id", &likedIDs).Error; err != nil {
		return result
	}
	for _, id := range likedIDs {
		result[id] = true
	}
	return result
}

func forumReplyLikedSet(db *gorm.DB, userID uint, ids []uint) map[uint]bool {
	result := map[uint]bool{}
	if userID == 0 || len(ids) == 0 {
		return result
	}
	var likedIDs []uint
	if err := db.Model(&models.ForumReplyLike{}).Where("user_id = ? AND reply_id IN ?", userID, ids).Pluck("reply_id", &likedIDs).Error; err != nil {
		return result
	}
	for _, id := range likedIDs {
		result[id] = true
	}
	return result
}

func forumReplyDescendantIDs(tx *gorm.DB, rootID uint) ([]uint, error) {
	ids := []uint{rootID}
	frontier := []uint{rootID}
	for len(frontier) > 0 {
		var children []uint
		if err := tx.Model(&models.ForumReply{}).Where("parent_id IN ?", frontier).Pluck("id", &children).Error; err != nil {
			return nil, err
		}
		if len(children) == 0 {
			break
		}
		ids = append(ids, children...)
		frontier = children
	}
	return ids, nil
}

func refreshForumReplyCount(tx *gorm.DB, topicID uint) error {
	var count int64
	if err := tx.Model(&models.ForumReply{}).Where("topic_id = ? AND status = ?", topicID, "published").Count(&count).Error; err != nil {
		return err
	}
	return tx.Model(&models.ForumTopic{}).Where("id = ?", topicID).Update("replies_count", count).Error
}

func textExcerpt(value string, limit int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "…"
}
