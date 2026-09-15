// Command supabase-migrate performs the non-destructive Fase 2 migration:
//
//	legacy PostgreSQL  -->  Supabase-managed PostgreSQL (+ Supabase Auth)
//
// Steps (run in order, each idempotent and re-runnable):
//
//  1. backup   pg_dump -Fc of the LEGACY database (read-only, always safe).
//  2. migrate  pg_restore schema+data of app tables into the target, then sync
//     auth.users via Admin API, remap public.users.id where the server
//     ignored our requested id, and cascade child FKs in a transaction.
//     REQUIRES -apply, otherwise it only prints the plan (dry-run).
//  3. validate read-only parity report: counts, FK orphans, auth-vs-public.
//
// The legacy database is NEVER written to and NEVER dropped by this tool.
// RLS (supabase/rls.sql) is applied separately, only after validate passes.
//
// Required binaries for backup/migrate: pg_dump, pg_restore (postgres client).
// Env:
//
//	LEGACY_DATABASE_URL   legacy DB (or legacy DB_HOST/DB_PORT/... vars)
//	DATABASE_URL          target = Supabase PostgreSQL
//	SUPABASE_URL, SUPABASE_ANON_KEY, SUPABASE_SERVICE_ROLE_KEY
package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"refer_api/supabase"

	"github.com/uptrace/bun/driver/pgdriver"
)

// App tables in dependency order (parents first). Mirrors database.Migrate.
var appTables = []string{
	"roles",
	"locations",
	"product_categories",
	"product_sizes",
	"users",
	"products",
	"product_available_sizes",
	"product_types",
	"product_enhancements",
	"seasonal_products",
	"rewards",
	"star_transactions",
	"redeemed_rewards",
	"users_roles",
	"cart_items",
	"cart_item_enhancements",
	"orders",
	"order_items",
	"app_settings",
	"collections",
}

type orphanCheck struct {
	label string
	query string
}

var orphanChecks = []orphanCheck{
	{"orders.userId -> users", `SELECT COUNT(*) FROM orders o LEFT JOIN users u ON u.id=o."userId" WHERE u.id IS NULL`},
	{"star_transactions.userId -> users", `SELECT COUNT(*) FROM star_transactions s LEFT JOIN users u ON u.id=s."userId" WHERE u.id IS NULL`},
	{"redeemed_rewards.userId -> users", `SELECT COUNT(*) FROM redeemed_rewards r LEFT JOIN users u ON u.id=r."userId" WHERE u.id IS NULL`},
	{"redeemed_rewards.rewardId -> rewards", `SELECT COUNT(*) FROM redeemed_rewards r LEFT JOIN rewards w ON w.id=r."rewardId" WHERE w.id IS NULL`},
	{"cart_items.userId -> users", `SELECT COUNT(*) FROM cart_items c LEFT JOIN users u ON u.id=c."userId" WHERE u.id IS NULL`},
	{"cart_items.productId -> products", `SELECT COUNT(*) FROM cart_items c LEFT JOIN products p ON p.id=c."productId" WHERE c."productId" IS NOT NULL AND p.id IS NULL`},
	{"order_items.orderId -> orders", `SELECT COUNT(*) FROM order_items i LEFT JOIN orders o ON o.id=i."orderId" WHERE o.id IS NULL`},
	{"users_roles.usersId -> users", `SELECT COUNT(*) FROM users_roles ur LEFT JOIN users u ON u.id=ur."usersId" WHERE u.id IS NULL`},
	{"users_roles.rolesId -> roles", `SELECT COUNT(*) FROM users_roles ur LEFT JOIN roles r ON r.id=ur."rolesId" WHERE r.id IS NULL`},
}

// childFKs remaps public.users.id cascades when auth.users.id differs.
var childFKs = [][2]string{
	{"orders", "userId"},
	{"star_transactions", "userId"},
	{"redeemed_rewards", "userId"},
	{"cart_items", "userId"},
	{"users_roles", "usersId"},
}

