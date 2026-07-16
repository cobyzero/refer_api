package services

import (
	"refer_api/models"
	"refer_api/repository"
)

// LocationService defines the business logic interface for managing locations.
type LocationService interface {
	FindAll() ([]models.Location, error)
	FindByID(id string) (*models.Location, error)
}

type locationService struct {
	locationRepo repository.LocationRepository
}

// NewLocationService constructs a new LocationService.
func NewLocationService(locationRepo repository.LocationRepository) LocationService {
	return &locationService{locationRepo: locationRepo}
}

func (s *locationService) FindAll() ([]models.Location, error) {
	return s.locationRepo.FindAll()
}

func (s *locationService) FindByID(id string) (*models.Location, error) {
	return s.locationRepo.FindByID(id)
}
