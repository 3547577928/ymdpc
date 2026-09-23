package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	"quietsignal/backend/models"
)

func TestRequireAuthRejectsRevokedSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "-") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatal(err)
	}
	user := models.User{Username: "alice", PasswordHash: "hash", Nickname: "Alice", Role: "user", Status: "active", SessionVersion: 1}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  user.ID,
		"role": user.Role,
		"sv":   user.SessionVersion,
		"exp":  time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	router.GET("/private", RequireAuth("test-secret", db), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	request := func() *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/private", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		router.ServeHTTP(recorder, req)
		return recorder
	}

	if response := request(); response.Code != http.StatusNoContent {
		t.Fatalf("active session returned %d: %s", response.Code, response.Body.String())
	}
	if err := db.Model(&models.User{}).Where("id = ?", user.ID).Update("session_version", 2).Error; err != nil {
		t.Fatal(err)
	}
	if response := request(); response.Code != http.StatusUnauthorized {
		t.Fatalf("revoked session returned %d: %s", response.Code, response.Body.String())
	}
}
