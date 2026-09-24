package controllers

import (
	"encoding/xml"
	"net/http"
	"time"

	"quietsignal/backend/models"

	"github.com/gin-gonic/gin"
)

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Language    string    `xml:"language"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	GUID        string `xml:"guid"`
	PubDate     string `xml:"pubDate"`
	Creator     string `xml:"dc:creator"`
	Description string `xml:"description"`
}

// RSSFeed 输出最新 20 篇公开文章的 RSS 2.0。站点地址从请求推断：
// 反向代理后的公网地址依赖 nginx 透传的 Host 与 X-Forwarded-Proto
func (p *PostController) RSSFeed(c *gin.Context) {
	var posts []models.Post
	if err := p.DB.Where("status = ? AND moderation_status = ?", "published", "normal").Preload("Author").Order("published_at DESC, id DESC").Limit(20).Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "生成订阅源失败"})
		return
	}
	scheme := c.Request.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	base := scheme + "://" + c.Request.Host
	feed := rssFeed{
		Version: "2.0",
		Channel: rssChannel{
			Title:       "Quiet Signal",
			Link:        base,
			Description: "一个安静的写作社区：写下界面的细节、系统的取舍，以及还没有答案的问题。",
			Language:    "zh-CN",
			Items:       make([]rssItem, 0, len(posts)),
		},
	}
	for _, post := range posts {
		link := base + "/posts/" + post.Slug
		pubDate := post.PublishedAt
		if pubDate.IsZero() {
			pubDate = post.CreatedAt
		}
		feed.Channel.Items = append(feed.Channel.Items, rssItem{
			Title:       post.Title,
			Link:        link,
			GUID:        link,
			PubDate:     pubDate.UTC().Format(time.RFC1123),
			Creator:     post.Author.Nickname,
			Description: post.Summary,
		})
	}
	c.Header("Content-Type", "application/rss+xml; charset=utf-8")
	c.String(http.StatusOK, xml.Header)
	if err := xml.NewEncoder(c.Writer).Encode(feed); err != nil {
		_ = c.Error(err)
	}
}
