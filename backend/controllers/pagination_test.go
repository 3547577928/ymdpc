package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"quietsignal/backend/models"

	"github.com/gin-gonic/gin"
)

// 评论分页：total 只计顶层评论，回复随父评论一并返回
func TestListCommentsPaged(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newCommunityTestDB(t)
	user := models.User{Username: "alice", Nickname: "Alice", Role: "user", Status: "active"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	post := models.Post{AuthorID: user.ID, Title: "Hello", Slug: "hello", Content: "body", Status: "published", PublishedAt: time.Now()}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	rootIDs := make([]uint, 0, 3)
	for i := 1; i <= 3; i++ {
		comment := models.Comment{PostID: post.ID, AuthorID: user.ID, Content: fmt.Sprintf("root-%d", i), Status: "published"}
		if err := db.Create(&comment).Error; err != nil {
			t.Fatal(err)
		}
		rootIDs = append(rootIDs, comment.ID)
	}
	reply := models.Comment{PostID: post.ID, AuthorID: user.ID, ParentID: &rootIDs[0], Content: "reply-1", Status: "published"}
	if err := db.Create(&reply).Error; err != nil {
		t.Fatal(err)
	}
	community := &CommunityController{DB: db}
	slugParam := gin.Params{{Key: "slug", Value: "hello"}}

	type pagePayload struct {
		Data struct {
			Items []CommentDTO `json:"items"`
			Total int64        `json:"total"`
			Page  int          `json:"page"`
		} `json:"data"`
	}
	first := performRequest(community.ListComments, http.MethodGet, "/api/posts/hello/comments?page=1&pageSize=2", "", slugParam)
	if first.Code != http.StatusOK {
		t.Fatalf("page 1 returned %d: %s", first.Code, first.Body.String())
	}
	var firstPage pagePayload
	if err := json.Unmarshal(first.Body.Bytes(), &firstPage); err != nil {
		t.Fatal(err)
	}
	// 第一页：2 条顶层 + 第一条的 1 条回复
	if firstPage.Data.Total != 3 || len(firstPage.Data.Items) != 3 {
		t.Fatalf("expected total 3 with 3 items on page 1, got total=%d items=%d", firstPage.Data.Total, len(firstPage.Data.Items))
	}
	second := performRequest(community.ListComments, http.MethodGet, "/api/posts/hello/comments?page=2&pageSize=2", "", slugParam)
	var secondPage pagePayload
	if err := json.Unmarshal(second.Body.Bytes(), &secondPage); err != nil {
		t.Fatal(err)
	}
	if len(secondPage.Data.Items) != 1 || secondPage.Data.Items[0].Content != "root-3" {
		t.Fatalf("expected only root-3 on page 2, got %#v", secondPage.Data.Items)
	}

	// 不带 page 参数时保持完整列表行为
	legacy := performRequest(community.ListComments, http.MethodGet, "/api/posts/hello/comments", "", slugParam)
	var legacyPayload struct {
		Data []CommentDTO `json:"data"`
	}
	if err := json.Unmarshal(legacy.Body.Bytes(), &legacyPayload); err != nil {
		t.Fatalf("legacy list must stay a plain array: %v", err)
	}
	if len(legacyPayload.Data) != 4 {
		t.Fatalf("expected all 4 comments in legacy response, got %d", len(legacyPayload.Data))
	}
}

// 论坛回复分页：repliesTotal 只计顶层回复，子回复随父回复带出
func TestTopicDetailRepliesPaged(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newForumTestDB(t)
	user := models.User{Username: "alice", Nickname: "Alice", Role: "user", Status: "active"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	topic := models.ForumTopic{AuthorID: user.ID, Content: "topic", Kind: "discuss", Status: "published"}
	if err := db.Create(&topic).Error; err != nil {
		t.Fatal(err)
	}
	rootIDs := make([]uint, 0, 2)
	for i := 1; i <= 2; i++ {
		reply := models.ForumReply{TopicID: topic.ID, AuthorID: user.ID, Content: fmt.Sprintf("root-%d", i), Status: "published"}
		if err := db.Create(&reply).Error; err != nil {
			t.Fatal(err)
		}
		rootIDs = append(rootIDs, reply.ID)
	}
	child := models.ForumReply{TopicID: topic.ID, AuthorID: user.ID, ParentID: &rootIDs[0], Content: "child-1", Status: "published"}
	if err := db.Create(&child).Error; err != nil {
		t.Fatal(err)
	}
	forum := &ForumController{DB: db}
	idParam := gin.Params{{Key: "id", Value: fmt.Sprint(topic.ID)}}

	type detailPayload struct {
		Data struct {
			Replies      []ForumReplyDTO `json:"replies"`
			RepliesTotal int64           `json:"repliesTotal"`
		} `json:"data"`
	}
	paged := performRequest(forum.TopicDetail, http.MethodGet, "/api/forum/1?replyPage=1&replyPageSize=1", "", idParam)
	if paged.Code != http.StatusOK {
		t.Fatalf("paged detail returned %d: %s", paged.Code, paged.Body.String())
	}
	var pagedPayload detailPayload
	if err := json.Unmarshal(paged.Body.Bytes(), &pagedPayload); err != nil {
		t.Fatal(err)
	}
	// 第一页：1 条顶层 + 它的 1 条子回复，顶层总数为 2
	if pagedPayload.Data.RepliesTotal != 2 || len(pagedPayload.Data.Replies) != 2 {
		t.Fatalf("expected repliesTotal 2 with root+child on page 1, got total=%d replies=%d", pagedPayload.Data.RepliesTotal, len(pagedPayload.Data.Replies))
	}

	full := performRequest(forum.TopicDetail, http.MethodGet, "/api/forum/1", "", idParam)
	var fullPayload detailPayload
	if err := json.Unmarshal(full.Body.Bytes(), &fullPayload); err != nil {
		t.Fatal(err)
	}
	if len(fullPayload.Data.Replies) != 3 || fullPayload.Data.RepliesTotal != 2 {
		t.Fatalf("expected all 3 replies and repliesTotal 2, got replies=%d total=%d", len(fullPayload.Data.Replies), fullPayload.Data.RepliesTotal)
	}
}
