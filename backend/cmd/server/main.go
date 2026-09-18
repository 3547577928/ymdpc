package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"quietsignal/backend/config"
	"quietsignal/backend/controllers"
	"quietsignal/backend/models"
	"quietsignal/backend/routes"
)

func main() {
	cfg := config.Load()
	if err := os.MkdirAll(filepath.Dir(cfg.DatabasePath), 0o755); err != nil {
		log.Fatalf("create database directory: %v", err)
	}
	db, err := gorm.Open(sqlite.Open(cfg.DatabasePath), &gorm.Config{})
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Post{}, &models.Tag{}); err != nil {
		log.Fatalf("migrate database: %v", err)
	}
	if err := models.Seed(db, cfg.AdminUsername, cfg.AdminPassword); err != nil {
		log.Fatalf("seed database: %v", err)
	}

	r := gin.New()
	allowedOrigins := strings.Split(cfg.AllowedOrigins, ",")
	r.Use(gin.Logger(), gin.Recovery(), cors.New(cors.Config{AllowOrigins: allowedOrigins, AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}, AllowHeaders: []string{"Origin", "Content-Type", "Authorization"}, AllowCredentials: true}))
	routes.Register(r, routes.Dependencies{Posts: &controllers.PostController{DB: db}, Tags: &controllers.TagController{DB: db}, Auth: &controllers.AuthController{DB: db, Secret: cfg.JWTSecret, CookieSecure: cfg.CookieSecure}, Secret: cfg.JWTSecret})

	log.Printf("quiet signal api listening on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