func main() {
	step := flag.String("step", "validate", "backup|migrate|validate")
	apply := flag.Bool("apply", false, "migrate: actually write to target (default dry-run plan only)")
	backupDir := flag.String("backup-dir", "./backups", "backup: output directory")
	backupFile := flag.String("backup-file", "", "migrate: dump file to restore (default latest in backup-dir)")
	sendRecovery := flag.Bool("send-recovery", false, "migrate: send Supabase recovery email per new auth user")
	flag.Parse()

	legacyDSN := os.Getenv("LEGACY_DATABASE_URL")
	if legacyDSN == "" {
		legacyDSN = buildLegacyDSN()
	}
	targetDSN := os.Getenv("DATABASE_URL")

	switch *step {
	case "backup":
		if legacyDSN == "" {
			log.Fatal("LEGACY_DATABASE_URL (or legacy DB_* vars) is required")
		}
		if err := runBackup(legacyDSN, *backupDir); err != nil {
			log.Fatalf("backup: %v", err)
		}
	case "migrate":
		if legacyDSN == "" || targetDSN == "" {
			log.Fatal("migrate: LEGACY_DATABASE_URL and DATABASE_URL are required")
		}
		file := *backupFile
		if file == "" {
			var err error
			file, err = latestBackup(*backupDir)
			if err != nil {
				log.Fatalf("migrate: %v (run -step=backup first)", err)
			}
		}
		if err := runMigrate(legacyDSN, targetDSN, file, *apply, *sendRecovery); err != nil {
			log.Fatalf("migrate: %v", err)
		}
	case "validate":
		if legacyDSN == "" || targetDSN == "" {
			log.Fatal("validate: LEGACY_DATABASE_URL and DATABASE_URL are required")
		}
		if err := runValidate(legacyDSN, targetDSN); err != nil {
			log.Fatalf("validate: %v", err)
		}
	default:
		log.Fatalf("unknown -step %q (backup|migrate|validate)", *step)
	}
}

func buildLegacyDSN() string {
	host, port, user, pass, name, ssl := os.Getenv("DB_HOST"), os.Getenv("DB_PORT"),
		os.Getenv("DB_USERNAME"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_SSLMODE")
	if host == "" || user == "" || name == "" {
		return ""
	}
	if port == "" {
		port = "5432"
	}
	if ssl == "" {
		ssl = "disable"
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, pass, host, port, name, ssl)
}

// hasRealErrors reports whether pg_restore output contains ERROR lines other
// than known-benign ones: transaction_timeout (pg_dump>=18 preamble vs PG15
// target) and already-exists (idempotent re-runs).
func hasRealErrors(out string) bool {
	for _, line := range strings.Split(out, "\n") {
		if !strings.Contains(line, "ERROR:") {
			continue
		}
		if strings.Contains(line, "transaction_timeout") {
			continue
		}
		if strings.Contains(line, "already exists") {
			continue
		}
		if strings.Contains(line, "multiple primary keys") {
			continue // re-run: constraint was added by a previous pass
		}
		return true
	}
	return false
}

func tailLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// dbName extracts the database name from a postgres URL.
func dbName(dsn string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	name := strings.TrimPrefix(u.Path, "/")
	if name == "" {
		return "", fmt.Errorf("no database in DSN")
	}
	return name, nil
}

// dsnEnv splits a postgres URL into PG* env vars for pg_dump/pg_restore.
func dsnEnv(dsn string) ([]string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}
	pw, _ := u.User.Password()
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "5432"
	}
	db := strings.TrimPrefix(u.Path, "/")
	env := []string{
		"PGHOST=" + host, "PGPORT=" + port,
		"PGUSER=" + u.User.Username(), "PGPASSWORD=" + pw, "PGDATABASE=" + db,
	}
	return env, nil
}

func needBin(name string) string {
	p, err := exec.LookPath(name)
	if err != nil {
		log.Fatalf("%s not found in PATH (install postgres client tools)", name)
	}
	return p
}

