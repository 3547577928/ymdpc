package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func RequireAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !setClaims(c, secret) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "unauthorized"})
			return
		}
		c.Next()
	}
}

func OptionalAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		setClaims(c, secret)
		c.Next()
	}
}

func RequireAdmin(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !setClaims(c, secret) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "unauthorized"})
			return
		}
		tokenRole, ok := c.Get("userRole")
		if !ok || tokenRole != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "message": "需要管理员权限"})
			return
		}
		c.Next()
	}
}

func setClaims(c *gin.Context, secret string) bool {
	tokenString := ""
	if cookie, err := c.Cookie("qs_token"); err == nil {
		tokenString = cookie
	}
	if tokenString == "" {
		tokenString = strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	}
	if tokenString == "" {
		return false
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		return false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return false
	}
	subject, ok := claims["sub"].(float64)
	if !ok || subject < 1 {
		return false
	}
	c.Set("userID", uint(subject))
	if role, ok := claims["role"].(string); ok {
		c.Set("userRole", role)
	}
	return true
}
