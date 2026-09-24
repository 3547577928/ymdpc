package routes

import (
	"time"

	"quietsignal/backend/controllers"
	"quietsignal/backend/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Dependencies struct {
	Posts        *controllers.PostController
	Community    *controllers.CommunityController
	Interactions *controllers.InteractionController
	Tags         *controllers.TagController
	Auth         *controllers.AuthController
	Uploads      *controllers.UploadController
	Secret       string
	// DB 供管理端中间件复核角色与账号状态，见 middleware.RequireAdmin
	DB *gorm.DB
}

func Register(r *gin.Engine, deps Dependencies) {
	// 限流：登录注册按 IP 每分钟 10 次，发文、评论、点赞、收藏、举报等写操作每分钟 30 次
	authLimit := middleware.RateLimit(10, time.Minute)
	writeLimit := middleware.RateLimit(30, time.Minute)

	api := r.Group("/api")
	api.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"code": 0, "message": "ok"}) })
	api.POST("/auth/register", authLimit, deps.Auth.Register)
	api.POST("/auth/login", authLimit, deps.Auth.Login)
	api.POST("/auth/email-code/request", authLimit, deps.Auth.RequestEmailCode)
	api.POST("/auth/email-code/verify", authLimit, deps.Auth.VerifyEmailCode)
	api.POST("/auth/logout", middleware.RequireAuth(deps.Secret, deps.DB), deps.Auth.Logout)
	api.GET("/auth/me", middleware.RequireAuth(deps.Secret, deps.DB), deps.Auth.Me)
	api.PATCH("/me/profile", middleware.RequireAuth(deps.Secret, deps.DB), deps.Auth.UpdateProfile)
	api.PATCH("/me/password", middleware.RequireAuth(deps.Secret, deps.DB), deps.Auth.UpdatePassword)
	api.GET("/me/posts", middleware.RequireAuth(deps.Secret, deps.DB), deps.Interactions.MyPosts)
	api.GET("/me/favorites", middleware.RequireAuth(deps.Secret, deps.DB), deps.Interactions.MyFavorites)
	api.GET("/notifications", middleware.RequireAuth(deps.Secret, deps.DB), deps.Interactions.Notifications)
	api.PATCH("/notifications/:id/read", middleware.RequireAuth(deps.Secret, deps.DB), deps.Interactions.MarkNotificationRead)
	api.POST("/notifications/read-all", middleware.RequireAuth(deps.Secret, deps.DB), deps.Interactions.MarkAllNotificationsRead)
	api.POST("/reports", writeLimit, middleware.RequireAuth(deps.Secret, deps.DB), deps.Interactions.CreateReport)
	api.POST("/uploads/images", writeLimit, middleware.RequireAuth(deps.Secret, deps.DB), deps.Uploads.Image)
	api.GET("/categories", deps.Interactions.ListCategories)
	api.GET("/trending/tags", deps.Interactions.TrendingTags)
	api.POST("/categories", writeLimit, middleware.RequireAuth(deps.Secret, deps.DB), deps.Interactions.CreateCategory)
	api.GET("/feed", middleware.OptionalAuth(deps.Secret, deps.DB), deps.Community.Feed)
	api.GET("/users/:username", middleware.OptionalAuth(deps.Secret, deps.DB), deps.Community.Profile)
	api.POST("/users/:id/follow", writeLimit, middleware.RequireAuth(deps.Secret, deps.DB), deps.Community.FollowUser)
	api.DELETE("/users/:id/follow", middleware.RequireAuth(deps.Secret, deps.DB), deps.Community.UnfollowUser)
	api.GET("/posts", middleware.OptionalAuth(deps.Secret, deps.DB), deps.Posts.List)
	api.POST("/posts", writeLimit, middleware.RequireAuth(deps.Secret, deps.DB), deps.Community.CreatePost)
	api.GET("/posts/id/:id", middleware.RequireAuth(deps.Secret, deps.DB), deps.Interactions.OwnerPostDetail)
	api.GET("/posts/id/:id/revisions", middleware.RequireAuth(deps.Secret, deps.DB), deps.Community.PostRevisions)
	api.POST("/posts/id/:id/revisions/:revisionId/restore", writeLimit, middleware.RequireAuth(deps.Secret, deps.DB), deps.Community.RestorePostRevision)
	api.PUT("/posts/id/:id", writeLimit, middleware.RequireAuth(deps.Secret, deps.DB), deps.Community.UpdatePost)
	api.DELETE("/posts/id/:id", writeLimit, middleware.RequireAuth(deps.Secret, deps.DB), deps.Community.DeletePost)
	api.GET("/posts/:slug", middleware.OptionalAuth(deps.Secret, deps.DB), deps.Posts.Detail)
	api.POST("/posts/:slug/views", writeLimit, deps.Posts.RecordView)
	api.POST("/posts/:slug/favorite", writeLimit, middleware.RequireAuth(deps.Secret, deps.DB), deps.Interactions.FavoritePost)
	api.DELETE("/posts/:slug/favorite", writeLimit, middleware.RequireAuth(deps.Secret, deps.DB), deps.Interactions.UnfavoritePost)
	api.GET("/posts/:slug/comments", deps.Community.ListComments)
	api.POST("/posts/:slug/comments", writeLimit, middleware.RequireAuth(deps.Secret, deps.DB), deps.Community.CreateComment)
	api.POST("/posts/:slug/like", writeLimit, middleware.RequireAuth(deps.Secret, deps.DB), deps.Community.LikePost)
	api.DELETE("/posts/:slug/like", writeLimit, middleware.RequireAuth(deps.Secret, deps.DB), deps.Community.UnlikePost)
	api.POST("/comments/:id/like", writeLimit, middleware.RequireAuth(deps.Secret, deps.DB), deps.Interactions.LikeComment)
	api.DELETE("/comments/:id/like", writeLimit, middleware.RequireAuth(deps.Secret, deps.DB), deps.Interactions.UnlikeComment)
	api.DELETE("/comments/:id", writeLimit, middleware.RequireAuth(deps.Secret, deps.DB), deps.Community.DeleteComment)
	api.PATCH("/comments/:id/pin", writeLimit, middleware.RequireAuth(deps.Secret, deps.DB), deps.Community.PinComment)
	api.GET("/tags", deps.Tags.List)

	admin := api.Group("/admin", middleware.RequireAdmin(deps.Secret, deps.DB))
	admin.GET("/posts", deps.Posts.AdminList)
	admin.GET("/stats", deps.Community.UserStats)
	admin.GET("/users", deps.Community.AdminUsers)
	admin.PATCH("/users/:id", deps.Community.UpdateUser)
	admin.GET("/posts/:id", deps.Posts.AdminDetail)
	admin.POST("/posts", deps.Posts.Create)
	admin.PUT("/posts/:id", deps.Posts.Update)
	admin.DELETE("/posts/:id", deps.Posts.Delete)
	admin.PATCH("/posts/:id/status", deps.Posts.UpdateStatus)
	admin.PATCH("/posts/:id/moderation", deps.Posts.UpdateModeration)
	admin.GET("/comments", deps.Community.AdminComments)
	admin.PATCH("/comments/:id/status", deps.Community.AdminUpdateCommentStatus)
	admin.POST("/comments/batch", deps.Community.AdminBatchUpdateCommentStatus)
	admin.GET("/reports", deps.Interactions.AdminReports)
	admin.GET("/reports/:id", deps.Interactions.AdminReportDetail)
	admin.PATCH("/reports/:id", deps.Interactions.AdminHandleReport)
	admin.POST("/reports/batch", deps.Interactions.AdminHandleReportsBatch)
	admin.GET("/logs", deps.Interactions.AdminLogs)
	admin.GET("/export", deps.Interactions.AdminExport)
	admin.GET("/settings", deps.Interactions.AdminSettings)
	admin.POST("/uploads/cleanup", deps.Uploads.CleanupUploads)
	admin.PATCH("/settings", deps.Interactions.AdminUpdateSettings)
	admin.GET("/tags", deps.Interactions.AdminTags)
	admin.POST("/categories", deps.Interactions.CreateCategory)
	admin.DELETE("/categories/:id", deps.Interactions.DeleteCategory)
	admin.DELETE("/tags/:id", deps.Interactions.DeleteTag)
	admin.GET("/me", deps.Auth.Me)
}
