package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"quietsignal/backend/models"
)

type AuthController struct {
	DB           *gorm.DB
	Secret       string
	CookieSecure bool
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
	if err := a.DB.Where("username = ?", input.Username).First(&user).Error; err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "用户名或密码错误"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": user.ID, "exp": time.Now().Add(24 * time.Hour).Unix()})
	signed, err := token.SignedString([]byte(a.Secret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "登录失败"})
		return
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("qs_token", signed, 86400, "/", "", a.CookieSecure, true)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"id": user.ID, "username": user.Username, "nickname": user.Nickname}})
}

func (a *AuthController) Logout(c *gin.Context) {
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
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"id": user.ID, "username": user.Username, "nickname": user.Nickname}})
}
