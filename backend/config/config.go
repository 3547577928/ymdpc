package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	defaultJWTSecret     = "change-me-in-production"
	defaultAdminPassword = "admin123"
)

type Config struct {
	Port           string
	DatabasePath   string
	UploadDir      string
	JWTSecret      string
	AllowedOrigins string
	AdminUsername  string
	AdminPassword  string
	CookieSecure   bool
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return Config{
		Port:           port,
		DatabasePath:   envOr("DATABASE_PATH", "./data/quietsig.db"),
		UploadDir:      envOr("UPLOAD_DIR", "./data/uploads"),
		JWTSecret:      envOr("JWT_SECRET", defaultJWTSecret),
		AllowedOrigins: envOr("ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173"),
		AdminUsername:  envOr("ADMIN_USERNAME", "admin"),
		AdminPassword:  envOr("ADMIN_PASSWORD", defaultAdminPassword),
		CookieSecure:   os.Getenv("COOKIE_SECURE") == "true",
	}
}

func (c Config) ValidateProduction() error {
	var invalid []string
	if c.JWTSecret == defaultJWTSecret || len(c.JWTSecret) < 32 {
		invalid = append(invalid, "JWT_SECRET must contain at least 32 characters")
	}
	if c.AdminPassword == defaultAdminPassword || len(c.AdminPassword) < 12 {
		invalid = append(invalid, "ADMIN_PASSWORD must contain at least 12 characters")
	}
	if len(invalid) > 0 {
		return fmt.Errorf("invalid production configuration: %s", strings.Join(invalid, "; "))
	}
	return nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func IntEnv(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return value
}
