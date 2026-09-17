package database

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

// CheckSupabaseDB verifica solo-lectura que la PG externa sea supabase/postgres
// (roles + schemas auth/storage). No crea nada: si falta, devuelve error
// explícito para log.Fatalf en main (evita PGRST000/3F000 crípticos después).
func CheckSupabaseDB(db *bun.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var missingRoles []string
	for _, role := range []string{"postgres", "authenticator", "supabase_admin", "anon"} {
		var count int
		if err := db.NewRaw("SELECT count(*) FROM pg_roles WHERE rolname = ?", role).Scan(ctx, &count); err != nil {
			return fmt.Errorf("check role %q: %w", role, err)
		}
		if count == 0 {
			missingRoles = append(missingRoles, role)
		}
	}
	if len(missingRoles) > 0 {
		return fmt.Errorf("PG externa sin roles %v — ¿es supabase/postgres o postgres pelado? ¿password mismatch? Revisa SUPABASE_POSTGRES_PASSWORD idéntico en refer-db y refer-api", missingRoles)
	}

	var schemas []string
	if err := db.NewRaw("SELECT schema_name FROM information_schema.schemata WHERE schema_name IN ('auth','storage','public')").Scan(ctx, &schemas); err != nil {
		return fmt.Errorf("check schemas auth/storage/public: %w", err)
	}
	has := map[string]bool{}
	for _, s := range schemas {
		has[s] = true
	}
	if !has["auth"] || !has["storage"] {
		return fmt.Errorf("PG externa sin schemas auth/storage (hay=%v) — la instancia no es supabase/postgres inicializada o el volumen quedó a medias; no se crea nada desde Go", schemas)
	}

	// auth.users debe existir (vacía ok en DB nueva)
	var authUsers int
	if err := db.NewRaw("SELECT count(*) FROM auth.users").Scan(ctx, &authUsers); err != nil {
		return fmt.Errorf("schema auth existe pero auth.users inaccesible (¿migrate.sh incompleto? search_path roto?): %w", err)
	}
	return nil
}
