package database

import (
	"context"
	"fmt"
	"log"
	"refer_api/models"

	"github.com/uptrace/bun"
)

// Migrate creates all tables if they do not exist.
// It respects FK ordering and uses IF NOT EXISTS.
func Migrate(db *bun.DB) error {
	ctx := context.Background()
	tables := []interface{}{
		(*models.Role)(nil),
		(*models.Location)(nil),
		(*models.ProductCategory)(nil),
		(*models.ProductSize)(nil),
		(*models.User)(nil),
		(*models.Product)(nil),
		(*models.ProductAvailableSize)(nil),
		(*models.ProductType)(nil),
		(*models.ProductEnhancement)(nil),
		(*models.SeasonalProduct)(nil),
		(*models.Reward)(nil),
		(*models.StarTransaction)(nil),
		(*models.RedeemedReward)(nil),
		(*models.UserRole)(nil),
		(*models.CartItem)(nil),
		(*models.CartItemEnhancement)(nil),
		(*models.Order)(nil),
		(*models.OrderItem)(nil),
		(*models.AppSetting)(nil),
		(*models.Collection)(nil),
		(*models.AuditLog)(nil),
	}

	for _, m := range tables {
		if _, err := db.NewCreateTable().Model(m).IfNotExists().Exec(ctx); err != nil {
			return fmt.Errorf("failed to create table for %T: %w", m, err)
		}
	}
	// IfNotExists never adds columns to existing tables — evolve tables explicitly.
	if _, err := db.ExecContext(ctx, `ALTER TABLE "seasonal_products" ADD COLUMN IF NOT EXISTS "collectionId" char(36)`); err != nil {
		return fmt.Errorf("failed to add collectionId to seasonal_products: %w", err)
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE "product_categories" ADD COLUMN IF NOT EXISTS "description" text NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("failed to add description to product_categories: %w", err)
	}
	// Order-first flow (Fix #2): link Stripe intents + paid timestamp.
	// Nullable, so existing rows are untouched.
	if _, err := db.ExecContext(ctx, `ALTER TABLE "orders" ADD COLUMN IF NOT EXISTS "paymentIntentId" text`); err != nil {
		return fmt.Errorf("failed to add paymentIntentId to orders: %w", err)
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE "orders" ADD COLUMN IF NOT EXISTS "paidAt" timestamptz`); err != nil {
		return fmt.Errorf("failed to add paidAt to orders: %w", err)
	}
	// Fase 0: indexes for hot paths (best-effort, never fail boot).
	ensureIndexes(ctx, db)
	log.Println("Database migration completed (tables ensured)")
	return nil
}

// ensureIndexes creates the Fase 0 hot-path indexes idempotently.
// Failures only warn (e.g. missing pg_trgm permission) so deploys never break.
func ensureIndexes(ctx context.Context, db *bun.DB) {
	stmts := []string{
		// pg_trgm for ILIKE %query% in /search (requires extension; may need superuser once).
		`CREATE EXTENSION IF NOT EXISTS pg_trgm`,
		// Catalog: filters + ordering used by FindAll/TopRated/Search.
		`CREATE INDEX IF NOT EXISTS idx_products_available_rating ON "products" ("isAvailable", "rating" DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_products_category ON "products" ("categoryId")`,
		`CREATE INDEX IF NOT EXISTS idx_products_created ON "products" ("createdAt" DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_products_name_trgm ON "products" USING gin ("name" gin_trgm_ops)`,
		// Children lookups (types/enhancements/sizes per product).
		`CREATE INDEX IF NOT EXISTS idx_product_types_product ON "product_types" ("productId")`,
		`CREATE INDEX IF NOT EXISTS idx_product_enhancements_product ON "product_enhancements" ("productId")`,
		`CREATE INDEX IF NOT EXISTS idx_product_sizes_product ON "product_available_sizes" ("productId")`,
		`CREATE INDEX IF NOT EXISTS idx_product_sizes_size ON "product_available_sizes" ("sizeId")`,
		// Auth + dashboard + audit hot paths.
		`CREATE INDEX IF NOT EXISTS idx_users_email ON "users" ("email")`,
		`CREATE INDEX IF NOT EXISTS idx_orders_user_created ON "orders" ("userId", "createdAt" DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_orders_created ON "orders" ("createdAt" DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_star_tx_user_date ON "star_transactions" ("userId", "date" DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_star_tx_type ON "star_transactions" ("type")`,
		`CREATE INDEX IF NOT EXISTS idx_audit_created ON "audit_logs" ("createdAt" DESC)`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			log.Printf("Warning: ensureIndexes: %v (stmt %.60s...)", err, s)
		}
	}
}
