package services

import (
	"errors"
	"strings"
	"time"

	"refer_api/models"
	"refer_api/repository"

	"github.com/google/uuid"
)

// ProductService defines the business logic interface for managing products, categories, and options.
type ProductService interface {
	FindAll() ([]models.Product, error)
	// FindAllPaged returns one page plus total (page 1-based; limit<=0 = all).
	FindAllPaged(page, limit int) ([]models.Product, int, error)
	// FindAllTypesLight / FindAllEnhancementsLight avoid the full product graph.
	FindAllTypesLight() ([]models.ProductType, map[string]string, error)
	FindAllEnhancementsLight() ([]models.ProductEnhancement, map[string]string, error)
	FindByID(id string) (*models.Product, error)
	FindTopRated() ([]models.Product, error)
	FindSeasonal() ([]models.Product, error)
	FindCategories() ([]models.ProductCategory, error)
	CreateCategory(name, imageUrl, description string) (*models.ProductCategory, error)
	UpdateCategory(id string, name, imageUrl, description *string) (*models.ProductCategory, error)
	DeleteCategory(id string) error
	FindSizes() ([]models.ProductSize, error)
	CreateSize(name string) (*models.ProductSize, error)
	UpdateSize(id, name string) (*models.ProductSize, error)
	DeleteSize(id string) error
	AssignSize(productID, sizeID string, price float64) (*models.Product, error)
	RemoveSize(productID, sizeID string) (*models.Product, error)
	Create(body map[string]interface{}) (*models.Product, error)
	Update(id string, body map[string]interface{}) (*models.Product, error)
	AddEnhancement(productID, name string, price float64) (*models.Product, error)
	DeleteEnhancement(enhancementID string) (*models.Product, error)
	AddType(productID, name string, price float64) (*models.Product, error)
	DeleteType(typeID string) (*models.Product, error)
}

type productService struct {
	productRepo repository.ProductRepository
}

// NewProductService constructs a new ProductService.
func NewProductService(productRepo repository.ProductRepository) ProductService {
	return &productService{productRepo: productRepo}
}

func (s *productService) FindAll() ([]models.Product, error) {
	return s.productRepo.FindAll()
}

func (s *productService) FindAllPaged(page, limit int) ([]models.Product, int, error) {
	return s.productRepo.FindAllPaged(page, limit)
}

// FindAllTypesLight returns variants + owner names with 2 small queries.
func (s *productService) FindAllTypesLight() ([]models.ProductType, map[string]string, error) {
	opts, err := s.productRepo.FindAllTypesLight()
	if err != nil {
		return nil, nil, err
	}
	ids := make([]string, 0, len(opts))
	seen := map[string]struct{}{}
	for _, o := range opts {
		if _, ok := seen[o.ProductID]; !ok {
			seen[o.ProductID] = struct{}{}
			ids = append(ids, o.ProductID)
		}
	}
	names, err := s.productRepo.FindProductNames(ids)
	if err != nil {
		return opts, map[string]string{}, nil
	}
	return opts, names, nil
}

// FindAllEnhancementsLight returns extras + owner names with 2 small queries.
func (s *productService) FindAllEnhancementsLight() ([]models.ProductEnhancement, map[string]string, error) {
	enhs, err := s.productRepo.FindAllEnhancementsLight()
	if err != nil {
		return nil, nil, err
	}
	ids := make([]string, 0, len(enhs))
	seen := map[string]struct{}{}
	for _, e := range enhs {
		if _, ok := seen[e.ProductID]; !ok {
			seen[e.ProductID] = struct{}{}
			ids = append(ids, e.ProductID)
		}
	}
	names, err := s.productRepo.FindProductNames(ids)
	if err != nil {
		return enhs, map[string]string{}, nil
	}
	return enhs, names, nil
}

func (s *productService) FindByID(id string) (*models.Product, error) {
	return s.productRepo.FindByID(id)
}

