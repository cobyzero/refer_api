package repository

import (
	"context"
	"refer_api/models"

	"github.com/uptrace/bun"
)

// ProductRepository defines database operations for products, categories, and enhancements.
type ProductRepository interface {
	FindAll() ([]models.Product, error)
	// FindAllPaged returns one page plus the total count (for ?page&limit).
	// page is 1-based; limit <= 0 means legacy full list.
	FindAllPaged(page, limit int) ([]models.Product, int, error)
	CountAll() (int, error)
	FindByID(id string) (*models.Product, error)
	FindTopRated() ([]models.Product, error)
	FindSeasonal() ([]models.SeasonalProduct, error)
	// FindAllTypesLight / FindAllEnhancementsLight are dedicated lightweight
	// queries for the /types and /enhancements listings: they avoid loading
	// the full product graph (5 relations) just to list children.
	FindAllTypesLight() ([]models.ProductType, error)
	FindAllEnhancementsLight() ([]models.ProductEnhancement, error)
	// FindProductNames returns id -> name for the given ids in one query.
	FindProductNames(ids []string) (map[string]string, error)
	FindCategories() ([]models.ProductCategory, error)
	CreateCategory(category *models.ProductCategory) error
	UpdateCategory(category *models.ProductCategory) error
	DeleteCategory(id string) error
	UnassignCategory(productCategoryID string) error
	FindSizes() ([]models.ProductSize, error)
	CreateSize(size *models.ProductSize) error
	UpdateSize(size *models.ProductSize) error
	DeleteSize(id string) error
	FindSizeAssignment(productID, sizeID string) (*models.ProductAvailableSize, error)
	UpsertSizeAssignment(assignment *models.ProductAvailableSize) error
	RemoveSizeAssignment(productID, sizeID string) error
	RemoveSizeAssignmentsBySize(sizeID string) error
	Create(product *models.Product) error
	Update(product *models.Product) error
	CreateEnhancement(enh *models.ProductEnhancement) error
	DeleteEnhancement(id string) error
	FindEnhancementByID(id string) (*models.ProductEnhancement, error)
	CreateType(opt *models.ProductType) error
	DeleteType(id string) error
	FindTypeByID(id string) (*models.ProductType, error)
	Search(query string, categoryIdOrName string) ([]models.Product, error)
	// SearchPaged is the paginated variant of Search (page 1-based).
	SearchPaged(query string, categoryIdOrName string, page, limit int) ([]models.Product, int, error)
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
	products := []models.Product{}
	err := r.withRelations(r.db.NewSelect().Model(&products)).
		OrderExpr("?TableAlias.\"createdAt\" DESC").
		Scan(context.Background())
	return products, err
}

// CountAll returns SELECT COUNT(*) over products (cheap, no relations).
func (r *productRepository) CountAll() (int, error) {
	ctx := context.Background()
	count, err := r.db.NewSelect().Model((*models.Product)(nil)).Count(ctx)
	return count, err
}

// FindAllPaged returns one page of products with relations plus the total.
// page is 1-based; page < 1 defaults to 1; limit <= 0 means full list.
func (r *productRepository) FindAllPaged(page, limit int) ([]models.Product, int, error) {
	ctx := context.Background()
	total, err := r.CountAll()
	if err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		return r.FindAllWithTotal(total)
	}
	if page < 1 {
		page = 1
	}
	products := []models.Product{}
	err = r.withRelations(r.db.NewSelect().Model(&products)).
		OrderExpr("?TableAlias.\"createdAt\" DESC").
		Limit(limit).
		Offset((page - 1) * limit).
		Scan(ctx)
	return products, total, err
}

// FindAllWithTotal reuses the legacy full-list query when pagination is off.
func (r *productRepository) FindAllWithTotal(total int) ([]models.Product, int, error) {
	products, err := r.FindAll()
	return products, total, err
}

// FindAllTypesLight lists every variant without loading products.
func (r *productRepository) FindAllTypesLight() ([]models.ProductType, error) {
	opts := []models.ProductType{}
	err := r.db.NewSelect().Model(&opts).
		OrderExpr("\"product_type\".\"name\" ASC").
		Scan(context.Background())
	return opts, err
}

// FindAllEnhancementsLight lists every extra without loading products.
func (r *productRepository) FindAllEnhancementsLight() ([]models.ProductEnhancement, error) {
	enhs := []models.ProductEnhancement{}
	err := r.db.NewSelect().Model(&enhs).
		OrderExpr("\"product_enhancement\".\"name\" ASC").
		Scan(context.Background())
	return enhs, err
}

// FindProductNames resolves product names for a set of ids in one query.
func (r *productRepository) FindProductNames(ids []string) (map[string]string, error) {
	out := map[string]string{}
	if len(ids) == 0 {
		return out, nil
	}
	type nameRow struct {
		ID   string `bun:"id"`
		Name string `bun:"name"`
	}
	rows := []nameRow{}
	err := r.db.NewSelect().
		Model((*models.Product)(nil)).
		Column("id", "name").
		Where("id IN (?)", bun.In(ids)).
		Scan(context.Background(), &rows)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = row.Name
	}
	return out, nil
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
	products := []models.Product{}
	err := r.withRelations(r.db.NewSelect().Model(&products)).
		Where("?TableAlias.\"isAvailable\" = ?", true).
		OrderExpr("?TableAlias.rating DESC").
		Limit(5).
		Scan(context.Background())
	return products, err
}

