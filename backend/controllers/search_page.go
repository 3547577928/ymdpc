package controllers

import (
	"net/http"
	"strings"

	"quietsignal/backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 搜索落地页：FTS5 全文检索 + 命中高亮。
// 高亮标记用 \x01/\x02 控制符（highlight/snippet 的标记参数），前端按控制符
// 切分后渲染 <mark>，全程不经 innerHTML，无 XSS 面。
type SearchResultDTO struct {
	PostSummaryDTO
	TitleMarked string `json:"titleMarked"` // 标题（含高亮标记）
	Snippet     string `json:"snippet"`     // 正文摘要（含高亮标记）；LIKE 回退时为空
}

type searchHit struct {
	ID          uint   `gorm:"column:rowid"`
	TitleMarked string `gorm:"column:title_marked"`
	Snippet     string `gorm:"column:snippet"`
}

// Search 全局搜索。FTS 可用时按 bm25 相关度排序并返回高亮摘要；
// 不可用或关键字过短时回退 LIKE（无高亮，fts=false 告知前端）
func (p *PostController) Search(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("q"))
	page, pageSize := pagination(c)
	if keyword == "" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"items": make([]SearchResultDTO, 0), "total": 0, "page": page, "pageSize": pageSize, "fts": postSearchFTS.Load()}})
		return
	}
	match := ftsMatchExpression(keyword)
	if !postSearchFTS.Load() || match == "" {
		p.searchWithLike(c, keyword, page, pageSize)
		return
	}
	// 话题与用户量级小，LIKE 检索即可，随文章结果一并返回（多类型搜索）

	base := p.DB.Table("posts_fts").Joins("JOIN posts ON posts.id = posts_fts.rowid").
		Where("posts_fts MATCH ?", match).
		Where("posts.status = ? AND posts.moderation_status = ?", "published", "normal")
	var total int64
	if err := base.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "搜索失败"})
		return
	}
	var hits []searchHit
	if err := base.Select("posts_fts.rowid, highlight(posts_fts, 0, char(1), char(2)) AS title_marked, snippet(posts_fts, 2, char(1), char(2), '…', 24) AS snippet").
		Order("bm25(posts_fts)").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&hits).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "搜索失败"})
		return
	}
	// 按相关度顺序加载文章（IN 查询不保序，用 map 回填）
	ids := make([]uint, 0, len(hits))
	for _, hit := range hits {
		ids = append(ids, hit.ID)
	}
	var posts []models.Post
	if len(ids) > 0 {
		if err := p.DB.Where("id IN ?", ids).Preload("Tags").Preload("Author").Preload("Category").Find(&posts).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "搜索失败"})
			return
		}
	}
	byID := make(map[uint]models.Post, len(posts))
	for _, post := range posts {
		byID[post.ID] = post
	}
	liked, favorited := interactionSets(p.DB, currentUserID(c), ids)
	items := make([]SearchResultDTO, 0, len(hits))
	for _, hit := range hits {
		post, ok := byID[hit.ID]
		if !ok {
			continue
		}
		item := toPostSummaryDTO(post)
		item.Liked = liked[post.ID]
		item.Favorited = favorited[post.ID]
		items = append(items, SearchResultDTO{PostSummaryDTO: item, TitleMarked: hit.TitleMarked, Snippet: hit.Snippet})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"items": items, "total": total, "page": page, "pageSize": pageSize, "fts": true, "topics": searchForumTopics(p.DB, keyword, 5), "users": searchUsers(p.DB, keyword, 5)}})
}

// searchForumTopics 论坛话题 LIKE 检索（最多 limit 条，按时间倒序）
func searchForumTopics(db *gorm.DB, keyword string, limit int) []ForumTopicDTO {
	var topics []models.ForumTopic
	if err := db.Where("status = ? AND content LIKE ? ESCAPE '\\'", "published", likePattern(keyword)).Preload("Author").Preload("Images").Order("created_at DESC, id DESC").Limit(limit).Find(&topics).Error; err != nil {
		return []ForumTopicDTO{}
	}
	items := make([]ForumTopicDTO, 0, len(topics))
	for _, topic := range topics {
		items = append(items, forumTopicDTO(topic, false))
	}
	return items
}

// searchUsers 用户 LIKE 检索（用户名/昵称，仅正常状态账号）
func searchUsers(db *gorm.DB, keyword string, limit int) []UserDTO {
	like := likePattern(keyword)
	var users []models.User
	if err := db.Where("status = ? AND (username LIKE ? ESCAPE '\\' OR nickname LIKE ? ESCAPE '\\')", "active", like, like).Order("created_at DESC, id DESC").Limit(limit).Find(&users).Error; err != nil {
		return []UserDTO{}
	}
	items := make([]UserDTO, 0, len(users))
	for _, user := range users {
		items = append(items, toUserDTO(user))
	}
	return items
}

// searchWithLike LIKE 回退：无高亮，按发布时间排序
func (p *PostController) searchWithLike(c *gin.Context, keyword string, page, pageSize int) {
	like := likePattern(keyword)
	base := p.DB.Model(&models.Post{}).Where("status = ? AND moderation_status = ?", "published", "normal").
		Where("title LIKE ? ESCAPE '\\' OR summary LIKE ? ESCAPE '\\' OR content LIKE ? ESCAPE '\\'", like, like, like)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "搜索失败"})
		return
	}
	var posts []models.Post
	if err := base.Order("published_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).
		Preload("Tags").Preload("Author").Preload("Category").Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "搜索失败"})
		return
	}
	liked, favorited := interactionSets(p.DB, currentUserID(c), postIDs(posts))
	items := make([]SearchResultDTO, 0, len(posts))
	for _, post := range posts {
		item := toPostSummaryDTO(post)
		item.Liked = liked[post.ID]
		item.Favorited = favorited[post.ID]
		items = append(items, SearchResultDTO{PostSummaryDTO: item, TitleMarked: post.Title})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"items": items, "total": total, "page": page, "pageSize": pageSize, "fts": false, "topics": searchForumTopics(p.DB, keyword, 5), "users": searchUsers(p.DB, keyword, 5)}})
}