func (s *productService) FindTopRated() ([]models.Product, error) {
	return s.productRepo.FindTopRated()
}

func (s *productService) FindSeasonal() ([]models.Product, error) {
	seasonal, err := s.productRepo.FindSeasonal()
	if err != nil {
		return nil, err
	}

	var products []models.Product
	for _, sp := range seasonal {
		if sp.Product != nil {
			products = append(products, *sp.Product)
		}
	}
	return products, nil
}

func (s *productService) FindCategories() ([]models.ProductCategory, error) {
	return s.productRepo.FindCategories()
}

// ErrCategoryNameInvalid is returned when the category name is missing or too short.
var ErrCategoryNameInvalid = errors.New("Category name is required")

// ErrCategoryNotFound is returned when the category does not exist.
var ErrCategoryNotFound = errors.New("Category not found")

// ErrCategoryExists is returned when the category name is already taken.
var ErrCategoryExists = errors.New("Category already exists")

func (s *productService) CreateCategory(name string, imageUrl string, description string) (*models.ProductCategory, error) {
	name = strings.TrimSpace(name)
	if len([]rune(name)) < 2 {
		return nil, ErrCategoryNameInvalid
	}

	categories, err := s.productRepo.FindCategories()
	if err != nil {
		return nil, err
	}
	for _, c := range categories {
		if strings.EqualFold(strings.TrimSpace(c.Name), name) {
			return nil, ErrCategoryExists
		}
	}

	category := &models.ProductCategory{
		ID:          uuid.New().String(),
		Name:        name,
		Description: strings.TrimSpace(description),
	}
	if imageUrl = strings.TrimSpace(imageUrl); imageUrl != "" {
		category.ImageURL = &imageUrl
	}
	if err := s.productRepo.CreateCategory(category); err != nil {
		// Unique-constraint backstop for the check-then-insert race.
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, ErrCategoryExists
		}
		return nil, err
	}
	return category, nil
}

func (s *productService) UpdateCategory(id string, name, imageUrl, description *string) (*models.ProductCategory, error) {
	categories, err := s.productRepo.FindCategories()
	if err != nil {
		return nil, err
	}
	var category *models.ProductCategory
	for _, c := range categories {
		if c.ID == id {
			category = &c
			break
		}
	}
	if category == nil {
		return nil, ErrCategoryNotFound
	}

	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if len([]rune(trimmed)) < 2 {
			return nil, ErrCategoryNameInvalid
		}
		for _, c := range categories {
			if c.ID != id && strings.EqualFold(strings.TrimSpace(c.Name), trimmed) {
				return nil, ErrCategoryExists
			}
		}
		category.Name = trimmed
	}
	if description != nil {
		category.Description = strings.TrimSpace(*description)
	}
	if imageUrl != nil {
		if trimmed := strings.TrimSpace(*imageUrl); trimmed != "" {
			category.ImageURL = &trimmed
		} else {
			category.ImageURL = nil
		}
	}

	if err := s.productRepo.UpdateCategory(category); err != nil {
		return nil, err
	}
	return category, nil
}

func (s *productService) DeleteCategory(id string) error {
	categories, err := s.productRepo.FindCategories()
	if err != nil {
		return err
	}
	found := false
	for _, c := range categories {
		if c.ID == id {
			found = true
			break
		}
	}
	if !found {
		return ErrCategoryNotFound
	}

	if err := s.productRepo.UnassignCategory(id); err != nil {
		return err
	}
	return s.productRepo.DeleteCategory(id)
}

// ErrSizeNameInvalid is returned when the size name is missing or too short.
var ErrSizeNameInvalid = errors.New("Size name is required")

// ErrSizeExists is returned when the size name is already taken.
var ErrSizeExists = errors.New("Size already exists")

// ErrSizeNotFound is returned when the size does not exist.
var ErrSizeNotFound = errors.New("Size not found")

