package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"quietsignal/backend/models"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newCommunityTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "-") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Post{}, &models.Tag{}, &models.Comment{}, &models.PostLike{}, &models.Follow{}, &models.Category{}, &models.Favorite{}, &models.CommentLike{}, &models.Notification{}, &models.Report{}, &models.AdminLog{}, &models.Setting{}, &models.PostRevision{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func authenticatedRequest(handler gin.HandlerFunc, method, target, body string, params gin.Params, userID uint, role string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(method, target, strings.NewReader(body))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = params
	context.Set("userID", userID)
	context.Set("userRole", role)
	handler(context)
	return recorder
}

func TestCommunityInteractions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newCommunityTestDB(t)
	users := []models.User{{Username: "admin", Nickname: "Admin", Role: "admin", Status: "active"}, {Username: "alice", Nickname: "Alice", Role: "user", Status: "active"}, {Username: "bob", Nickname: "Bob", Role: "user", Status: "active"}}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	post := models.Post{AuthorID: users[1].ID, Title: "Hello", Slug: "hello", Content: "body", Status: "published", PublishedAt: time.Now()}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	community := &CommunityController{DB: db}

	createResponse := authenticatedRequest(community.CreatePost, http.MethodPost, "/api/posts", `{"title":"Second","content":"body","status":"published"}`, nil, users[2].ID, "user")
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("create returned %d: %s", createResponse.Code, createResponse.Body.String())
	}
	var createPayload struct {
		Data PostDTO `json:"data"`
	}
	if err := json.Unmarshal(createResponse.Body.Bytes(), &createPayload); err != nil {
		t.Fatal(err)
	}
	if createPayload.Data.Author.Username != "bob" {
		t.Fatalf("expected created post author bob, got %#v", createPayload.Data.Author)
	}

	likeResponse := authenticatedRequest(community.LikePost, http.MethodPost, "/api/posts/hello/like", "", gin.Params{{Key: "slug", Value: "hello"}}, users[2].ID, "user")
	if likeResponse.Code != http.StatusOK {
		t.Fatalf("like returned %d: %s", likeResponse.Code, likeResponse.Body.String())
	}
	likeAgain := authenticatedRequest(community.LikePost, http.MethodPost, "/api/posts/hello/like", "", gin.Params{{Key: "slug", Value: "hello"}}, users[2].ID, "user")
	if likeAgain.Code != http.StatusOK {
		t.Fatalf("duplicate like returned %d: %s", likeAgain.Code, likeAgain.Body.String())
	}
	var storedPost models.Post
	if err := db.First(&storedPost, post.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedPost.LikesCount != 1 {
		t.Fatalf("expected one like, got %d", storedPost.LikesCount)
	}

	commentResponse := authenticatedRequest(community.CreateComment, http.MethodPost, "/api/posts/hello/comments", `{"content":"First comment"}`, gin.Params{{Key: "slug", Value: "hello"}}, users[2].ID, "user")
	if commentResponse.Code != http.StatusCreated {
		t.Fatalf("comment returned %d: %s", commentResponse.Code, commentResponse.Body.String())
	}
	commentsResponse := performRequest(community.ListComments, http.MethodGet, "/api/posts/hello/comments", "", gin.Params{{Key: "slug", Value: "hello"}})
	if commentsResponse.Code != http.StatusOK || !strings.Contains(commentsResponse.Body.String(), "First comment") {
		t.Fatalf("unexpected comments response: %d %s", commentsResponse.Code, commentsResponse.Body.String())
	}

	followResponse := authenticatedRequest(community.FollowUser, http.MethodPost, "/api/users/2/follow", "", gin.Params{{Key: "id", Value: "2"}}, users[2].ID, "user")
	if followResponse.Code != http.StatusOK {
		t.Fatalf("follow returned %d: %s", followResponse.Code, followResponse.Body.String())
	}
	var followCount int64
	db.Model(&models.Follow{}).Where("following_id = ?", users[1].ID).Count(&followCount)
	if followCount != 1 {
		t.Fatalf("expected one follower, got %d", followCount)
	}
}