func (r *productRepository) FindSeasonal() ([]models.SeasonalProduct, error) {
	seasonal := []models.SeasonalProduct{}
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
	categories := []models.ProductCategory{}
	err := r.db.NewSelect().Model(&categories).Scan(context.Background())
	return categories, err
}

func (r *productRepository) CreateCategory(category *models.ProductCategory) error {
	_, err := r.db.NewInsert().Model(category).Exec(context.Background())
	return err
}

func (r *productRepository) UpdateCategory(category *models.ProductCategory) error {
	_, err := r.db.NewUpdate().Model(category).WherePK().Exec(context.Background())
	return err
}

func (r *productRepository) DeleteCategory(id string) error {
	_, err := r.db.NewDelete().Model((*models.ProductCategory)(nil)).Where("id = ?", id).Exec(context.Background())
	return err
}

// UnassignCategory clears the category of every product using it.
// Used before deleting a category so no product is ever removed.
func (r *productRepository) UnassignCategory(productCategoryID string) error {
	_, err := r.db.NewUpdate().Model((*models.Product)(nil)).
		Set("\"categoryId\" = NULL").
		Where("\"categoryId\" = ?", productCategoryID).
		Exec(context.Background())
	return err
}

func (r *productRepository) FindSizes() ([]models.ProductSize, error) {
	sizes := []models.ProductSize{}
	err := r.db.NewSelect().Model(&sizes).OrderExpr("\"product_size\".\"name\" ASC").Scan(context.Background())
	return sizes, err
}

func (r *productRepository) CreateSize(size *models.ProductSize) error {
	_, err := r.db.NewInsert().Model(size).Exec(context.Background())
	return err
}

func (r *productRepository) UpdateSize(size *models.ProductSize) error {
	_, err := r.db.NewUpdate().Model(size).WherePK().Exec(context.Background())
	return err
}

func (r *productRepository) DeleteSize(id string) error {
	_, err := r.db.NewDelete().Model((*models.ProductSize)(nil)).Where("id = ?", id).Exec(context.Background())
	return err
}

func (r *productRepository) FindSizeAssignment(productID, sizeID string) (*models.ProductAvailableSize, error) {
	var assignment models.ProductAvailableSize
	err := r.db.NewSelect().Model(&assignment).
		Where("\"product_available_size\".\"productId\" = ?", productID).
		Where("\"product_available_size\".\"sizeId\" = ?", sizeID).
		Scan(context.Background())
	if err != nil {
		return nil, err
	}
	return &assignment, nil
}

func (r *productRepository) UpsertSizeAssignment(assignment *models.ProductAvailableSize) error {
	existing, err := r.FindSizeAssignment(assignment.ProductID, assignment.SizeID)
	if err == nil && existing != nil {
		existing.Price = assignment.Price
		_, err = r.db.NewUpdate().Model(existing).WherePK().Exec(context.Background())
		return err
	}
	_, err = r.db.NewInsert().Model(assignment).Exec(context.Background())
	return err
}

func (r *productRepository) RemoveSizeAssignment(productID, sizeID string) error {
	_, err := r.db.NewDelete().Model((*models.ProductAvailableSize)(nil)).
		Where("\"productId\" = ?", productID).
		Where("\"sizeId\" = ?", sizeID).
		Exec(context.Background())
	return err
}

func (r *productRepository) RemoveSizeAssignmentsBySize(sizeID string) error {
	_, err := r.db.NewDelete().Model((*models.ProductAvailableSize)(nil)).
		Where("\"sizeId\" = ?", sizeID).
		Exec(context.Background())
	return err
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

func (r *productRepository) CreateType(opt *models.ProductType) error {
	_, err := r.db.NewInsert().Model(opt).Exec(context.Background())
	return err
}

func (r *productRepository) DeleteType(id string) error {
	_, err := r.db.NewDelete().Model((*models.ProductType)(nil)).Where("id = ?", id).Exec(context.Background())
	return err
}

func (r *productRepository) FindTypeByID(id string) (*models.ProductType, error) {
	var opt models.ProductType
	err := r.db.NewSelect().Model(&opt).Where("id = ?", id).Scan(context.Background())
	if err != nil {
		return nil, err
	}
	return &opt, nil
}

func (r *productRepository) Search(query string, categoryIdOrName string) ([]models.Product, error) {
	products, _, err := r.SearchPaged(query, categoryIdOrName, 0, 0)
	return products, err
}

func (r *productRepository) applySearchFilters(q *bun.SelectQuery, query string, categoryIdOrName string) *bun.SelectQuery {
	q = q.Where("?TableAlias.\"isAvailable\" = ?", true)

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
	return q
}

// SearchPaged returns one page of search results plus the total count.
// page < 1 or limit <= 0 means legacy full list (with total still computed).
func (r *productRepository) SearchPaged(query string, categoryIdOrName string, page, limit int) ([]models.Product, int, error) {
	ctx := context.Background()

	countModel := []models.Product{}
	countQ := r.db.NewSelect().Model(&countModel)
	countQ = r.applySearchFilters(countQ, query, categoryIdOrName)
	total, err := countQ.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	products := []models.Product{}
	q := r.withRelations(r.db.NewSelect().Model(&products))
	q = r.applySearchFilters(q, query, categoryIdOrName)
	q = q.OrderExpr("?TableAlias.name ASC")
	if limit > 0 {
		if page < 1 {
			page = 1
		}
		q = q.Limit(limit).Offset((page - 1) * limit)
	}

	err = q.Scan(ctx)
	return products, total, err
}
