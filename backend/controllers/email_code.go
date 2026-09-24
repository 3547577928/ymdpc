package controllers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"regexp"
	"strings"
	"time"

	"quietsignal/backend/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	emailLocalPartPattern = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)
	emailCodePattern      = regexp.MustCompile(`^[0-9]{6}$`)
	errRegistrationClosed = errors.New("registration closed")
	errAccountDisabled    = errors.New("account disabled")
	errCodeConsumed       = errors.New("email code consumed")
)

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

func randomEmailCode() (string, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", value.Int64()), nil
}

func hashEmailCode(secret, email, code string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(email + ":" + code))
	return hex.EncodeToString(mac.Sum(nil))
}

func (a *AuthController) RequestEmailCode(c *gin.Context) {
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
	if a.SMTPPassword == "" && a.SendEmailCode == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "邮件服务尚未配置，请联系管理员"})
		return
	}
	// 按邮箱限流：IP 限流管不住同一 IP 向大量不同邮箱发码（把本站当轰炸代理），
	// 同一邮箱一分钟内只允许一条验证码
	var latest models.EmailLoginCode
	if err := a.DB.Where("email = ?", email).Order("created_at DESC").First(&latest).Error; err == nil && time.Since(latest.CreatedAt) < time.Minute {
		c.JSON(http.StatusTooManyRequests, gin.H{"code": 429, "message": "验证码发送过于频繁，请一分钟后再试"})
		return
	}

	var user models.User
	userErr := a.DB.Where("email = ?", email).First(&user).Error
	if userErr != nil && !errors.Is(userErr, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "发送验证码失败"})
		return
	}
	if userErr == nil && user.Status != "active" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "如果邮箱可用，验证码已发送，请查收邮件。"})
		return
	}

	code, err := randomEmailCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "生成验证码失败"})
		return
	}
	ttl := a.EmailCodeTTL
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	record := models.EmailLoginCode{Email: email, CodeHash: hashEmailCode(a.Secret, email, code), ExpiresAt: time.Now().Add(ttl), RequestIP: c.ClientIP()}
	if userErr == nil {
		record.UserID = &user.ID
	}
	if err := a.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("email = ? AND used_at IS NULL", email).Delete(&models.EmailLoginCode{}).Error; err != nil {
			return err
		}
		return tx.Create(&record).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "发送验证码失败"})
		return
	}
	if err := a.sendEmailCode(email, code); err != nil {
		log.Printf("email login code delivery failed: host=%s port=%d err=%v", a.SMTPHost, a.SMTPPort, err)
		a.DB.Delete(&record)
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "邮件发送失败，请稍后重试"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "验证码已发送，请查收邮件。"})
}

func (a *AuthController) VerifyEmailCode(c *gin.Context) {
	var input struct {
		Email string `json:"email" binding:"required"`
		Code  string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请输入邮箱和验证码"})
		return
	}
	email := normalizeEmail(input.Email)
	code := strings.TrimSpace(input.Code)
	if err := validateEmail(email); err != nil || !emailCodePattern.MatchString(code) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请输入有效的 6 位验证码"})
		return
	}

	record, ok := a.checkEmailCode(c, email, code)
	if !ok {
		return
	}

	var user models.User
	err := a.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		result := tx.Model(&models.EmailLoginCode{}).Where("id = ? AND used_at IS NULL", record.ID).Update("used_at", now)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errCodeConsumed
		}
		if err := tx.Where("email = ?", email).First(&user).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			if !models.BoolSetting(tx, "open_registration", true) {
				return errRegistrationClosed
			}
			created, createErr := createEmailCodeUser(tx, email)
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
		} else if errors.Is(err, errCodeConsumed) {
			status, message = http.StatusUnauthorized, "验证码无效或已使用"
		}
		c.JSON(status, gin.H{"code": status, "message": message})
		return
	}
	a.issueSession(c, user)
}

