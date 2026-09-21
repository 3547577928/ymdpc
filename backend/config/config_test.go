package config

import "testing"

func TestValidateProduction(t *testing.T) {
	t.Run("rejects defaults", func(t *testing.T) {
		cfg := Config{JWTSecret: defaultJWTSecret, AdminPassword: defaultAdminPassword}
		if err := cfg.ValidateProduction(); err == nil {
			t.Fatal("expected default credentials to be rejected")
		}
	})

	t.Run("accepts strong credentials", func(t *testing.T) {
		cfg := Config{JWTSecret: "0123456789abcdef0123456789abcdef", AdminPassword: "a-strong-password"}
		if err := cfg.ValidateProduction(); err != nil {
			t.Fatalf("expected valid production config: %v", err)
		}
	})
}