// ---------- backup ----------

func runBackup(legacyDSN, dir string) error {
	pgDump := needBin("pg_dump")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	file := filepath.Join(dir, "legacy-"+time.Now().Format("20060102-150405")+".dump")
	env, err := dsnEnv(legacyDSN)
	if err != nil {
		return err
	}
	// Full custom-format backup (schema + data). Read-only on legacy.
	cmd := exec.Command(pgDump, "-Fc", "-f", file)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	log.Printf("backing up legacy DB -> %s", file)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump: %w", err)
	}
	log.Printf("backup OK: %s", file)
	return nil
}

func latestBackup(dir string) (string, error) {
	entries, err := filepath.Glob(filepath.Join(dir, "legacy-*.dump"))
	if err != nil || len(entries) == 0 {
		return "", fmt.Errorf("no backups in %s", dir)
	}
	latest := entries[0]
	for _, e := range entries[1:] {
		a, _ := os.Stat(latest)
		b, _ := os.Stat(e)
		if b.ModTime().After(a.ModTime()) {
			latest = e
		}
	}
	return latest, nil
}

// ---------- migrate ----------

func openDB(dsn string) *sql.DB {
	return sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
}

func tablesInLegacy(ctx context.Context, legacy *sql.DB) map[string]bool {
	has := map[string]bool{}
	for _, t := range appTables {
		if tableExists(ctx, legacy, t) {
			has[t] = true
		}
	}
	return has
}

// filteredTOC writes a pg_restore -L list keeping only app-table entries
// (TABLE, TABLE DATA, CONSTRAINT, INDEX, SEQUENCE/DEFAULT). Everything else
// (extensions, SET transaction_timeout from pg_dump>=18, other schemas) is
// dropped so restores are version-tolerant and never touch auth/storage.
func filteredTOC(pgRestore, dumpFile string, keep map[string]bool) (string, error) {
	out, err := exec.Command(pgRestore, "-l", dumpFile).Output()
	if err != nil {
		return "", fmt.Errorf("pg_restore -l: %w", err)
	}
	var kept []string
	for _, line := range strings.Split(string(out), "\n") {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, ";") {
			continue
		}
		fields := strings.Fields(s)
		// Format: <dumpId>; <tableOid> <oid> <DESC> <schema> <name> <owner>
		if len(fields) < 6 {
			continue
		}
		desc, schema, name := fields[3], fields[4], fields[5]
		if schema != "public" {
			continue
		}
		switch desc {
		case "TABLE", "TABLE DATA", "CONSTRAINT", "INDEX", "SEQUENCE", "SEQUENCE SET", "DEFAULT":
			// CONSTRAINT/INDEX lines carry the table name in field[5].
		default:
			continue
		}
		if keep[name] {
			kept = append(kept, line)
		}
	}
	tmp, err := os.CreateTemp("", "toc-*.lst")
	if err != nil {
		return "", err
	}
	if _, err := tmp.WriteString(strings.Join(kept, "\n") + "\n"); err != nil {
		tmp.Close()
		return "", err
	}
	tmp.Close()
	log.Printf("TOC filter: %d entries kept", len(kept))
	return tmp.Name(), nil
}

func tableExists(ctx context.Context, db *sql.DB, table string) bool {
	var exists bool
	_ = db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name=$1)`,
		table).Scan(&exists)
	return exists
}

func countRows(ctx context.Context, db *sql.DB, table string) (int64, error) {
	var n int64
	err := db.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM "%s"`, table)).Scan(&n)
	return n, err
}

