package controllers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"quietsignal/backend/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthController struct {
	DB           *gorm.DB
	Secret       string
	CookieSecure bool
}

type AuthUserDTO struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Bio      string `json:"bio"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

func (a *AuthController) Register(c *gin.Context) {
	// 管理员可关闭公开注册
	if !models.BoolSetting(a.DB, "open_registration", true) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "当前已关闭公开注册"})
		return
	}
	var input struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Nickname string `json:"nickname"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "用户名和密码不能为空"})
		return
	}
	input.Username = strings.TrimSpace(input.Username)
	input.Nickname = strings.TrimSpace(input.Nickname)
	if len(input.Username) < 3 || len(input.Username) > 60 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "用户名长度需要在 3 到 60 个字符之间"})
		return
	}
	if len(input.Password) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "密码至少需要 8 个字符"})
		return
	}
	if input.Nickname == "" {
		input.Nickname = input.Username
	}
	var count int64
	if err := a.DB.Model(&models.User{}).Where("username = ?", input.Username).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "注册失败"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"code": 409, "message": "用户名已存在"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "注册失败"})
		return
	}
	// SessionVersion 显式置 1（与列默认值一致）：Create 后 GORM 不一定把默认值回填到
	// 结构体，签发 session 时用字面量值才是最可靠的
	user := models.User{Username: input.Username, PasswordHash: string(hash), Nickname: input.Nickname, Role: "user", Status: "active", SessionVersion: 1}
	if err := a.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "注册失败"})
		return
	}
	a.issueSession(c, user)
}

func (a *AuthController) Login(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "用户名和密码不能为空"})
		return
	}

	var user models.User
	err := a.DB.Where("username = ?", input.Username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "用户名或密码错误"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "登录失败"})
		return
	}
	if user.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "账号已被限制登录"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "用户名或密码错误"})
		return
	}

	a.issueSession(c, user)
}

func (a *AuthController) issueSession(c *gin.Context, user models.User) {
	// sv 写入会话版本号，中间件校验 token 时会与数据库现值比对。
	// 兼容尚未经过一次启动迁移的旧用户，避免签发版本为 0 的不可用 token。
	version := user.SessionVersion
	if version < 1 {
		version = 1
		if err := a.DB.Model(&models.User{}).Where("id = ?", user.ID).Update("session_version", version).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "登录失败"})
			return
		}
		user.SessionVersion = version
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": user.ID, "role": user.Role, "sv": version, "exp": time.Now().Add(24 * time.Hour).Unix()})
	signed, err := token.SignedString([]byte(a.Secret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "登录失败"})
		return
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("qs_token", signed, 86400, "/", "", a.CookieSecure, true)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": toAuthUserDTO(user)})
}

// bumpSessionVersion 递增会话版本号，使该操作之前签发的 token 全部失效
func (a *AuthController) bumpSessionVersion(userID uint) error {
	return a.DB.Model(&models.User{}).Where("id = ?", userID).Update("session_version", gorm.Expr("session_version + 1")).Error
}

func (a *AuthController) Logout(c *gin.Context) {
	// 登出即视为所有设备下线：递增会话版本号，旧 token 不再被中间件接受
	if userID := currentUserID(c); userID > 0 {
		if err := a.bumpSessionVersion(userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "退出登录失败"})
			return
		}
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("qs_token", "", -1, "/", "", a.CookieSecure, true)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

func (a *AuthController) Me(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "unauthorized"})
		return
	}
	var user models.User
	if err := a.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "unauthorized"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": toAuthUserDTO(user)})
}

func (a *AuthController) UpdateProfile(c *gin.Context) {
	userID := currentUserID(c)
	var input struct {
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
		Bio      string `json:"bio"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "资料格式错误"})
		return
	}
	updates := map[string]any{"nickname": strings.TrimSpace(input.Nickname), "avatar": strings.TrimSpace(input.Avatar), "bio": strings.TrimSpace(input.Bio)}
	if updates["nickname"] == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "昵称不能为空"})
		return
	}
	if err := a.DB.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "保存资料失败"})
		return
	}
	var user models.User
	if err := a.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取资料失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": toAuthUserDTO(user)})
}

// UpdatePassword 修改当前登录用户的密码，需要先验证原密码
func (a *AuthController) UpdatePassword(c *gin.Context) {
	userID := currentUserID(c)
	var input struct {
		CurrentPassword string `json:"currentPassword" binding:"required"`
		NewPassword     string `json:"newPassword" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "密码不能为空"})
		return
	}
	if len(input.NewPassword) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "新密码至少需要 8 个字符"})
		return
	}
	var user models.User
	if err := a.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "unauthorized"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.CurrentPassword)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "当前密码错误"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "修改密码失败"})
		return
	}
	// 换密码成功后递增会话版本号：当前 token 随之失效，用户需重新登录，
	// 避免密码泄露后旧会话在其他设备上继续可用
	if err := a.DB.Model(&user).Updates(map[string]any{"password_hash": string(hash), "session_version": gorm.Expr("session_version + 1")}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "修改密码失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

func toAuthUserDTO(user models.User) AuthUserDTO {
	return AuthUserDTO{ID: user.ID, Username: user.Username, Nickname: user.Nickname, Avatar: user.Avatar, Bio: user.Bio, Role: user.Role, Status: user.Status}
}
