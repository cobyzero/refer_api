package utils

import (
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// signTestToken mints an API-shaped JWT with the test secret.
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

func TestValidateToken_APIJWT(t *testing.T) {
	secret := "test-auth-jwt-secret-0123456789abcdef"
	os.Setenv("AUTH_JWT_SECRET", secret)
	defer os.Unsetenv("AUTH_JWT_SECRET")

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

func TestMintToken_RoundTrip(t *testing.T) {
	os.Setenv("AUTH_JWT_SECRET", "test-mint-secret-0123456789abcdef")
	os.Setenv("AUTH_JWT_EXP_SEC", "3600")
	defer os.Unsetenv("AUTH_JWT_SECRET")
	defer os.Unsetenv("AUTH_JWT_EXP_SEC")

	token, err := MintToken("u9", "b@b.com")
	if err != nil {
		t.Fatalf("mint failed: %v", err)
	}
	got, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("minted token should validate: %v", err)
	}
	if got.UserID != "u9" || got.Email != "b@b.com" {
		t.Errorf("unexpected claims: %+v", got)
	}
}

func TestParseUnverifiedUserID_Expired(t *testing.T) {
	secret := "test-unverified-secret-0123456789ab"
	claims := JWTClaims{
		Sub:   "u-exp",
		Email: "e@e.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateToken(token); err == nil {
		t.Fatal("expired token should not validate")
	}
	if got := ParseUnverifiedUserID(token); got != "u-exp" {
		t.Errorf("expected u-exp, got %q", got)
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	os.Setenv("AUTH_JWT_SECRET", "correct-secret-0123456789abcdef")
	defer os.Unsetenv("AUTH_JWT_SECRET")

	token := signTestToken(t, "other-secret-0123456789abcdef", "u1", "a@a.com", nil)
	if _, err := ValidateToken(token); err == nil {
		t.Fatal("expected error for token signed with another secret, got nil")
	}
}

func TestValidateToken_Invalid(t *testing.T) {
	os.Setenv("AUTH_JWT_SECRET", "test-secret-0123456789abcdef")
	defer os.Unsetenv("AUTH_JWT_SECRET")

	if _, err := ValidateToken("invalid-token"); err == nil {
		t.Fatal("expected error for invalid token string, got nil")
	}
}

func TestValidateToken_MissingSecret(t *testing.T) {
	os.Unsetenv("AUTH_JWT_SECRET")

	if _, err := ValidateToken("anything"); err == nil {
		t.Fatal("expected error when AUTH_JWT_SECRET is missing, got nil")
	}
}
