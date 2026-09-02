package repository

import (
	"context"
	"refer_api/models"

	"github.com/uptrace/bun"
)

// ProductRepository defines database operations for products, categories, and enhancements.
type ProductRepository interface {
	FindAll() ([]models.Product, error)
	FindByID(id string) (*models.Product, error)
	FindTopRated() ([]models.Product, error)
	FindSeasonal() ([]models.SeasonalProduct, error)
	FindCategories() ([]models.ProductCategory, error)
	Create(product *models.Product) error
	Update(product *models.Product) error
	CreateEnhancement(enh *models.ProductEnhancement) error
	DeleteEnhancement(id string) error
	FindEnhancementByID(id string) (*models.ProductEnhancement, error)
	Search(query string, categoryIdOrName string) ([]models.Product, error)
}

type productRepository struct {
	db *bun.DB
}

// NewProductRepository constructs a new ProductRepository.
func NewProductRepository(db *bun.DB) ProductRepository {
	return &productRepository{db: db}
}

func (r *productRepository) withRelations(q *bun.SelectQuery) *bun.SelectQuery {
	return q.
		Relation("Category").
		Relation("SizeOptions").
		Relation("SizeOptions.Size").
		Relation("Options").
		Relation("Enhancements")
}

func (r *productRepository) FindAll() ([]models.Product, error) {
	var products []models.Product
	err := r.withRelations(r.db.NewSelect().Model(&products)).
		OrderExpr("?TableAlias.\"createdAt\" DESC").
		Scan(context.Background())
	return products, err
}

func (r *productRepository) FindByID(id string) (*models.Product, error) {
	var product models.Product
	err := r.withRelations(r.db.NewSelect().Model(&product)).
		Where("?TableAlias.id = ?", id).
		Scan(context.Background())
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *productRepository) FindTopRated() ([]models.Product, error) {
	var products []models.Product
	err := r.withRelations(r.db.NewSelect().Model(&products)).
		Where("?TableAlias.\"isAvailable\" = ?", true).
		OrderExpr("?TableAlias.rating DESC").
		Limit(5).
		Scan(context.Background())
	return products, err
}

func (r *productRepository) FindSeasonal() ([]models.SeasonalProduct, error) {
	var seasonal []models.SeasonalProduct
	err := r.db.NewSelect().Model(&seasonal).
		Relation("Product").
		Relation("Product.Category").
		Relation("Product.SizeOptions").
		Relation("Product.SizeOptions.Size").
		Relation("Product.Options").
		Relation("Product.Enhancements").
		Scan(context.Background())
	return seasonal, err
}

func (r *productRepository) FindCategories() ([]models.ProductCategory, error) {
	var categories []models.ProductCategory
	err := r.db.NewSelect().Model(&categories).Scan(context.Background())
	return categories, err
}

func (r *productRepository) Create(product *models.Product) error {
	_, err := r.db.NewInsert().Model(product).Exec(context.Background())
	return err
}

func (r *productRepository) Update(product *models.Product) error {
	_, err := r.db.NewUpdate().Model(product).WherePK().Exec(context.Background())
	return err
}

func (r *productRepository) CreateEnhancement(enh *models.ProductEnhancement) error {
	_, err := r.db.NewInsert().Model(enh).Exec(context.Background())
	return err
}

func (r *productRepository) DeleteEnhancement(id string) error {
	_, err := r.db.NewDelete().Model((*models.ProductEnhancement)(nil)).Where("id = ?", id).Exec(context.Background())
	return err
}

func (r *productRepository) FindEnhancementByID(id string) (*models.ProductEnhancement, error) {
	var enh models.ProductEnhancement
	err := r.db.NewSelect().Model(&enh).Where("id = ?", id).Scan(context.Background())
	if err != nil {
		return nil, err
	}
	return &enh, nil
}

func (r *productRepository) Search(query string, categoryIdOrName string) ([]models.Product, error) {
	ctx := context.Background()
	var products []models.Product

	q := r.withRelations(r.db.NewSelect().Model(&products)).
		Where("?TableAlias.\"isAvailable\" = ?", true)

	if query != "" {
		q = q.Where("?TableAlias.name ILIKE ?", "%"+query+"%")
	}

	if categoryIdOrName != "" && categoryIdOrName != "All" {
		if len(categoryIdOrName) == 36 {
			q = q.Where("?TableAlias.\"categoryId\" = ?", categoryIdOrName)
		} else {
			q = q.Join("JOIN product_categories AS cat ON cat.id = ?TableAlias.\"categoryId\"").
				Where("cat.name = ?", categoryIdOrName)
		}
	}

	err := q.OrderExpr("?TableAlias.name ASC").Scan(ctx)
	return products, err
}
