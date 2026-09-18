package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port           string
	DatabasePath   string
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
		JWTSecret:      envOr("JWT_SECRET", "change-me-in-production"),
		AllowedOrigins: envOr("ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173"),
		AdminUsername:  envOr("ADMIN_USERNAME", "admin"),
		AdminPassword:  envOr("ADMIN_PASSWORD", "admin123"),
		CookieSecure:   os.Getenv("COOKIE_SECURE") == "true",
	}
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