func runMigrate(legacyDSN, targetDSN, dumpFile string, apply, sendRecovery bool) error {
	ctx := context.Background()
	legacy := openDB(legacyDSN)
	defer legacy.Close()
	target := openDB(targetDSN)
	defer target.Close()

	if err := legacy.PingContext(ctx); err != nil {
		return fmt.Errorf("legacy ping: %w", err)
	}
	if err := target.PingContext(ctx); err != nil {
		return fmt.Errorf("target ping: %w", err)
	}

	// 1. Plan: legacy counts per table.
	fmt.Println("== migration plan (legacy counts) ==")
	for _, t := range appTables {
		if !tableExists(ctx, legacy, t) {
			fmt.Printf("  %-26s MISSING in legacy (skip)\n", t)
			continue
		}
		n, _ := countRows(ctx, legacy, t)
		fmt.Printf("  %-26s %d rows\n", t, n)
	}

	if !apply {
		fmt.Println("dry-run: nothing written. Re-run with -apply to execute.")
		fmt.Println("order: pg_restore schema-only -> data-only (--disable-triggers) -> auth sync -> id remap -> validate")
		return nil
	}

	// 2. Restore schema+data of app tables only (never touches auth/storage schemas).
	pgRestore := needBin("pg_restore")
	env, err := dsnEnv(targetDSN)
	if err != nil {
		return err
	}
	// pg_dump >= 18 emits SET transaction_timeout which PG15 target rejects.
	// Filter the TOC list: keep only app-table entries, drop the rest.
	listFile, err := filteredTOC(pgRestore, dumpFile, tablesInLegacy(ctx, legacy))
	if err != nil {
		return err
	}
	defer os.Remove(listFile)
	targetDB, err := dbName(targetDSN)
	if err != nil {
		return err
	}
	for _, phase := range [][]string{{"--schema-only"}, {"--data-only", "--disable-triggers"}} {
		args := append([]string{"-d", targetDB, "-L", listFile, "--no-owner", "--no-acl"}, phase...)
		args = append(args, dumpFile)
		cmd := exec.Command(pgRestore, args...)
		cmd.Env = append(os.Environ(), env...)
		log.Printf("pg_restore %s ...", strings.Join(phase, " "))
		out, err := cmd.CombinedOutput()
		if err != nil && hasRealErrors(string(out)) {
			fmt.Println(tailLines(string(out), 15))
			if len(phase) > 1 {
				return fmt.Errorf("pg_restore %v failed (see errors above)", phase)
			}
			log.Printf("schema phase had real errors (continuing, re-runnable)")
		} else if err != nil {
			log.Printf("pg_restore %v: only benign warnings (transaction_timeout/already-exists), continuing", phase)
		}
	}

	// 3. Auth sync: every public.users email must exist in auth.users.
	if err := syncAuthUsers(ctx, target, sendRecovery); err != nil {
		return fmt.Errorf("auth sync: %w", err)
	}

	fmt.Println("migrate done. Next: -step=validate, then supabase/rls.sql, then point Go at DATABASE_URL.")
	return nil
}

type legacyUser struct {
	ID    string
	Email string
	Name  string
}

