package models

import (
	"testing"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestSeedRotatesAdminPassword(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:seed-test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &Post{}, &Tag{}); err != nil {
		t.Fatal(err)
	}
	if err := Seed(db, "admin", "first-password"); err != nil {
		t.Fatal(err)
	}
	if err := Seed(db, "admin", "second-password"); err != nil {
		t.Fatal(err)
	}

	var user User
	if err := db.Where("username = ?", "admin").First(&user).Error; err != nil {
		t.Fatal(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("second-password")) != nil {
		t.Fatal("expected the configured password to replace the previous hash")
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("first-password")) == nil {
		t.Fatal("previous password is still valid")
	}
	var postCount int64
	if err := db.Model(&Post{}).Count(&postCount).Error; err != nil {
		t.Fatal(err)
	}
	if postCount != 0 {
		t.Fatalf("expected no demo posts to be seeded, got %d", postCount)
	}
}
