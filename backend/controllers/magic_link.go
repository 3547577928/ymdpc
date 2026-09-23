package controllers

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/url"
	"regexp"
	"strings"
	"time"

	"quietsignal/backend/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var emailLocalPartPattern = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func normalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func validateEmail(value string) error {
	if value == "" || len(value) > 254 {
		return errors.New("请输入有效邮箱")
	}
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value || !strings.Contains(value, "@") {
		return errors.New("请输入有效邮箱")
	}
	return nil
}

func emailTaken(db *gorm.DB, email string, excludedUserID uint) bool {
	query := db.Model(&models.User{}).Where("email = ?", email)
	if excludedUserID > 0 {
		query = query.Where("id <> ?", excludedUserID)
	}
	var count int64
	return query.Count(&count).Error == nil && count > 0
}

func randomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (a *AuthController) RequestMagicLink(c *gin.Context) {
	var input struct {
		Email string `json:"email" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请输入邮箱"})
		return
	}
	email := normalizeEmail(input.Email)
	if err := validateEmail(email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if a.SMTPPassword == "" && a.SendMagicLink == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "邮件服务尚未配置，请联系管理员"})
		return
	}

	var user models.User
	err := a.DB.Where("email = ?", email).First(&user).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "发送登录链接失败"})
		return
	}
	if err == nil && user.Status != "active" {
		// 对外继续返回同一提示，避免通过接口枚举被限制的账号。
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "如果邮箱可用，登录链接已发送，请查收邮件。"})
		return
	}

	token, err := randomToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "生成登录链接失败"})
		return
	}
	ttl := a.MagicLinkTTL
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	magicToken := models.MagicLinkToken{Email: email, TokenHash: hashToken(token), ExpiresAt: time.Now().Add(ttl), RequestIP: c.ClientIP()}
	if err == nil {
		magicToken.UserID = &user.ID
	}
	if tx := a.DB.Where("email = ? AND used_at IS NULL", email).Delete(&models.MagicLinkToken{}); tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "发送登录链接失败"})
		return
	}
	if err := a.DB.Create(&magicToken).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "发送登录链接失败"})
		return
	}

	link := strings.TrimRight(a.AppBaseURL, "/") + "/auth/magic-link?token=" + url.QueryEscape(token)
	if err := a.sendMagicLink(email, link); err != nil {
		a.DB.Delete(&magicToken)
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "邮件发送失败，请稍后重试"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "如果邮箱可用，登录链接已发送，请查收邮件。"})
}

func (a *AuthController) VerifyMagicLink(c *gin.Context) {
	var input struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || len(input.Token) < 20 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "登录链接无效"})
		return
	}
	var stored models.MagicLinkToken
	if err := a.DB.Where("token_hash = ? AND used_at IS NULL", hashToken(input.Token)).First(&stored).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "登录链接无效或已使用"})
		return
	}
	if time.Now().After(stored.ExpiresAt) {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "登录链接已过期，请重新获取"})
		return
	}

	var user models.User
	err := a.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		result := tx.Model(&models.MagicLinkToken{}).Where("id = ? AND used_at IS NULL", stored.ID).Update("used_at", now)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return gorm.ErrDuplicatedKey
		}
		if err := tx.Where("email = ?", stored.Email).First(&user).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			if !models.BoolSetting(tx, "open_registration", true) {
				return errRegistrationClosed
			}
			created, createErr := createMagicLinkUser(tx, stored.Email, input.Token)
			if createErr != nil {
				return createErr
			}
			user = created
		} else if err != nil {
			return err
		}
		if user.Status != "active" {
			return errAccountDisabled
		}
		return nil
	})
	if err != nil {
		status := http.StatusInternalServerError
		message := "登录失败"
		if errors.Is(err, errRegistrationClosed) {
			status, message = http.StatusForbidden, "当前已关闭公开注册，该邮箱尚未绑定账号"
		} else if errors.Is(err, errAccountDisabled) {
			status, message = http.StatusForbidden, "账号已被限制登录"
		} else if errors.Is(err, gorm.ErrDuplicatedKey) {
			status, message = http.StatusUnauthorized, "登录链接无效或已使用"
		}
		c.JSON(status, gin.H{"code": status, "message": message})
		return
	}
	a.issueSession(c, user)
}

var (
	errRegistrationClosed = errors.New("registration closed")
	errAccountDisabled    = errors.New("account disabled")
)

func createMagicLinkUser(db *gorm.DB, email, token string) (models.User, error) {
	base := emailLocalPartPattern.ReplaceAllString(strings.Split(email, "@")[0], "-")
	base = strings.Trim(base, "-_")
	if len(base) < 3 {
		base = "writer"
	}
	if len(base) > 50 {
		base = base[:50]
	}
	username := base
	for suffix := 2; ; suffix++ {
		var count int64
		if err := db.Model(&models.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
			return models.User{}, err
		}
		if count == 0 {
			break
		}
		username = fmt.Sprintf("%s-%d", base, suffix)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("magic-link:"+token), bcrypt.DefaultCost)
	if err != nil {
		return models.User{}, err
	}
	user := models.User{Username: username, Email: email, PasswordHash: string(hash), Nickname: username, Role: "user", Status: "active", SessionVersion: 1}
	if err := db.Create(&user).Error; err != nil {
		return models.User{}, err
	}
	return user, nil
}

func (a *AuthController) sendMagicLink(to, link string) error {
	if a.SendMagicLink != nil {
		return a.SendMagicLink(to, link)
	}
	if a.SMTPPassword == "" {
		return errors.New("smtp is not configured")
	}
	host := a.SMTPHost
	if host == "" {
		host = "smtp.qq.com"
	}
	port := a.SMTPPort
	if port <= 0 {
		port = 465
	}
	username := a.SMTPUsername
	from := a.SMTPFrom
	if from == "" {
		from = username
	}
	message := buildEmailMessage(from, to, "Quiet Signal 登录链接", fmt.Sprintf("你好，\n\n请点击以下链接登录 Quiet Signal：\n%s\n\n该链接 15 分钟内有效，且只能使用一次。\n如果这不是你的操作，请忽略此邮件。\n", link))
	address := net.JoinHostPort(host, fmt.Sprint(port))
	var client *smtp.Client
	var err error
	if port == 465 {
		conn, dialErr := tls.Dial("tcp", address, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
		if dialErr != nil {
			return dialErr
		}
		client, err = smtp.NewClient(conn, host)
	} else {
		client, err = smtp.Dial(address)
		if err == nil {
			err = client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
		}
	}
	if err != nil {
		return err
	}
	defer client.Close()
	if err := client.Auth(smtp.PlainAuth("", username, a.SMTPPassword, host)); err != nil {
		return err
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func buildEmailMessage(from, to, subject, body string) []byte {
	encodedSubject := "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(subject)) + "?="
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n%s", from, to, encodedSubject, strings.ReplaceAll(body, "\n", "\r\n"))
	return []byte(message)
}
