package utils

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTClaims struct {
	Sub   string   `json:"sub"`
	Email string   `json:"email"`
	Roles []string `json:"roles"`
	jwt.RegisteredClaims
}

// GenerateToken generates an access token for a given user
func GenerateToken(userID string, email string, roles []string) (string, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "super-secret-key-12345"
	}

	claims := JWTClaims{
		Sub:   userID,
		Email: email,
		Roles: roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // 1 day
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

// ValidateToken validates a legacy JWT token and returns its claims
func ValidateToken(tokenString string) (*JWTClaims, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "super-secret-key-12345"
	}

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// AuthProvider identifies which issuer produced a validated token.
type AuthProvider string

const (
	AuthProviderLegacy   AuthProvider = "legacy"
	AuthProviderSupabase AuthProvider = "supabase"
)

// ValidatedClaims is the result of dual JWT validation.
type ValidatedClaims struct {
	UserID   string
	Email    string
	Roles    []string
	Provider AuthProvider
}

// ValidateSupabaseToken validates a Supabase Auth JWT (HS256, SUPABASE_JWT_SECRET).
// Supabase self-hosted signs with the JWT secret; sub is auth.users.id (UUID).
func ValidateSupabaseToken(tokenString string) (*JWTClaims, error) {
	jwtSecret := os.Getenv("SUPABASE_JWT_SECRET")
	if jwtSecret == "" {
		return nil, errors.New("SUPABASE_JWT_SECRET is not configured")
	}

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(jwtSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}

// ValidateAnyToken tries Supabase first (when configured) and falls back to
// the legacy JWT_SECRET. This enables a non-destructive migration: old Flutter
// sessions keep working while new logins return Supabase JWTs.
func ValidateAnyToken(tokenString string) (*ValidatedClaims, error) {
	if os.Getenv("SUPABASE_JWT_SECRET") != "" {
		if claims, err := ValidateSupabaseToken(tokenString); err == nil {
			return &ValidatedClaims{
				UserID:   claims.Sub,
				Email:    claims.Email,
				Roles:    claims.Roles,
				Provider: AuthProviderSupabase,
			}, nil
		}
	}
	claims, err := ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}
	return &ValidatedClaims{
		UserID:   claims.Sub,
		Email:    claims.Email,
		Roles:    claims.Roles,
		Provider: AuthProviderLegacy,
	}, nil
}
