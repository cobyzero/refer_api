package services

import (
	"errors"
	"strings"
	"time"

	"refer_api/models"
	"refer_api/repository"

	"github.com/google/uuid"
)

// ErrCollectionNameInvalid is returned when the collection name is missing or too short.
var ErrCollectionNameInvalid = errors.New("Collection name is required")

// ErrCollectionExists is returned when the collection name is already taken.
var ErrCollectionExists = errors.New("Collection already exists")

// ErrCollectionNotFound is returned when the collection does not exist.
var ErrCollectionNotFound = errors.New("Collection not found")

// ErrCollectionItemExists is returned when the product is already in the collection.
var ErrCollectionItemExists = errors.New("Product is already in this collection")

// ErrProductNotFound is returned when the product to feature does not exist.
var ErrProductNotFound = errors.New("Product not found")

// CollectionService defines business logic for featured collections.
type CollectionService interface {
	FindAll() ([]models.Collection, error)
	FindByID(id string) (*models.Collection, error)
	Create(name, description string) (*models.Collection, error)
	Update(id string, name *string, description *string, isActive *bool) (*models.Collection, error)
	Delete(id string) error
	AddItem(collectionID, productID string) (*models.Collection, error)
	RemoveItem(collectionID, productID string) (*models.Collection, error)
}

type collectionService struct {
	collectionRepo repository.CollectionRepository
	productRepo    repository.ProductRepository
}

// NewCollectionService constructs a new CollectionService.
func NewCollectionService(collectionRepo repository.CollectionRepository, productRepo repository.ProductRepository) CollectionService {
	return &collectionService{collectionRepo: collectionRepo, productRepo: productRepo}
}

func (s *collectionService) FindAll() ([]models.Collection, error) {
	collections, err := s.collectionRepo.FindAll()
	if err != nil {
		return nil, err
	}
	if collections == nil {
		collections = []models.Collection{}
	}
	return collections, nil
}

func (s *collectionService) FindByID(id string) (*models.Collection, error) {
	collection, err := s.collectionRepo.FindByID(id)
	if err != nil {
		return nil, ErrCollectionNotFound
	}
	return collection, nil
}

func (s *collectionService) Create(name, description string) (*models.Collection, error) {
	name = strings.TrimSpace(name)
	if len([]rune(name)) < 2 {
		return nil, ErrCollectionNameInvalid
	}

	existing, err := s.collectionRepo.FindAll()
	if err != nil {
		return nil, err
	}
	for _, c := range existing {
		if strings.EqualFold(strings.TrimSpace(c.Name), name) {
			return nil, ErrCollectionExists
		}
	}

	collection := &models.Collection{
		ID:          uuid.New().String(),
		Name:        name,
		Description: strings.TrimSpace(description),
		IsActive:    true,
		CreatedAt:   time.Now(),
	}
	if err := s.collectionRepo.Create(collection); err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, ErrCollectionExists
		}
		return nil, err
	}
	return s.collectionRepo.FindByID(collection.ID)
}

func (s *collectionService) Update(id string, name *string, description *string, isActive *bool) (*models.Collection, error) {
	collection, err := s.collectionRepo.FindByID(id)
	if err != nil {
		return nil, ErrCollectionNotFound
	}

	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if len([]rune(trimmed)) < 2 {
			return nil, ErrCollectionNameInvalid
		}
		existing, err := s.collectionRepo.FindAll()
		if err != nil {
			return nil, err
		}
		for _, c := range existing {
			if c.ID != id && strings.EqualFold(strings.TrimSpace(c.Name), trimmed) {
				return nil, ErrCollectionExists
			}
		}
		collection.Name = trimmed
	}
	if description != nil {
		collection.Description = strings.TrimSpace(*description)
	}
	if isActive != nil {
		collection.IsActive = *isActive
	}

	if err := s.collectionRepo.Update(collection); err != nil {
		return nil, err
	}
	return s.collectionRepo.FindByID(id)
}

func (s *collectionService) Delete(id string) error {
	if _, err := s.collectionRepo.FindByID(id); err != nil {
		return ErrCollectionNotFound
	}
	// Unassign items first so products themselves are never deleted.
	if err := s.collectionRepo.RemoveAllItems(id); err != nil {
		return err
	}
	return s.collectionRepo.Delete(id)
}

func (s *collectionService) AddItem(collectionID, productID string) (*models.Collection, error) {
	if _, err := s.collectionRepo.FindByID(collectionID); err != nil {
		return nil, ErrCollectionNotFound
	}
	if _, err := s.productRepo.FindByID(productID); err != nil {
		return nil, ErrProductNotFound
	}
	if existing, err := s.collectionRepo.FindItem(collectionID, productID); err == nil && existing != nil {
		return nil, ErrCollectionItemExists
	}

	item := &models.SeasonalProduct{
		ID:           uuid.New().String(),
		ProductID:    productID,
		CollectionID: &collectionID,
	}
	if err := s.collectionRepo.AddItem(item); err != nil {
		return nil, err
	}
	return s.collectionRepo.FindByID(collectionID)
}

func (s *collectionService) RemoveItem(collectionID, productID string) (*models.Collection, error) {
	if _, err := s.collectionRepo.FindByID(collectionID); err != nil {
		return nil, ErrCollectionNotFound
	}
	if err := s.collectionRepo.RemoveItem(collectionID, productID); err != nil {
		return nil, err
	}
	return s.collectionRepo.FindByID(collectionID)
}
