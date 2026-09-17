package auth

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds the Ory Kratos + API-session configuration used by the Go
// backend. Kratos owns credentials (password hashing, email OTP codes);
// Go mints short-lived API JWTs after Kratos confirms the identity.
// Secrets live server-side only.
type Config struct {
	// Kratos endpoints (server-side). In compose: http://kratos:4433/4434.
	// Local `go run`: http://localhost:4433/4434 (ports bound to 127.0.0.1).
	PublicURL string
	AdminURL  string
	// API session JWT (HS256, minted by Go, validated locally).
	JWTSecret string
	JWTExpSec int
	// Opaque refresh tokens (sha256 in PG, rotated on use).
	RefreshTTLDays int
}

// LoadConfig reads auth configuration from environment variables.
func LoadConfig() Config {
	return Config{
		PublicURL:      strings.TrimRight(firstEnv("KRATOS_PUBLIC_URL", "http://kratos:4433"), "/"),
		AdminURL:       strings.TrimRight(firstEnv("KRATOS_ADMIN_URL", "http://kratos:4434"), "/"),
		JWTSecret:      os.Getenv("AUTH_JWT_SECRET"),
		JWTExpSec:      envInt("AUTH_JWT_EXP_SEC", 3600),
		RefreshTTLDays: envInt("AUTH_REFRESH_TTL_DAYS", 30),
	}
}

func firstEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

// Enabled reports whether Kratos + API JWT signing are configured.
func (c Config) Enabled() bool {
	return c.PublicURL != "" && c.AdminURL != "" && c.JWTSecret != ""
}

// Validate ensures every setting the backend needs is present.
// The server refuses to boot without them: there is no legacy fallback.
func (c Config) Validate() error {
	var missing []string
	if c.PublicURL == "" {
		missing = append(missing, "KRATOS_PUBLIC_URL")
	}
	if c.AdminURL == "" {
		missing = append(missing, "KRATOS_ADMIN_URL")
	}
	if c.JWTSecret == "" {
		missing = append(missing, "AUTH_JWT_SECRET")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required env: %s", strings.Join(missing, ", "))
	}
	return nil
}
