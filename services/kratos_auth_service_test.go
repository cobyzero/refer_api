package services

import (
	"testing"

	"refer_api/auth"
	"refer_api/models"
)

// Reuses mockUserRepo from auth_user_service_test.go (same package).

func testKratosCfg() auth.Config {
	return auth.Config{
		PublicURL:      "http://127.0.0.1:1",
		AdminURL:       "http://127.0.0.1:1",
		JWTSecret:      "test",
		JWTExpSec:      3600,
		RefreshTTLDays: 30,
	}
}

func TestKratosAuthService_ImplementsSessionInterface(t *testing.T) {
	repo := &mockUserRepo{}
	cfg := testKratosCfg()
	svc := NewKratosAuthService(repo, auth.NewClient(cfg), cfg)
	var _ SessionAuthService = svc
	var _ AuthService = svc
}

func TestKratosAuthService_Register_EmailExists_NoNetwork(t *testing.T) {
	// When the email already exists locally, Register must fail BEFORE any
	// Kratos network call (no orphan identities possible).
	repo := &mockUserRepo{user: &models.User{Email: "exists@example.com"}}
	cfg := testKratosCfg()
	svc := NewKratosAuthService(repo, auth.NewClient(cfg), cfg)
	_, _, err := svc.Register("John", "exists@example.com", "password123", false)
	if err == nil || err.Error() != "email already exists" {
		t.Fatalf("expected 'email already exists', got: %v", err)
	}
}

func TestKratosAuthService_Refresh_EmptyToken(t *testing.T) {
	repo := &mockUserRepo{}
	cfg := testKratosCfg()
	svc := NewKratosAuthService(repo, auth.NewClient(cfg), cfg)
	if _, err := svc.RefreshSession(""); err == nil {
		t.Fatal("expected error for empty refresh token, got nil")
	}
}

func TestKratosAuthService_Logout_EmptyToken(t *testing.T) {
	repo := &mockUserRepo{}
	cfg := testKratosCfg()
	svc := NewKratosAuthService(repo, auth.NewClient(cfg), cfg)
	if err := svc.Logout(""); err != nil {
		t.Fatalf("logout with empty token should be nil, got: %v", err)
	}
}

func TestKratosConfig_Validate(t *testing.T) {
	c := auth.Config{}
	if err := c.Validate(); err == nil {
		t.Error("empty config should not validate")
	}
	c = auth.Config{PublicURL: "http://kratos:4433", AdminURL: "http://kratos:4434", JWTSecret: "s"}
	if err := c.Validate(); err != nil {
		t.Errorf("full config should validate: %v", err)
	}
}