func (s *productService) FindSizes() ([]models.ProductSize, error) {
	sizes, err := s.productRepo.FindSizes()
	if err != nil {
		return nil, err
	}
	if sizes == nil {
		sizes = []models.ProductSize{}
	}
	return sizes, nil
}

func (s *productService) CreateSize(name string) (*models.ProductSize, error) {
	name = strings.TrimSpace(name)
	if len([]rune(name)) < 2 {
		return nil, ErrSizeNameInvalid
	}

	sizes, err := s.productRepo.FindSizes()
	if err != nil {
		return nil, err
	}
	for _, sz := range sizes {
		if strings.EqualFold(strings.TrimSpace(sz.Name), name) {
			return nil, ErrSizeExists
		}
	}

	size := &models.ProductSize{
		ID:   uuid.New().String(),
		Name: name,
	}
	if err := s.productRepo.CreateSize(size); err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, ErrSizeExists
		}
		return nil, err
	}
	return size, nil
}

func (s *productService) UpdateSize(id, name string) (*models.ProductSize, error) {
	name = strings.TrimSpace(name)
	if len([]rune(name)) < 2 {
		return nil, ErrSizeNameInvalid
	}

	sizes, err := s.productRepo.FindSizes()
	if err != nil {
		return nil, err
	}
	var size *models.ProductSize
	for _, sz := range sizes {
		if sz.ID == id {
			size = &sz
		} else if strings.EqualFold(strings.TrimSpace(sz.Name), name) {
			return nil, ErrSizeExists
		}
	}
	if size == nil {
		return nil, ErrSizeNotFound
	}

	size.Name = name
	if err := s.productRepo.UpdateSize(size); err != nil {
		return nil, err
	}
	return size, nil
}

func (s *productService) DeleteSize(id string) error {
	sizes, err := s.productRepo.FindSizes()
	if err != nil {
		return err
	}
	found := false
	for _, sz := range sizes {
		if sz.ID == id {
			found = true
			break
		}
	}
	if !found {
		return ErrSizeNotFound
	}

	if err := s.productRepo.RemoveSizeAssignmentsBySize(id); err != nil {
		return err
	}
	return s.productRepo.DeleteSize(id)
}

func (s *productService) AssignSize(productID, sizeID string, price float64) (*models.Product, error) {
	product, err := s.productRepo.FindByID(productID)
	if err != nil || product == nil {
		return nil, ErrProductNotFound
	}

	sizes, err := s.productRepo.FindSizes()
	if err != nil {
		return nil, err
	}
	found := false
	for _, sz := range sizes {
		if sz.ID == sizeID {
			found = true
			break
		}
	}
	if !found {
		return nil, ErrSizeNotFound
	}

	if price < 0 {
		price = 0
	}
	assignment := &models.ProductAvailableSize{
		ID:        uuid.New().String(),
		ProductID: productID,
		SizeID:    sizeID,
		Price:     price,
	}
	if err := s.productRepo.UpsertSizeAssignment(assignment); err != nil {
		return nil, err
	}
	return s.productRepo.FindByID(productID)
}

func (s *productService) RemoveSize(productID, sizeID string) (*models.Product, error) {
	product, err := s.productRepo.FindByID(productID)
	if err != nil || product == nil {
		return nil, ErrProductNotFound
	}
	if err := s.productRepo.RemoveSizeAssignment(productID, sizeID); err != nil {
		return nil, err
	}
	return s.productRepo.FindByID(productID)
}

