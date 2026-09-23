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

// sessionActive 复核登录会话：用户存在、未被封禁，且 token 中的会话版本号
// 与数据库现值一致。JWT 签发后在有效期内无法作废，版本号是让旧 token
// 立即失效的唯一手段（改密码、退出登录时递增）
func sessionActive(db *gorm.DB, c *gin.Context) bool {
	var user models.User
	if err := db.Select("status", "session_version").First(&user, userIDFromContext(c)).Error; err != nil || user.Status == "banned" {
		return false
	}
	tokenVersion, ok := c.Get("sessionVersion")
	version, ok := tokenVersion.(int)
	return ok && version > 0 && version == user.SessionVersion
}

// RequireAuth 校验 JWT 并复核会话状态，见 sessionActive
func RequireAuth(secret string, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !setClaims(c, secret) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "unauthorized"})
			return
		}
		if !sessionActive(db, c) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "登录状态已失效，请重新登录"})
			return
		}
		c.Next()
	}
}

func OptionalAuth(secret string, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if setClaims(c, secret) && !sessionActive(db, c) {
			// 公开接口仍然可以访问，但无效会话不能影响 liked/favorited/following 等结果。
			c.Set("userID", uint(0))
			c.Set("sessionVersion", 0)
			c.Set("userRole", "")
		}
		c.Next()
	}
}

func RequireAdmin(secret string, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !setClaims(c, secret) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "unauthorized"})
			return
		}
		// 会话复核（含会话版本号）失败说明 token 已被改密码/登出作废
		if !sessionActive(db, c) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "unauthorized"})
			return
		}
		// token 签发时写入的 role 声明在 24 小时内不会更新：管理员被降级后，
		// 旧 token 仍带着 admin 声明。这里按数据库实时状态复核，权限变更即时生效
		var user models.User
		if err := db.Select("role").First(&user, userIDFromContext(c)).Error; err != nil || user.Role != "admin" {
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
	// sv 为会话版本号：旧版本 token（无此 claim）解析为 0，与数据库现值必然不符，
	// 效果是强制这些 token 的持有者重新登录一次
	if version, ok := claims["sv"].(float64); ok {
		c.Set("sessionVersion", int(version))
	} else {
		c.Set("sessionVersion", 0)
	}
	return true
}
