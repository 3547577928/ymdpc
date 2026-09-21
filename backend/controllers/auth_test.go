package controllers

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"quietsignal/backend/models"
)

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
