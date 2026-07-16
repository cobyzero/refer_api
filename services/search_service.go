package services

import (
	"refer_api/models"
	"refer_api/repository"
)

// SearchService defines business logic for text-based and category-based searches.
type SearchService interface {
	SearchProducts(query, category string) ([]models.MappedProductResponse, error)
	GetCategories() ([]models.ProductCategory, error)
}

type searchService struct {
	productRepo repository.ProductRepository
}

// NewSearchService constructs a new SearchService.
func NewSearchService(productRepo repository.ProductRepository) SearchService {
	return &searchService{productRepo: productRepo}
}

func (s *searchService) SearchProducts(query, category string) ([]models.MappedProductResponse, error) {
	products, err := s.productRepo.Search(query, category)
	if err != nil {
		return nil, err
	}

	var mapped []models.MappedProductResponse
	for _, p := range products {
		mapped = append(mapped, *models.MapProduct(&p))
	}

	return mapped, nil
}

func (s *searchService) GetCategories() ([]models.ProductCategory, error) {
	return s.productRepo.FindCategories()
}
