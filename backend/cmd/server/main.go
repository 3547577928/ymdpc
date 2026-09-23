package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"quietsignal/backend/config"
	"quietsignal/backend/controllers"
	"quietsignal/backend/models"
	"quietsignal/backend/routes"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()
	if gin.Mode() == gin.ReleaseMode {
		if err := cfg.ValidateProduction(); err != nil {
			log.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(cfg.DatabasePath), 0o755); err != nil {
		log.Fatalf("create database directory: %v", err)
	}
	separator := "?"
	if strings.Contains(cfg.DatabasePath, "?") {
		separator = "&"
	}
	// 迁移连接不启用外键：SQLite 变更表结构时需要重建表，外键引用会阻止 DROP TABLE
	// TranslateError 把驱动错误翻译为 gorm.ErrDuplicatedKey 等语义错误，
	// 文章 slug 唯一索引冲突时的重试依赖它识别
	migrateDSN := cfg.DatabasePath + separator + "_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	migrateDB, err := gorm.Open(sqlite.Open(migrateDSN), &gorm.Config{TranslateError: true})
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	if err := migrateDB.AutoMigrate(&models.User{}, &models.Post{}, &models.Tag{}, &models.Comment{}, &models.PostLike{}, &models.Follow{}, &models.Category{}, &models.Favorite{}, &models.CommentLike{}, &models.Notification{}, &models.Report{}, &models.AdminLog{}, &models.Setting{}); err != nil {
		log.Fatalf("migrate database: %v", err)
	}
	if sqlDB, err := migrateDB.DB(); err == nil {
		sqlDB.Close()
	}
	// 运行时连接保持外键开启，由数据库兜底引用完整性
	dsn := cfg.DatabasePath + separator + "_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	if err := models.Seed(db, cfg.AdminUsername, cfg.AdminPassword); err != nil {
		log.Fatalf("seed database: %v", err)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), cors.New(corsConfig(cfg.AllowedOrigins, gin.Mode() != gin.ReleaseMode)))
	routes.Register(r, routes.Dependencies{Posts: &controllers.PostController{DB: db}, Community: &controllers.CommunityController{DB: db}, Interactions: &controllers.InteractionController{DB: db}, Tags: &controllers.TagController{DB: db}, Auth: &controllers.AuthController{DB: db, Secret: cfg.JWTSecret, CookieSecure: cfg.CookieSecure}, Secret: cfg.JWTSecret, DB: db})

	log.Printf("quiet signal api listening on :%s", cfg.Port)
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("serve api: %v", err)
		}
	}()

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, syscall.SIGINT, syscall.SIGTERM)
	<-shutdownSignal
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown api: %v", err)
	}
}

func corsConfig(configuredOrigins string, allowLocalDevelopment bool) cors.Config {
	allowedOrigins := make(map[string]struct{})
	for _, origin := range strings.Split(configuredOrigins, ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			allowedOrigins[origin] = struct{}{}
		}
	}

	return cors.Config{
		AllowOriginFunc: func(origin string) bool {
			if _, ok := allowedOrigins[origin]; ok {
				return true
			}
			if !allowLocalDevelopment {
				return false
			}

			parsed, err := url.Parse(origin)
			if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
				return false
			}
			hostname := parsed.Hostname()
			return hostname == "localhost" || net.ParseIP(hostname).IsLoopback()
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}
}
