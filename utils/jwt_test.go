package utils

import (
	"os"
	"testing"
)

func TestJWTGenerateAndValidate(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key-999")
	defer os.Unsetenv("JWT_SECRET")

	userID := "12345"
	email := "test@example.com"
	roles := []string{"USER", "ADMIN"}

	// 1. Generate Token
	token, err := GenerateToken(userID, email, roles)
	if err != nil {
		t.Fatalf("Expected no error during token generation, got: %v", err)
	}
	if token == "" {
		t.Fatal("Expected token string to be non-empty")
	}

	// 2. Validate Token
	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("Expected token to be valid, got validation error: %v", err)
	}

	if claims.Sub != userID {
		t.Errorf("Expected Subject %s, got %s", userID, claims.Sub)
	}
	if claims.Email != email {
		t.Errorf("Expected Email %s, got %s", email, claims.Email)
	}
	if len(claims.Roles) != 2 || claims.Roles[0] != "USER" || claims.Roles[1] != "ADMIN" {
		t.Errorf("Expected roles ['USER', 'ADMIN'], got %v", claims.Roles)
	}
}

func TestJWTValidateInvalidToken(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key-999")
	defer os.Unsetenv("JWT_SECRET")

	_, err := ValidateToken("invalid-token-string")
	if err == nil {
		t.Fatal("Expected error when validating invalid token string, got nil")
	}
}