func TestFollowingFeedAndPostNotifications(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newCommunityTestDB(t)
	users := []models.User{
		{Username: "author", Nickname: "Author", Role: "user", Status: "active"},
		{Username: "follower", Nickname: "Follower", Role: "user", Status: "active"},
		{Username: "other", Nickname: "Other", Role: "user", Status: "active"},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Follow{FollowerID: users[1].ID, FollowingID: users[0].ID}).Error; err != nil {
		t.Fatal(err)
	}
	older := models.Post{AuthorID: users[0].ID, Title: "Older", Slug: "older", Content: "body", Status: "published", PublishedAt: time.Now().Add(-time.Hour)}
	other := models.Post{AuthorID: users[2].ID, Title: "Other", Slug: "other", Content: "body", Status: "published", PublishedAt: time.Now()}
	if err := db.Create(&[]models.Post{older, other}).Error; err != nil {
		t.Fatal(err)
	}
	community := &CommunityController{DB: db}

	draftResponse := authenticatedRequest(community.CreatePost, http.MethodPost, "/api/posts", `{"title":"Newest","content":"body","status":"draft"}`, nil, users[0].ID, "user")
	if draftResponse.Code != http.StatusCreated {
		t.Fatalf("create draft returned %d: %s", draftResponse.Code, draftResponse.Body.String())
	}
	var draft models.Post
	if err := db.Where("slug = ?", "newest").First(&draft).Error; err != nil {
		t.Fatal(err)
	}
	var notificationCount int64
	db.Model(&models.Notification{}).Where("user_id = ? AND type = ?", users[1].ID, "post").Count(&notificationCount)
	if notificationCount != 0 {
		t.Fatalf("draft should not notify followers, got %d notifications", notificationCount)
	}

	params := gin.Params{{Key: "id", Value: fmt.Sprint(draft.ID)}}
	publishResponse := authenticatedRequest(community.UpdatePost, http.MethodPut, "/api/posts/id/3", `{"title":"Newest","content":"published body","status":"published"}`, params, users[0].ID, "user")
	if publishResponse.Code != http.StatusOK {
		t.Fatalf("publish returned %d: %s", publishResponse.Code, publishResponse.Body.String())
	}
	db.Model(&models.Notification{}).Where("user_id = ? AND actor_id = ? AND type = ? AND resource_id = ?", users[1].ID, users[0].ID, "post", draft.ID).Count(&notificationCount)
	if notificationCount != 1 {
		t.Fatalf("expected one post notification, got %d", notificationCount)
	}

	editResponse := authenticatedRequest(community.UpdatePost, http.MethodPut, "/api/posts/id/3", `{"title":"Newest edited","content":"published body","status":"published"}`, params, users[0].ID, "user")
	if editResponse.Code != http.StatusOK {
		t.Fatalf("edit returned %d: %s", editResponse.Code, editResponse.Body.String())
	}
	db.Model(&models.Notification{}).Where("user_id = ? AND type = ? AND resource_id = ?", users[1].ID, "post", draft.ID).Count(&notificationCount)
	if notificationCount != 1 {
		t.Fatalf("editing a published post should not notify again, got %d", notificationCount)
	}

	feedResponse := authenticatedRequest(community.Feed, http.MethodGet, "/api/feed?mode=following&pageSize=10", "", nil, users[1].ID, "user")
	if feedResponse.Code != http.StatusOK {
		t.Fatalf("following feed returned %d: %s", feedResponse.Code, feedResponse.Body.String())
	}
	var feedPayload struct {
		Data struct {
			Items []PostSummaryDTO `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(feedResponse.Body.Bytes(), &feedPayload); err != nil {
		t.Fatal(err)
	}
	if len(feedPayload.Data.Items) != 2 || feedPayload.Data.Items[0].Slug != "newest" || feedPayload.Data.Items[1].Slug != "older" {
		t.Fatalf("unexpected following feed: %#v", feedPayload.Data.Items)
	}

	notifications := &InteractionController{DB: db}
	notificationResponse := authenticatedRequest(notifications.Notifications, http.MethodGet, "/api/notifications", "", nil, users[1].ID, "user")
	if notificationResponse.Code != http.StatusOK || !strings.Contains(notificationResponse.Body.String(), `"resourceSlug":"newest"`) {
		t.Fatalf("post notification missing resource data: %d %s", notificationResponse.Code, notificationResponse.Body.String())
	}
}

// TestMutedUserCannotPostOrComment 验证禁言用户不能发布文章和评论
func TestMutedUserCannotPostOrComment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newCommunityTestDB(t)
	users := []models.User{{Username: "author", Nickname: "Author", Role: "user", Status: "active"}, {Username: "muted", Nickname: "Muted", Role: "user", Status: "muted"}}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	post := models.Post{AuthorID: users[0].ID, Title: "Hello", Slug: "hello", Content: "body", Status: "published", PublishedAt: time.Now()}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	community := &CommunityController{DB: db}

	postResponse := authenticatedRequest(community.CreatePost, http.MethodPost, "/api/posts", `{"title":"Nope","content":"body"}`, nil, users[1].ID, "user")
	if postResponse.Code != http.StatusForbidden {
		t.Fatalf("muted create post returned %d: %s", postResponse.Code, postResponse.Body.String())
	}
	commentResponse := authenticatedRequest(community.CreateComment, http.MethodPost, "/api/posts/hello/comments", `{"content":"nope"}`, gin.Params{{Key: "slug", Value: "hello"}}, users[1].ID, "user")
	if commentResponse.Code != http.StatusForbidden {
		t.Fatalf("muted create comment returned %d: %s", commentResponse.Code, commentResponse.Body.String())
	}
	var postCount, commentCount int64
	db.Model(&models.Post{}).Where("author_id = ?", users[1].ID).Count(&postCount)
	db.Model(&models.Comment{}).Where("author_id = ?", users[1].ID).Count(&commentCount)
	if postCount != 0 || commentCount != 0 {
		t.Fatalf("muted user data leaked: posts=%d comments=%d", postCount, commentCount)
	}
}

// TestTakedownHidesPostFromPublic 验证管理员下架后公开接口不可见，恢复后重新可见
func TestTakedownHidesPostFromPublic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newCommunityTestDB(t)
	user := models.User{Username: "author", Nickname: "Author", Role: "user", Status: "active"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	post := models.Post{AuthorID: user.ID, Title: "Hidden", Slug: "hidden", Content: "body", Status: "published", PublishedAt: time.Now()}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	posts := &PostController{DB: db}
	params := gin.Params{{Key: "id", Value: fmt.Sprint(post.ID)}}

	takedown := authenticatedRequest(posts.UpdateModeration, http.MethodPatch, "/api/admin/posts/1/moderation", `{"status":"hidden"}`, params, user.ID, "admin")
	if takedown.Code != http.StatusOK {
		t.Fatalf("takedown returned %d: %s", takedown.Code, takedown.Body.String())
	}
	detail := performRequest(posts.Detail, http.MethodGet, "/api/posts/hidden", "", gin.Params{{Key: "slug", Value: "hidden"}})
	if detail.Code != http.StatusNotFound {
		t.Fatalf("hidden post detail returned %d", detail.Code)
	}
	adminList := performRequest(posts.AdminList, http.MethodGet, "/api/admin/posts", "", nil)
	if !strings.Contains(adminList.Body.String(), "Hidden") {
		t.Fatalf("admin list should include hidden post: %s", adminList.Body.String())
	}
	restore := authenticatedRequest(posts.UpdateModeration, http.MethodPatch, "/api/admin/posts/1/moderation", `{"status":"normal"}`, params, user.ID, "admin")
	if restore.Code != http.StatusOK {
		t.Fatalf("restore returned %d: %s", restore.Code, restore.Body.String())
	}
	restoredDetail := performRequest(posts.Detail, http.MethodGet, "/api/posts/hidden", "", gin.Params{{Key: "slug", Value: "hidden"}})
	if restoredDetail.Code != http.StatusOK {
		t.Fatalf("restored post detail returned %d", restoredDetail.Code)
	}
}

// TestReplyToReplyStaysTwoLevels 验证对回复的回复会归一化挂到顶层评论下，保持两层结构
func TestReplyToReplyStaysTwoLevels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newCommunityTestDB(t)
	users := []models.User{{Username: "author", Nickname: "Author", Role: "user", Status: "active"}, {Username: "alice", Nickname: "Alice", Role: "user", Status: "active"}}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	post := models.Post{AuthorID: users[0].ID, Title: "Post", Slug: "post", Content: "body", Status: "published", PublishedAt: time.Now()}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	community := &CommunityController{DB: db}
	commentParams := gin.Params{{Key: "slug", Value: "post"}}

	rootResponse := authenticatedRequest(community.CreateComment, http.MethodPost, "/api/posts/post/comments", `{"content":"root"}`, commentParams, users[0].ID, "user")
	if rootResponse.Code != http.StatusCreated {
		t.Fatalf("root comment returned %d: %s", rootResponse.Code, rootResponse.Body.String())
	}
	var rootPayload struct {
		Data CommentDTO `json:"data"`
	}
	if err := json.Unmarshal(rootResponse.Body.Bytes(), &rootPayload); err != nil {
		t.Fatal(err)
	}

	replyResponse := authenticatedRequest(community.CreateComment, http.MethodPost, "/api/posts/post/comments", fmt.Sprintf(`{"content":"reply","parentId":%d}`, rootPayload.Data.ID), commentParams, users[1].ID, "user")
	if replyResponse.Code != http.StatusCreated {
		t.Fatalf("reply returned %d: %s", replyResponse.Code, replyResponse.Body.String())
	}
	var replyPayload struct {
		Data CommentDTO `json:"data"`
	}
	if err := json.Unmarshal(replyResponse.Body.Bytes(), &replyPayload); err != nil {
		t.Fatal(err)
	}

	// 对回复再回复，应挂到根评论而不是形成第三层
	nestedResponse := authenticatedRequest(community.CreateComment, http.MethodPost, "/api/posts/post/comments", fmt.Sprintf(`{"content":"nested","parentId":%d}`, replyPayload.Data.ID), commentParams, users[0].ID, "user")
	if nestedResponse.Code != http.StatusCreated {
		t.Fatalf("nested reply returned %d: %s", nestedResponse.Code, nestedResponse.Body.String())
	}
	var nested models.Comment
	if err := db.Where("content = ?", "nested").First(&nested).Error; err != nil {
		t.Fatal(err)
	}
	if nested.ParentID == nil || *nested.ParentID != rootPayload.Data.ID {
		t.Fatalf("nested reply should attach to root comment %d, got %v", rootPayload.Data.ID, nested.ParentID)
	}
}

// TestAdminCommentModeration 验证管理端评论列表与隐藏/恢复，评论计数同步
func TestAdminCommentModeration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newCommunityTestDB(t)
	users := []models.User{{Username: "author", Nickname: "Author", Role: "user", Status: "active"}, {Username: "alice", Nickname: "Alice", Role: "user", Status: "active"}}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	post := models.Post{AuthorID: users[0].ID, Title: "Post", Slug: "post", Content: "body", Status: "published", CommentsCount: 1, PublishedAt: time.Now()}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	comment := models.Comment{PostID: post.ID, AuthorID: users[1].ID, Content: "spam message", Status: "published"}
	if err := db.Create(&comment).Error; err != nil {
		t.Fatal(err)
	}
	community := &CommunityController{DB: db}

	adminList := performRequest(community.AdminComments, http.MethodGet, "/api/admin/comments", "", nil)
	if adminList.Code != http.StatusOK || !strings.Contains(adminList.Body.String(), "spam message") {
		t.Fatalf("admin comments unexpected: %d %s", adminList.Code, adminList.Body.String())
	}
	hideParams := gin.Params{{Key: "id", Value: fmt.Sprint(comment.ID)}}
	hide := authenticatedRequest(community.AdminUpdateCommentStatus, http.MethodPatch, "/api/admin/comments/1/status", `{"status":"hidden"}`, hideParams, users[0].ID, "admin")
	if hide.Code != http.StatusOK {
		t.Fatalf("hide returned %d: %s", hide.Code, hide.Body.String())
	}
	var storedPost models.Post
	if err := db.First(&storedPost, post.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedPost.CommentsCount != 0 {
		t.Fatalf("expected comments count 0 after hide, got %d", storedPost.CommentsCount)
	}
	publicList := performRequest(community.ListComments, http.MethodGet, "/api/posts/post/comments", "", gin.Params{{Key: "slug", Value: "post"}})
	if strings.Contains(publicList.Body.String(), "spam message") {
		t.Fatalf("hidden comment should not be public: %s", publicList.Body.String())
	}
	restore := authenticatedRequest(community.AdminUpdateCommentStatus, http.MethodPatch, "/api/admin/comments/1/status", `{"status":"published"}`, hideParams, users[0].ID, "admin")
	if restore.Code != http.StatusOK {
		t.Fatalf("restore returned %d: %s", restore.Code, restore.Body.String())
	}
	if err := db.First(&storedPost, post.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedPost.CommentsCount != 1 {
		t.Fatalf("expected comments count 1 after restore, got %d", storedPost.CommentsCount)
	}
}

func TestDeleteCommentRemovesDescendantsAndRebuildsCount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newCommunityTestDB(t)
	users := []models.User{
		{Username: "author", Nickname: "Author", Role: "user", Status: "active"},
		{Username: "alice", Nickname: "Alice", Role: "user", Status: "active"},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	post := models.Post{AuthorID: users[0].ID, Title: "Post", Slug: "post", Content: "body", Status: "published", CommentsCount: 99, PublishedAt: time.Now()}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	root := models.Comment{PostID: post.ID, AuthorID: users[0].ID, Content: "root", Status: "published"}
	if err := db.Create(&root).Error; err != nil {
		t.Fatal(err)
	}
	reply := models.Comment{PostID: post.ID, AuthorID: users[1].ID, ParentID: &root.ID, Content: "reply", Status: "published"}
	if err := db.Create(&reply).Error; err != nil {
		t.Fatal(err)
	}
	nested := models.Comment{PostID: post.ID, AuthorID: users[0].ID, ParentID: &reply.ID, Content: "nested", Status: "published"}
	if err := db.Create(&nested).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.CommentLike{UserID: users[1].ID, CommentID: nested.ID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Notification{UserID: users[0].ID, ActorID: users[1].ID, Type: "reply", ResourceID: nested.ID}).Error; err != nil {
		t.Fatal(err)
	}

	community := &CommunityController{DB: db}
	response := authenticatedRequest(community.DeleteComment, http.MethodDelete, "/api/comments/1", "", gin.Params{{Key: "id", Value: fmt.Sprint(root.ID)}}, users[0].ID, "user")
	if response.Code != http.StatusOK {
		t.Fatalf("delete returned %d: %s", response.Code, response.Body.String())
	}
	var remaining int64
	db.Model(&models.Comment{}).Where("post_id = ?", post.ID).Count(&remaining)
	if remaining != 0 {
		t.Fatalf("expected all descendants removed, got %d comments", remaining)
	}
	db.Model(&models.CommentLike{}).Where("comment_id IN ?", []uint{root.ID, reply.ID, nested.ID}).Count(&remaining)
	if remaining != 0 {
		t.Fatalf("expected comment likes removed, got %d", remaining)
	}
	db.Model(&models.Notification{}).Where("type = ? AND resource_id = ?", "reply", nested.ID).Count(&remaining)
	if remaining != 0 {
		t.Fatalf("expected reply notifications removed, got %d", remaining)
	}
	var storedPost models.Post
	if err := db.First(&storedPost, post.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedPost.CommentsCount != 0 {
		t.Fatalf("expected rebuilt comments count 0, got %d", storedPost.CommentsCount)
	}
}

func TestPostRevisionHistoryAndRestore(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newCommunityTestDB(t)
	user := models.User{Username: "author", Nickname: "Author", Role: "user", Status: "active"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	post := models.Post{AuthorID: user.ID, Title: "Version one", Slug: "version-one", Summary: "first", Content: "# First", Status: "draft", ReadingTime: 1}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	community := &CommunityController{DB: db}
	params := gin.Params{{Key: "id", Value: fmt.Sprint(post.ID)}}

	update := authenticatedRequest(community.UpdatePost, http.MethodPut, "/api/posts/id/1", `{"title":"Version two","slug":"version-two","summary":"second","content":"# Second","status":"draft","tags":["Go"]}`, params, user.ID, "user")
	if update.Code != http.StatusOK {
		t.Fatalf("update returned %d: %s", update.Code, update.Body.String())
	}
	var revision models.PostRevision
	if err := db.Where("post_id = ?", post.ID).First(&revision).Error; err != nil {
		t.Fatal(err)
	}
	if revision.Title != "Version one" || revision.Content != "# First" {
		t.Fatalf("unexpected revision snapshot: %#v", revision)
	}

	list := authenticatedRequest(community.PostRevisions, http.MethodGet, "/api/posts/id/1/revisions", "", params, user.ID, "user")
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), "Version one") {
		t.Fatalf("unexpected revisions response: %d %s", list.Code, list.Body.String())
	}
	restoreParams := gin.Params{{Key: "id", Value: fmt.Sprint(post.ID)}, {Key: "revisionId", Value: fmt.Sprint(revision.ID)}}
	restore := authenticatedRequest(community.RestorePostRevision, http.MethodPost, "/api/posts/id/1/revisions/1/restore", "", restoreParams, user.ID, "user")
	if restore.Code != http.StatusOK {
		t.Fatalf("restore returned %d: %s", restore.Code, restore.Body.String())
	}
	if err := db.First(&post, post.ID).Error; err != nil {
		t.Fatal(err)
	}
	if post.Title != "Version one" || post.Content != "# First" {
		t.Fatalf("post was not restored: %#v", post)
	}
	var revisionCount int64
	if err := db.Model(&models.PostRevision{}).Where("post_id = ?", post.ID).Count(&revisionCount).Error; err != nil {
		t.Fatal(err)
	}
	if revisionCount != 2 {
		t.Fatalf("expected current content to be snapshotted before restore, got %d revisions", revisionCount)
	}
}
