package services

import (
	"refer_api/models"
	"refer_api/repository"
)

// SearchService defines business logic for text-based and category-based searches.
type SearchService interface {
	SearchProducts(query, category string) ([]models.MappedProductResponse, error)
	SearchProductsPaged(query, category string, page, limit int) ([]models.MappedProductResponse, int, error)
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
	mapped, _, err := s.SearchProductsPaged(query, category, 0, 0)
	return mapped, err
}

func (s *searchService) SearchProductsPaged(query, category string, page, limit int) ([]models.MappedProductResponse, int, error) {
	products, total, err := s.productRepo.SearchPaged(query, category, page, limit)
	if err != nil {
		return nil, 0, err
	}

	mapped := make([]models.MappedProductResponse, 0, len(products))
	for _, p := range products {
		mapped = append(mapped, *models.MapProduct(&p))
	}

	return mapped, total, nil
}

func (s *searchService) GetCategories() ([]models.ProductCategory, error) {
	return s.productRepo.FindCategories()
}
