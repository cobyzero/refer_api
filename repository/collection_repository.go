package repository

import (
	"context"

	"refer_api/models"

	"github.com/uptrace/bun"
)

// CollectionRepository defines persistence for featured collections and their items.
type CollectionRepository interface {
	FindAll() ([]models.Collection, error)
	FindByID(id string) (*models.Collection, error)
	Create(collection *models.Collection) error
	Update(collection *models.Collection) error
	Delete(id string) error
	FindItem(collectionID, productID string) (*models.SeasonalProduct, error)
	AddItem(item *models.SeasonalProduct) error
	RemoveItem(collectionID, productID string) error
	RemoveAllItems(collectionID string) error
}

type collectionRepository struct {
	db *bun.DB
}

// NewCollectionRepository constructs a new CollectionRepository.
func NewCollectionRepository(db *bun.DB) CollectionRepository {
	return &collectionRepository{db: db}
}

func withCollectionRelations(q *bun.SelectQuery) *bun.SelectQuery {
	return q.
		Relation("Items").
		Relation("Items.Product").
		Relation("Items.Product.Category").
		Relation("Items.Product.SizeOptions").
		Relation("Items.Product.SizeOptions.Size").
		Relation("Items.Product.Options").
		Relation("Items.Product.Enhancements")
}

func (r *collectionRepository) FindAll() ([]models.Collection, error) {
	collections := []models.Collection{}
	err := withCollectionRelations(r.db.NewSelect().Model(&collections)).
		OrderExpr("\"collection\".\"createdAt\" DESC").
		Scan(context.Background())
	return collections, err
}

func (r *collectionRepository) FindByID(id string) (*models.Collection, error) {
	var collection models.Collection
	err := withCollectionRelations(r.db.NewSelect().Model(&collection)).
		Where("\"collection\".\"id\" = ?", id).
		Scan(context.Background())
	if err != nil {
		return nil, err
	}
	return &collection, nil
}

func (r *collectionRepository) Create(collection *models.Collection) error {
	_, err := r.db.NewInsert().Model(collection).Exec(context.Background())
	return err
}

func (r *collectionRepository) Update(collection *models.Collection) error {
	_, err := r.db.NewUpdate().Model(collection).WherePK().Exec(context.Background())
	return err
}

func (r *collectionRepository) Delete(id string) error {
	_, err := r.db.NewDelete().Model((*models.Collection)(nil)).Where("id = ?", id).Exec(context.Background())
	return err
}

func (r *collectionRepository) FindItem(collectionID, productID string) (*models.SeasonalProduct, error) {
	var item models.SeasonalProduct
	err := r.db.NewSelect().Model(&item).
		Where("\"seasonal_product\".\"collectionId\" = ?", collectionID).
		Where("\"seasonal_product\".\"productId\" = ?", productID).
		Scan(context.Background())
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *collectionRepository) AddItem(item *models.SeasonalProduct) error {
	_, err := r.db.NewInsert().Model(item).Exec(context.Background())
	return err
}

func (r *collectionRepository) RemoveItem(collectionID, productID string) error {
	_, err := r.db.NewDelete().Model((*models.SeasonalProduct)(nil)).
		Where("\"collectionId\" = ?", collectionID).
		Where("\"productId\" = ?", productID).
		Exec(context.Background())
	return err
}

func (r *collectionRepository) RemoveAllItems(collectionID string) error {
	_, err := r.db.NewDelete().Model((*models.SeasonalProduct)(nil)).
		Where("\"collectionId\" = ?", collectionID).
		Exec(context.Background())
	return err
}
