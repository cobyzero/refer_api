package utils

import (
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// signTestToken mints a Supabase-shaped JWT with the test secret.
func signTestToken(t *testing.T, secret, userID, email string, roles []string) string {
	t.Helper()
	claims := JWTClaims{
		Sub:   userID,
		Email: email,
		Roles: roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign test token: %v", err)
	}
	return token
}

func TestValidateToken_SupabaseOK(t *testing.T) {
	secret := "test-supabase-jwt-secret-0123456789"
	os.Setenv("SUPABASE_JWT_SECRET", secret)
	defer os.Unsetenv("SUPABASE_JWT_SECRET")

	token := signTestToken(t, secret, "u1", "a@a.com", []string{"USER"})

	got, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("expected valid token, got: %v", err)
	}
	if got.UserID != "u1" || got.Email != "a@a.com" {
		t.Errorf("unexpected claims: %+v", got)
	}
	if len(got.Roles) != 1 || got.Roles[0] != "USER" {
		t.Errorf("unexpected roles: %v", got.Roles)
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	os.Setenv("SUPABASE_JWT_SECRET", "correct-secret-0123456789abcdef")
	defer os.Unsetenv("SUPABASE_JWT_SECRET")

	token := signTestToken(t, "other-secret-0123456789abcdef", "u1", "a@a.com", nil)
	if _, err := ValidateToken(token); err == nil {
		t.Fatal("expected error for token signed with another secret, got nil")
	}
}

func TestValidateToken_Invalid(t *testing.T) {
	os.Setenv("SUPABASE_JWT_SECRET", "test-secret-0123456789abcdef")
	defer os.Unsetenv("SUPABASE_JWT_SECRET")

	if _, err := ValidateToken("invalid-token"); err == nil {
		t.Fatal("expected error for invalid token string, got nil")
	}
}

func TestValidateToken_MissingSecret(t *testing.T) {
	os.Unsetenv("SUPABASE_JWT_SECRET")

	if _, err := ValidateToken("anything"); err == nil {
		t.Fatal("expected error when SUPABASE_JWT_SECRET is missing, got nil")
	}
}
