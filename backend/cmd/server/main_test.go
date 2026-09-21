package main

import "testing"

func TestCorsConfigAllowsLocalDevelopmentPorts(t *testing.T) {
	config := corsConfig("http://localhost:5173", true)

	for _, origin := range []string{
		"http://localhost:5174",
		"http://127.0.0.1:5175",
		"http://[::1]:5176",
	} {
		if !config.AllowOriginFunc(origin) {
			t.Fatalf("expected local development origin %q to be allowed", origin)
		}
	}
}

func TestCorsConfigKeepsProductionAllowlistStrict(t *testing.T) {
	config := corsConfig("https://blog.example.com", false)

	if !config.AllowOriginFunc("https://blog.example.com") {
		t.Fatal("expected configured production origin to be allowed")
	}
	if config.AllowOriginFunc("http://127.0.0.1:5174") {
		t.Fatal("expected unconfigured local origin to be rejected in production")
	}
	if config.AllowOriginFunc("https://example.net") {
		t.Fatal("expected unconfigured remote origin to be rejected")
	}
}
