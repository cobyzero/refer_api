package database

import (
	"os"
	"testing"
)

func withEnv(t *testing.T, kv map[string]string, fn func()) {
	t.Helper()
	saved := map[string]*string{}
	keys := []string{
		"DATABASE_URL", "DB_USERNAME", "DB_PASSWORD", "DB_HOST", "DB_PORT", "DB_NAME", "DB_SSLMODE",
	}
	for _, k := range keys {
		if v, ok := os.LookupEnv(k); ok {
			c := v
			saved[k] = &c
		} else {
			saved[k] = nil
		}
		os.Unsetenv(k)
	}
	for k, v := range kv {
		os.Setenv(k, v)
	}
	defer func() {
		for _, k := range keys {
			if saved[k] == nil {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, *saved[k])
			}
		}
	}()
	fn()
}

func TestBuildDSN_LegacyParts(t *testing.T) {
	withEnv(t, map[string]string{
		"DB_USERNAME": "postgres",
		"DB_PASSWORD": "pw",
		"DB_HOST":     "localhost",
		"DB_PORT":     "5432",
		"DB_NAME":     "refer",
		"DB_SSLMODE":  "disable",
	}, func() {
		dsn, err := buildDSN()
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		want := "postgres://postgres:pw@localhost:5432/refer?sslmode=disable"
		if dsn != want {
			t.Errorf("expected %q, got %q", want, dsn)
		}
	})
}

func TestBuildDSN_DatabaseURLWins(t *testing.T) {
	withEnv(t, map[string]string{
		"DATABASE_URL": "postgres://postgres:pw@supabase-db:5432/postgres?sslmode=disable",
		"DB_USERNAME":  "postgres",
		"DB_HOST":      "old-host",
		"DB_NAME":      "refer",
	}, func() {
		dsn, err := buildDSN()
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		want := "postgres://postgres:pw@supabase-db:5432/postgres?sslmode=disable"
		if dsn != want {
			t.Errorf("expected DATABASE_URL to win: %q, got %q", want, dsn)
		}
	})
}

func TestBuildDSN_MissingVars(t *testing.T) {
	withEnv(t, map[string]string{}, func() {
		if _, err := buildDSN(); err == nil {
			t.Fatal("expected error when vars are missing, got nil")
		}
	})
}
