package utils

import (
	"errors"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

type JWTClaims struct {
	Sub   string   `json:"sub"`
	Email string   `json:"email"`
	Roles []string `json:"roles"`
	jwt.RegisteredClaims
}

// ValidatedClaims is the result of Bearer token validation.
type ValidatedClaims struct {
	UserID string
	Email  string
	Roles  []string
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

// ValidateToken validates the request Bearer token.
// Supabase Auth is the only supported issuer.
func ValidateToken(tokenString string) (*ValidatedClaims, error) {
	claims, err := ValidateSupabaseToken(tokenString)
	if err != nil {
		return nil, err
	}
	return &ValidatedClaims{
		UserID: claims.Sub,
		Email:  claims.Email,
		Roles:  claims.Roles,
	}, nil
}
