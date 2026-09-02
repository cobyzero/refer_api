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
	}

	for _, m := range tables {
		if _, err := db.NewCreateTable().Model(m).IfNotExists().Exec(ctx); err != nil {
			return fmt.Errorf("failed to create table for %T: %w", m, err)
		}
	}
	log.Println("Database migration completed (tables ensured)")
	return nil
}
