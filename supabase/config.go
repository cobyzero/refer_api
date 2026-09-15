package supabase

import (
	"fmt"
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
// Supabase Auth is the only supported auth flow.
func (c Config) Enabled() bool {
	return c.URL != "" && c.AnonKey != ""
}

// Validate ensures every Supabase Auth setting the backend needs is present.
// The server refuses to boot without them: there is no legacy fallback.
func (c Config) Validate() error {
	var missing []string
	if c.URL == "" {
		missing = append(missing, "SUPABASE_URL")
	}
	if c.AnonKey == "" {
		missing = append(missing, "SUPABASE_ANON_KEY")
	}
	if c.JWTSecret == "" {
		missing = append(missing, "SUPABASE_JWT_SECRET")
	}
	if c.ServiceRoleKey == "" {
		missing = append(missing, "SUPABASE_SERVICE_ROLE_KEY")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required env: %s", strings.Join(missing, ", "))
	}
	return nil
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
