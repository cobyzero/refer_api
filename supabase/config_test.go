package supabase

import (
	"testing"
)

func TestConfig_Validate_RequiresAll(t *testing.T) {
	full := Config{
		URL:            "http://localhost:8000",
		AnonKey:        "anon",
		ServiceRoleKey: "service",
		JWTSecret:      "secret-with-at-least-32-characters-long",
	}
	if err := full.Validate(); err != nil {
		t.Fatalf("expected full config to validate, got: %v", err)
	}

	cases := map[string]func(*Config){
		"SUPABASE_URL":              func(c *Config) { c.URL = "" },
		"SUPABASE_ANON_KEY":         func(c *Config) { c.AnonKey = "" },
		"SUPABASE_SERVICE_ROLE_KEY": func(c *Config) { c.ServiceRoleKey = "" },
		"SUPABASE_JWT_SECRET":       func(c *Config) { c.JWTSecret = "" },
	}
	for name, clear := range cases {
		cfg := full
		clear(&cfg)
		if err := cfg.Validate(); err == nil {
			t.Errorf("expected error when %s is missing, got nil", name)
		}
	}
}