// checkEmailCode 校验邮箱验证码（未使用、未过期、5 次尝试上限、HMAC 比对），
// 失败时已写入错误响应；成功返回待消费的记录，由调用方在事务里标记 used_at
func (a *AuthController) checkEmailCode(c *gin.Context, email, code string) (models.EmailLoginCode, bool) {
	var record models.EmailLoginCode
	if err := a.DB.Where("email = ? AND used_at IS NULL", email).Order("created_at DESC").First(&record).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "验证码无效或已使用"})
		return record, false
	}
	if time.Now().After(record.ExpiresAt) {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "验证码已过期，请重新获取"})
		return record, false
	}
	if record.Attempts >= 5 {
		c.JSON(http.StatusTooManyRequests, gin.H{"code": 429, "message": "验证码错误次数过多，请重新获取"})
		return record, false
	}
	if !hmac.Equal([]byte(record.CodeHash), []byte(hashEmailCode(a.Secret, email, code))) {
		updates := map[string]any{"attempts": gorm.Expr("attempts + 1")}
		if record.Attempts+1 >= 5 {
			now := time.Now()
			updates["used_at"] = &now
		}
		_ = a.DB.Model(&models.EmailLoginCode{}).Where("id = ? AND used_at IS NULL", record.ID).Updates(updates).Error
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "验证码错误"})
		return record, false
	}
	return record, true
}

// ResetPassword 忘记密码：邮箱验证码校验通过后重置密码。
// 验证码一次性消费；成功后递增会话版本号，该账号所有旧会话立即失效
func (a *AuthController) ResetPassword(c *gin.Context) {
	var input struct {
		Email       string `json:"email" binding:"required"`
		Code        string `json:"code" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请输入邮箱、验证码和新密码"})
		return
	}
	email := normalizeEmail(input.Email)
	code := strings.TrimSpace(input.Code)
	if err := validateEmail(email); err != nil || !emailCodePattern.MatchString(code) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请输入有效的 6 位验证码"})
		return
	}
	if len(input.NewPassword) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "密码至少需要 8 个字符"})
		return
	}
	record, ok := a.checkEmailCode(c, email, code)
	if !ok {
		return
	}
	var user models.User
	if err := a.DB.Where("email = ?", email).First(&user).Error; err != nil {
		// 验证码已证明持有邮箱，此时才告知账号不存在，不构成账号枚举
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "该邮箱尚未绑定账号，请直接注册"})
		return
	}
	if user.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "账号已被限制登录"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "重置密码失败"})
		return
	}
	err = a.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.EmailLoginCode{}).Where("id = ? AND used_at IS NULL", record.ID).Update("used_at", time.Now())
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errCodeConsumed
		}
		return tx.Model(&user).Updates(map[string]any{"password_hash": string(hash), "session_version": gorm.Expr("session_version + 1")}).Error
	})
	if err != nil {
		status := http.StatusInternalServerError
		message := "重置密码失败"
		if errors.Is(err, errCodeConsumed) {
			status, message = http.StatusUnauthorized, "验证码无效或已使用"
		}
		c.JSON(status, gin.H{"code": status, "message": message})
		return
	}
	// 邮箱归属已验证，直接签发会话免二次登录
	a.issueSession(c, user)
}

func createEmailCodeUser(db *gorm.DB, email string) (models.User, error) {
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
	randomPassword := make([]byte, 32)
	if _, err := rand.Read(randomPassword); err != nil {
		return models.User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(base64.RawURLEncoding.EncodeToString(randomPassword)), bcrypt.DefaultCost)
	if err != nil {
		return models.User{}, err
	}
	user := models.User{Username: username, Email: email, PasswordHash: string(hash), Nickname: username, Role: "user", Status: "active", SessionVersion: 1}
	if err := db.Create(&user).Error; err != nil {
		return models.User{}, err
	}
	return user, nil
}

func (a *AuthController) sendEmailCode(to, code string) error {
	if a.SendEmailCode != nil {
		return a.SendEmailCode(to, code)
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
	body := fmt.Sprintf("你好，\n\n你的 Quiet Signal 登录验证码是：%s\n\n验证码 15 分钟内有效，最多可尝试 5 次。\n如果这不是你的操作，请忽略此邮件。\n", code)
	message := buildEmailMessage(from, to, "Quiet Signal 登录验证码", body)
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
	defer func() { _ = client.Close() }()
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
