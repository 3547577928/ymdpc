package controllers

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"quietsignal/backend/models"
)

func TestMagicLinkCreatesUserAndIsSingleUse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newCommunityTestDB(t)
	var sentLink string
	auth := &AuthController{
		DB:           db,
		Secret:       "test-secret",
		AppBaseURL:   "http://localhost:5173",
		MagicLinkTTL: time.Minute,
		SendMagicLink: func(to, link string) error {
			if to != "writer@example.com" {
				t.Fatalf("unexpected recipient: %s", to)
			}
			sentLink = link
			return nil
		},
	}

	request := performRequest(auth.RequestMagicLink, http.MethodPost, "/api/auth/magic-link/request", `{"email":"Writer@Example.com"}`, nil)
	if request.Code != http.StatusOK {
		t.Fatalf("request magic link returned %d: %s", request.Code, request.Body.String())
	}
	parsed, err := url.Parse(sentLink)
	if err != nil {
		t.Fatal(err)
	}
	token := parsed.Query().Get("token")
	if token == "" {
		t.Fatalf("magic link did not contain token: %s", sentLink)
	}

	verify := performRequest(auth.VerifyMagicLink, http.MethodPost, "/api/auth/magic-link/verify", mustJSON(t, map[string]string{"token": token}), nil)
	if verify.Code != http.StatusOK {
		t.Fatalf("verify magic link returned %d: %s", verify.Code, verify.Body.String())
	}
	var user models.User
	if err := db.Where("email = ?", "writer@example.com").First(&user).Error; err != nil {
		t.Fatal(err)
	}
	if user.Username != "writer" {
		t.Fatalf("expected generated username writer, got %q", user.Username)
	}
	if len(verify.Result().Cookies()) == 0 || verify.Result().Cookies()[0].Name != "qs_token" {
		t.Fatal("magic link login did not issue session cookie")
	}

	secondVerify := performRequest(auth.VerifyMagicLink, http.MethodPost, "/api/auth/magic-link/verify", mustJSON(t, map[string]string{"token": token}), nil)
	if secondVerify.Code != http.StatusUnauthorized {
		t.Fatalf("reused magic link returned %d: %s", secondVerify.Code, secondVerify.Body.String())
	}
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

// TestUpdatePassword 验证修改密码：原密码校验、长度校验与密码更新
func TestUpdatePassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "-") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatal(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("oldpassword"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	user := models.User{Username: "alice", PasswordHash: string(hash), Nickname: "Alice", Role: "user", Status: "active"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	auth := &AuthController{DB: db}

	// 原密码错误应被拒绝
	wrong := authenticatedRequest(auth.UpdatePassword, http.MethodPatch, "/api/me/password", `{"currentPassword":"wrongpass","newPassword":"newpassword"}`, nil, user.ID, "user")
	if wrong.Code != http.StatusUnauthorized {
		t.Fatalf("wrong current password returned %d: %s", wrong.Code, wrong.Body.String())
	}
	// 新密码过短应被拒绝
	short := authenticatedRequest(auth.UpdatePassword, http.MethodPatch, "/api/me/password", `{"currentPassword":"oldpassword","newPassword":"short"}`, nil, user.ID, "user")
	if short.Code != http.StatusBadRequest {
		t.Fatalf("short password returned %d: %s", short.Code, short.Body.String())
	}
	// 正确修改
	updated := authenticatedRequest(auth.UpdatePassword, http.MethodPatch, "/api/me/password", `{"currentPassword":"oldpassword","newPassword":"newpassword"}`, nil, user.ID, "user")
	if updated.Code != http.StatusOK {
		t.Fatalf("update password returned %d: %s", updated.Code, updated.Body.String())
	}
	var stored models.User
	if err := db.First(&stored, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte("newpassword")) != nil {
		t.Fatal("password was not updated to the new value")
	}
}

func TestLogoutBumpsSessionVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "-") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatal(err)
	}
	user := models.User{Username: "alice", PasswordHash: "hash", Nickname: "Alice", Role: "user", Status: "active", SessionVersion: 3}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}

	auth := &AuthController{DB: db, Secret: "test-secret"}
	response := authenticatedRequest(auth.Logout, http.MethodPost, "/api/auth/logout", "", nil, user.ID, "user")
	if response.Code != http.StatusOK {
		t.Fatalf("logout returned %d: %s", response.Code, response.Body.String())
	}
	var stored models.User
	if err := db.First(&stored, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.SessionVersion != 4 {
		t.Fatalf("expected session version 4 after logout, got %d", stored.SessionVersion)
	}
}
