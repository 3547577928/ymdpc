package routes

import (
	"github.com/gin-gonic/gin"
	"quietsignal/backend/controllers"
	"quietsignal/backend/middleware"
)

type Dependencies struct {
	Posts  *controllers.PostController
	Tags   *controllers.TagController
	Auth   *controllers.AuthController
	Secret string
}

func Register(r *gin.Engine, deps Dependencies) {
	api := r.Group("/api")
	api.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"code": 0, "message": "ok"}) })
	api.POST("/auth/login", deps.Auth.Login)
	api.POST("/auth/logout", deps.Auth.Logout)
	api.GET("/posts", deps.Posts.List)
	api.GET("/posts/:slug", deps.Posts.Detail)
	api.GET("/tags", deps.Tags.List)

	admin := api.Group("/admin", middleware.RequireAuth(deps.Secret))
	admin.GET("/posts", deps.Posts.AdminList)
	admin.GET("/posts/:id", deps.Posts.AdminDetail)
	admin.POST("/posts", deps.Posts.Create)
	admin.PUT("/posts/:id", deps.Posts.Update)
	admin.DELETE("/posts/:id", deps.Posts.Delete)
	admin.PATCH("/posts/:id/status", deps.Posts.UpdateStatus)
	admin.GET("/me", deps.Auth.Me)
	api.GET("/auth/me", middleware.RequireAuth(deps.Secret), deps.Auth.Me)
}
