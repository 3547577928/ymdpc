package middleware

import (
	"net/http"
	"strings"

	"quietsignal/backend/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// userIDFromContext 读取 setClaims 写入的当前用户 ID，未认证时为 0
func userIDFromContext(c *gin.Context) uint {
	value, ok := c.Get("userID")
	if !ok {
		return 0
	}
	id, _ := value.(uint)
	return id
}

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

func RequireAdmin(secret string, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !setClaims(c, secret) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "unauthorized"})
			return
		}
		// token 签发时写入的 role 声明在 24 小时内不会更新：管理员被降级或封禁后，
		// 旧 token 仍带着 admin 声明。这里按数据库实时状态复核，权限变更即时生效。
		// 普通写接口的账号状态由 CommunityController.ensureActiveUser 覆盖
		var user models.User
		if err := db.Select("role", "status").First(&user, userIDFromContext(c)).Error; err != nil || user.Role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "message": "需要管理员权限"})
			return
		}
		if user.Status == "banned" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "message": "账号已被封禁"})
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
