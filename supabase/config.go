package supabase

import (
	"os"
	"strings"
)

// Config holds the Supabase Auth configuration used by the Go backend.
// All secrets live server-side only. Never expose SERVICE_ROLE_KEY to Flutter.
type Config struct {
	URL            string
	AnonKey        string
	ServiceRoleKey string
	JWTSecret      string
}

// LoadConfig reads Supabase configuration from environment variables.
func LoadConfig() Config {
	return Config{
		URL:            strings.TrimRight(os.Getenv("SUPABASE_URL"), "/"),
		AnonKey:        os.Getenv("SUPABASE_ANON_KEY"),
		ServiceRoleKey: os.Getenv("SUPABASE_SERVICE_ROLE_KEY"),
		JWTSecret:      os.Getenv("SUPABASE_JWT_SECRET"),
	}
}

// Enabled reports whether Supabase Auth is configured.
// When false, the backend keeps using the legacy auth flow (zero-risk fallback).
func (c Config) Enabled() bool {
	return c.URL != "" && c.AnonKey != ""
}

// HasAdminKey reports whether admin operations (create/delete user, update password)
// are available via the service role key.
func (c Config) HasAdminKey() bool {
	return c.ServiceRoleKey != ""
}

// HasJWTSecret reports whether Supabase JWTs can be validated locally.
func (c Config) HasJWTSecret() bool {
	return c.JWTSecret != ""
}
