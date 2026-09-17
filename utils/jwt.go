package utils

import (
	"errors"
	"os"
	"strconv"
	"time"

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

func jwtSecret() (string, error) {
	if s := os.Getenv("AUTH_JWT_SECRET"); s != "" {
		return s, nil
	}
	return "", errors.New("AUTH_JWT_SECRET is not configured")
}

func jwtExpSec() int {
	if v := os.Getenv("AUTH_JWT_EXP_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 3600
}

// MintToken signs a short-lived API session JWT (HS256).
// sub is the identity id (Kratos identity == public.users.id, UUID).
// Passwords/credentials are never inside the token.
func MintToken(userID, email string) (string, error) {
	secret, err := jwtSecret()
	if err != nil {
		return "", err
	}
	now := time.Now()
	claims := JWTClaims{
		Sub:   userID,
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(jwtExpSec()) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// ValidateAPIToken validates an API session JWT minted by MintToken.
func ValidateAPIToken(tokenString string) (*JWTClaims, error) {
	secret, err := jwtSecret()
	if err != nil {
		return nil, err
	}

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}

// ParseUnverifiedUserID extracts sub without verifying (logout revocation:
// the token may already be expired, we still want the owner).
func ParseUnverifiedUserID(tokenString string) string {
	parser := jwt.NewParser()
	claims := &JWTClaims{}
	if _, _, err := parser.ParseUnverified(tokenString, claims); err != nil {
		return ""
	}
	return claims.Sub
}

// ValidateToken validates the request Bearer token (API session JWT).
func ValidateToken(tokenString string) (*ValidatedClaims, error) {
	claims, err := ValidateAPIToken(tokenString)
	if err != nil {
		return nil, err
	}
	return &ValidatedClaims{
		UserID: claims.Sub,
		Email:  claims.Email,
		Roles:  claims.Roles,
	}, nil
}