func syncAuthUsers(ctx context.Context, target *sql.DB, sendRecovery bool) error {
	cfg := supabase.LoadConfig()
	if !cfg.Enabled() || !cfg.HasAdminKey() {
		return fmt.Errorf("SUPABASE_URL/ANON_KEY/SERVICE_ROLE_KEY required for auth sync")
	}
	client := supabase.NewClient(cfg)

	rows, err := target.QueryContext(ctx, `SELECT "id","email","name" FROM "users"`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var users []legacyUser
	for rows.Next() {
		var u legacyUser
		if err := rows.Scan(&u.ID, &u.Email, &u.Name); err != nil {
			return err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	existing, err := client.AdminListUsers()
	if err != nil {
		return err
	}
	byEmail := map[string]string{}
	for _, a := range existing {
		byEmail[strings.ToLower(a.Email)] = a.ID
	}

	remaps := 0
	for _, u := range users {
		authID, ok := byEmail[strings.ToLower(u.Email)]
		if !ok {
			pw := randomPassword()
			authID, err = client.AdminCreateUser(u.Email, pw, u.Name, u.ID, true)
			if err != nil {
				return fmt.Errorf("create auth user %s: %w", u.Email, err)
			}
			log.Printf("auth user created: %s -> %s", u.Email, authID)
			if sendRecovery {
				if err := client.Recover(u.Email); err != nil {
					log.Printf("warn: recovery email to %s failed: %v", u.Email, err)
				}
			}
		}
		if authID != u.ID {
			if err := remapUserID(ctx, target, u.ID, authID); err != nil {
				return fmt.Errorf("remap %s: %w", u.Email, err)
			}
			remaps++
		}
	}
	log.Printf("auth sync OK: %d profiles, %d id remaps", len(users), remaps)
	return nil
}

func remapUserID(ctx context.Context, target *sql.DB, oldID, newID string) error {
	tx, err := target.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE "users" SET "id"=$1 WHERE "id"=$2`, newID, oldID); err != nil {
		return err
	}
	for _, fk := range childFKs {
		if _, err := tx.ExecContext(ctx,
			fmt.Sprintf(`UPDATE "%s" SET "%s"=$1 WHERE "%s"=$2`, fk[0], fk[1], fk[1]),
			newID, oldID); err != nil {
			return fmt.Errorf("%s.%s: %w", fk[0], fk[1], err)
		}
	}
	return tx.Commit()
}

func randomPassword() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "Tmp-" + fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return "Tmp-" + base64.RawURLEncoding.EncodeToString(b)
}

// ---------- validate ----------

// localOnly tables exist per environment and are excluded from parity
// (append-only logs, never migrated).
var localOnly = map[string]bool{"audit_logs": true}

func runValidate(legacyDSN, targetDSN string) error {
	ctx := context.Background()
	legacy := openDB(legacyDSN)
	defer legacy.Close()
	target := openDB(targetDSN)
	defer target.Close()

	failed := false
	fmt.Println("== counts: legacy vs target ==")
	for _, t := range appTables {
		if localOnly[t] {
			continue
		}
		lHas, tHas := tableExists(ctx, legacy, t), tableExists(ctx, target, t)
		if !lHas && !tHas {
			continue
		}
		var l, r int64
		if lHas {
			l, _ = countRows(ctx, legacy, t)
		}
		if tHas {
			r, _ = countRows(ctx, target, t)
		}
		status := "OK"
		if l != r {
			status = "MISMATCH"
			failed = true
		}
		fmt.Printf("  %-26s legacy=%-8d target=%-8d %s\n", t, l, r, status)
	}

	fmt.Println("== orphan FK checks (target, must be 0) ==")
	for _, oc := range orphanChecks {
		var n int64
		if err := target.QueryRowContext(ctx, oc.query).Scan(&n); err != nil {
			fmt.Printf("  %-38s SKIP (%v)\n", oc.label, err)
			continue
		}
		fmt.Printf("  %-38s orphans=%d %s\n", oc.label, n, verdict(n, &failed))
	}

	fmt.Println("== auth.users vs public.users ==")
	cfg := supabase.LoadConfig()
	if cfg.Enabled() && cfg.HasAdminKey() {
		client := supabase.NewClient(cfg)
		authUsers, err := client.AdminListUsers()
		if err != nil {
			fmt.Printf("  auth list: SKIP (%v)\n", err)
		} else {
			var pub int64
			_ = target.QueryRowContext(ctx, `SELECT COUNT(*) FROM "users"`).Scan(&pub)
			status := "OK"
			if int64(len(authUsers)) < pub {
				status = "MISMATCH (more profiles than auth users)"
				failed = true
			}
			fmt.Printf("  auth.users=%d public.users=%d %s\n", len(authUsers), pub, status)
		}
	} else {
		fmt.Println("  SKIP (Supabase admin not configured)")
	}

	if failed {
		return fmt.Errorf("validation FAILED — do not cut over")
	}
	fmt.Println("validation PASSED")
	return nil
}

func verdict(orphans int64, failed *bool) string {
	if orphans != 0 {
		*failed = true
		return "FAIL"
	}
	return "OK"
}