func (s *productService) Create(body map[string]interface{}) (*models.Product, error) {
	name, _ := body["name"].(string)
	description, _ := body["description"].(string)
	price, _ := body["price"].(float64)
	starsRewardFloat, _ := body["starsReward"].(float64)
	starsReward := int(starsRewardFloat)
	isAvailable, ok := body["isAvailable"].(bool)
	if !ok {
		isAvailable = true
	}
	imageUrl, _ := body["imageUrl"].(string)
	categoryId, _ := body["categoryId"].(string)

	if name == "" {
		name = "New Product"
	}

	if categoryId == "" {
		categories, err := s.productRepo.FindCategories()
		if err == nil && len(categories) > 0 {
			categoryId = categories[0].ID
		}
	}

	product := &models.Product{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Price:       price,
		StarsReward: starsReward,
		IsAvailable: isAvailable,
		Rating:      5.0,
		NumReviews:  1,
		CreatedAt:   time.Now(),
	}

	if categoryId != "" {
		product.CategoryID = &categoryId
	}
	if imageUrl != "" {
		product.ImageURL = &imageUrl
	}

	if err := s.productRepo.Create(product); err != nil {
		return nil, err
	}

	return s.productRepo.FindByID(product.ID)
}

func (s *productService) Update(id string, body map[string]interface{}) (*models.Product, error) {
	product, err := s.productRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	if name, ok := body["name"].(string); ok {
		product.Name = name
	}
	if description, ok := body["description"].(string); ok {
		product.Description = description
	}
	if price, ok := body["price"].(float64); ok {
		product.Price = price
	}
	if starsReward, ok := body["starsReward"].(float64); ok {
		product.StarsReward = int(starsReward)
	}
	if isAvailable, ok := body["isAvailable"].(bool); ok {
		product.IsAvailable = isAvailable
	}
	if imageUrl, ok := body["imageUrl"].(string); ok {
		product.ImageURL = &imageUrl
	}
	if categoryId, ok := body["categoryId"].(string); ok {
		product.CategoryID = &categoryId
	}

	if err := s.productRepo.Update(product); err != nil {
		return nil, err
	}

	return s.productRepo.FindByID(id)
}

func (s *productService) AddEnhancement(productID, name string, price float64) (*models.Product, error) {
	enhancement := &models.ProductEnhancement{
		ID:        uuid.New().String(),
		Name:      name,
		Price:     price,
		ProductID: productID,
	}

	if err := s.productRepo.CreateEnhancement(enhancement); err != nil {
		return nil, err
	}

	return s.productRepo.FindByID(productID)
}

func (s *productService) DeleteEnhancement(enhancementID string) (*models.Product, error) {
	enh, err := s.productRepo.FindEnhancementByID(enhancementID)
	if err != nil {
		return nil, errors.New("enhancement not found")
	}

	productID := enh.ProductID
	if err := s.productRepo.DeleteEnhancement(enhancementID); err != nil {
		return nil, err
	}

	return s.productRepo.FindByID(productID)
}

// ErrTypeNameInvalid is returned when the variant name is missing or too short.
var ErrTypeNameInvalid = errors.New("Variant name is required")

// ErrTypeNotFound is returned when the variant does not exist.
var ErrTypeNotFound = errors.New("Variant not found")

func (s *productService) AddType(productID, name string, price float64) (*models.Product, error) {
	if len([]rune(strings.TrimSpace(name))) < 2 {
		return nil, ErrTypeNameInvalid
	}
	product, err := s.productRepo.FindByID(productID)
	if err != nil || product == nil {
		return nil, ErrProductNotFound
	}

	opt := &models.ProductType{
		ID:        uuid.New().String(),
		Name:      strings.TrimSpace(name),
		Price:     price,
		ProductID: productID,
	}
	if price < 0 {
		opt.Price = 0
	}

	if err := s.productRepo.CreateType(opt); err != nil {
		return nil, err
	}

	return s.productRepo.FindByID(productID)
}

func (s *productService) DeleteType(typeID string) (*models.Product, error) {
	opt, err := s.productRepo.FindTypeByID(typeID)
	if err != nil {
		return nil, ErrTypeNotFound
	}

	productID := opt.ProductID
	if err := s.productRepo.DeleteType(typeID); err != nil {
		return nil, err
	}

	return s.productRepo.FindByID(productID)
}
