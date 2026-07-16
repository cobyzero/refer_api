package services

import (
	"errors"
	"time"

	"refer_api/models"
	"refer_api/repository"

	"github.com/google/uuid"
)

// ProductService defines the business logic interface for managing products, categories, and options.
type ProductService interface {
	FindAll() ([]models.Product, error)
	FindByID(id string) (*models.Product, error)
	FindTopRated() ([]models.Product, error)
	FindSeasonal() ([]models.Product, error)
	FindCategories() ([]models.ProductCategory, error)
	Create(body map[string]interface{}) (*models.Product, error)
	Update(id string, body map[string]interface{}) (*models.Product, error)
	AddEnhancement(productID, name string, price float64) (*models.Product, error)
	DeleteEnhancement(enhancementID string) (*models.Product, error)
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
