package utils

import (
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func supabaseTestToken(secret, sub, email string) string {
	claims := JWTClaims{
		Sub:   sub,
		Email: email,
		Roles: []string{},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(secret))
	return signed
}

func TestValidateAnyToken_LegacyFallback(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key-999")
	os.Unsetenv("SUPABASE_JWT_SECRET")
	defer os.Unsetenv("JWT_SECRET")

	token, err := GenerateToken("u1", "a@a.com", []string{"USER"})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	got, err := ValidateAnyToken(token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if got.Provider != AuthProviderLegacy {
		t.Errorf("expected legacy provider, got %s", got.Provider)
	}
	if got.UserID != "u1" {
		t.Errorf("expected sub u1, got %s", got.UserID)
	}
}

func TestValidateAnyToken_SupabaseFirst(t *testing.T) {
	os.Setenv("JWT_SECRET", "legacy-secret")
	os.Setenv("SUPABASE_JWT_SECRET", "supabase-secret")
	defer os.Unsetenv("JWT_SECRET")
	defer os.Unsetenv("SUPABASE_JWT_SECRET")

	token := supabaseTestToken("supabase-secret", "auth-uuid-123", "s@s.com")
	got, err := ValidateAnyToken(token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if got.Provider != AuthProviderSupabase {
		t.Errorf("expected supabase provider, got %s", got.Provider)
	}
	if got.UserID != "auth-uuid-123" {
		t.Errorf("expected sub auth-uuid-123, got %s", got.UserID)
	}
}

func TestValidateAnyToken_Invalid(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key-999")
	os.Unsetenv("SUPABASE_JWT_SECRET")
	defer os.Unsetenv("JWT_SECRET")

	if _, err := ValidateAnyToken("invalid-token"); err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestValidateSupabaseToken_RequiresSecret(t *testing.T) {
	os.Unsetenv("SUPABASE_JWT_SECRET")
	if _, err := ValidateSupabaseToken("whatever"); err == nil {
		t.Fatal("expected error when SUPABASE_JWT_SECRET is missing")
	}
}
