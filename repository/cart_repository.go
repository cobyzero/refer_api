package repository

import (
	"context"
	"refer_api/models"

	"github.com/uptrace/bun"
)

// CartRepository defines database operations for shopping cart items.
type CartRepository interface {
	FindAllByUser(userID string) ([]models.CartItem, error)
	Create(item *models.CartItem) error
	Delete(id, userID string) error
	UpdateQuantity(id, userID string, quantity int) error
	Clear(userID string) error
	FindByIDAndUser(id, userID string) (*models.CartItem, error)
	FindEnhancementsByIDs(ids []string) ([]models.ProductEnhancement, error)
}

type cartRepository struct {
	db *bun.DB
}

// NewCartRepository constructs a new CartRepository.
func NewCartRepository(db *bun.DB) CartRepository {
	return &cartRepository{db: db}
}

func (r *cartRepository) FindAllByUser(userID string) ([]models.CartItem, error) {
	var items []models.CartItem
	err := r.db.NewSelect().Model(&items).
		Where("?TableAlias.userId = ?", userID).
		Relation("Product").
		Relation("Product.Category").
		Relation("Product.SizeOptions").
		Relation("Size").
		Relation("Type").
		Relation("Enhancements").
		OrderExpr("?TableAlias.id ASC").
		Scan(context.Background())
	return items, err
}

func (r *cartRepository) Create(item *models.CartItem) error {
	ctx := context.Background()
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		enhancements := item.Enhancements
		item.Enhancements = nil

		if _, err := tx.NewInsert().Model(item).Exec(ctx); err != nil {
			return err
		}

		for _, enh := range enhancements {
			if _, err := tx.ExecContext(ctx,
				"INSERT IGNORE INTO cart_item_enhancements (cartItemId, enhancementId) VALUES (?, ?)",
				item.ID, enh.ID,
			); err != nil {
				return err
			}
		}

		item.Enhancements = enhancements
		return nil
	})
}

func (r *cartRepository) Delete(id, userID string) error {
	ctx := context.Background()
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.ExecContext(ctx,
			"DELETE FROM cart_item_enhancements WHERE cartItemId = ?", id,
		); err != nil {
			return err
		}
		_, err := tx.NewDelete().Model((*models.CartItem)(nil)).
			Where("id = ? AND userId = ?", id, userID).
			Exec(ctx)
		return err
	})
}

func (r *cartRepository) UpdateQuantity(id, userID string, quantity int) error {
	_, err := r.db.NewUpdate().Model((*models.CartItem)(nil)).
		Set("quantity = ?", quantity).
		Where("id = ? AND userId = ?", id, userID).
		Exec(context.Background())
	return err
}

func (r *cartRepository) Clear(userID string) error {
	ctx := context.Background()
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Get item IDs for this user
		var ids []string
		rows, err := tx.QueryContext(ctx, "SELECT id FROM cart_items WHERE userId = ?", userID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			ids = append(ids, id)
		}

		if len(ids) > 0 {
			if _, err := tx.ExecContext(ctx,
				"DELETE FROM cart_item_enhancements WHERE cartItemId IN (?)",
				bun.In(ids),
			); err != nil {
				return err
			}
		}

		_, err = tx.NewDelete().Model((*models.CartItem)(nil)).
			Where("userId = ?", userID).
			Exec(ctx)
		return err
	})
}

func (r *cartRepository) FindByIDAndUser(id, userID string) (*models.CartItem, error) {
	var item models.CartItem
	err := r.db.NewSelect().Model(&item).
		Where("id = ? AND userId = ?", id, userID).
		Scan(context.Background())
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *cartRepository) FindEnhancementsByIDs(ids []string) ([]models.ProductEnhancement, error) {
	var enhancements []models.ProductEnhancement
	if len(ids) == 0 {
		return enhancements, nil
	}
	err := r.db.NewSelect().Model(&enhancements).
		Where("id IN (?)", bun.In(ids)).
		Scan(context.Background())
	return enhancements, err
}
