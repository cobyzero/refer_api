package services

import (
	"testing"

	"refer_api/models"
	"refer_api/supabase"
)

// Reuses mockUserRepo from auth_user_service_test.go (same package).

func testSupabaseCfg() supabase.Config {
	return supabase.Config{
		URL:            "http://127.0.0.1:1",
		AnonKey:        "anon",
		ServiceRoleKey: "",
		JWTSecret:      "",
	}
}

func TestSupabaseAuthService_ImplementsSessionInterface(t *testing.T) {
	repo := &mockUserRepo{}
	cfg := testSupabaseCfg()
	svc := NewSupabaseAuthService(repo, supabase.NewClient(cfg), cfg)
	var _ SessionAuthService = svc
	var _ AuthService = svc
}

func TestSupabaseConfig_Enabled(t *testing.T) {
	c := supabase.Config{}
	if c.Enabled() {
		t.Error("empty config should not be enabled")
	}
	c = supabase.Config{URL: "http://localhost:8000", AnonKey: "anon"}
	if !c.Enabled() {
		t.Error("URL+anon should be enabled")
	}
	if c.HasAdminKey() {
		t.Error("should not have admin key")
	}
	c.ServiceRoleKey = "service"
	if !c.HasAdminKey() {
		t.Error("should have admin key")
	}
}

func TestSupabaseAuthService_Register_EmailExists_NoNetwork(t *testing.T) {
	// When the email already exists locally, Register must fail BEFORE any
	// Supabase network call (no orphan auth.users possible).
	repo := &mockUserRepo{user: &models.User{Email: "exists@example.com"}}
	cfg := testSupabaseCfg()
	svc := NewSupabaseAuthService(repo, supabase.NewClient(cfg), cfg)
	_, _, err := svc.Register("John", "exists@example.com", "password123", false)
	if err == nil || err.Error() != "email already exists" {
		t.Fatalf("expected 'email already exists', got: %v", err)
	}
}

func TestSupabaseAuthService_Refresh_EmptyToken(t *testing.T) {
	repo := &mockUserRepo{}
	cfg := testSupabaseCfg()
	svc := NewSupabaseAuthService(repo, supabase.NewClient(cfg), cfg)
	if _, err := svc.RefreshSession(""); err == nil {
		t.Fatal("expected error for empty refresh token")
	}
}

func TestSupabaseAuthService_Logout_EmptyToken(t *testing.T) {
	repo := &mockUserRepo{}
	cfg := testSupabaseCfg()
	svc := NewSupabaseAuthService(repo, supabase.NewClient(cfg), cfg)
	if err := svc.Logout(""); err != nil {
		t.Fatalf("logout with empty token should be nil, got: %v", err)
	}
}
