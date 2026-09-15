package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

var (
	DB   *bun.DB
	once sync.Once
)

// Connect initializes the Bun database connection using environment variables.
// It is safe to call multiple times; only the first call initializes the connection.
func Connect() error {
	var err error
	once.Do(func() {
		err = connect()
	})
	return err
}

// buildDSN resolves the Postgres DSN.
// Precedence: DATABASE_URL (Fase 2: Supabase-managed PostgreSQL) takes
// priority when set; otherwise the legacy DB_HOST/DB_PORT/... parts are used.
// Behavior without DATABASE_URL is byte-for-byte identical to before.
func buildDSN() (string, error) {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url, nil
	}

	dbUser := os.Getenv("DB_USERNAME")
	dbPass := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")

	// Validate required env vars
	if dbUser == "" || dbHost == "" || dbName == "" {
		return "", fmt.Errorf("database environment variables are missing (DB_USERNAME, DB_HOST, DB_NAME)")
	}

	// Default to 5432 if DB_PORT is not set
	if dbPort == "" {
		dbPort = "5432"
	}

	sslMode := os.Getenv("DB_SSLMODE")
	if sslMode == "" {
		sslMode = "disable"
	}

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		dbUser, dbPass, dbHost, dbPort, dbName, sslMode,
	), nil
}

// connect performs the actual initialization logic.
func connect() error {
	dsn, err := buildDSN()
	if err != nil {
		return err
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))

	// Configure connection pool
	maxOpenConns := getEnvInt("DB_MAX_OPEN_CONNS", 25)
	maxIdleConns := getEnvInt("DB_MAX_IDLE_CONNS", 10)
	connMaxLifetime := getEnvInt("DB_CONN_MAX_LIFETIME_SEC", 300)

	sqldb.SetMaxOpenConns(maxOpenConns)
	sqldb.SetMaxIdleConns(maxIdleConns)
	sqldb.SetConnMaxLifetime(time.Duration(connMaxLifetime) * time.Second)

	DB = bun.NewDB(sqldb, pgdialect.New())

	// Verify connectivity with a timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := DB.PingContext(ctx); err != nil {
		_ = sqldb.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Database connection successfully established (pool: max_idle=%d, max_open=%d, max_lifetime=%ds)",
		maxIdleConns, maxOpenConns, connMaxLifetime)
	return nil
}

// Close gracefully closes the database connection.
// It is safe to call multiple times; subsequent calls are a no-op.
func Close() error {
	if DB == nil {
		return nil
	}
	if err := DB.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}
	log.Println("Database connection closed")
	return nil
}

// getEnvInt retrieves an integer environment variable with a default value.
// If the variable is empty or not a valid integer, the default value is returned.
func getEnvInt(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		log.Printf("Warning: invalid integer value for %s=%q, using default %d", key, val, defaultVal)
		return defaultVal
	}
	return i
}
