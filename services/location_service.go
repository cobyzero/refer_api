package services

import (
	"errors"
	"strings"

	"refer_api/models"
	"refer_api/repository"

	"github.com/google/uuid"
)

// LocationService defines the business logic interface for managing locations.
type LocationService interface {
	FindAll() ([]models.Location, error)
	FindByID(id string) (*models.Location, error)
	FindAllAdmin() ([]models.Location, error)
	Create(input LocationInput) (*models.Location, error)
	Update(id string, input LocationInput) (*models.Location, error)
	SetActive(id string, isActive bool) (*models.Location, error)
	Delete(id string) error
}

// LocationInput carries the operator-editable fields of a location.
// Pointers distinguish "absent" from zero values on update.
type LocationInput struct {
	Name      *string
	Address   *string
	City      *string
	State     *string
	ZipCode   *string
	Latitude  *float64
	Longitude *float64
	IsActive  *bool
}

// ErrLocationNameInvalid is returned when the location name is missing or too short.
var ErrLocationNameInvalid = errors.New("Location name is required")

// ErrLocationAddressInvalid is returned when the address is missing.
var ErrLocationAddressInvalid = errors.New("Location address is required")

// ErrLocationNotFound is returned when the location does not exist.
var ErrLocationNotFound = errors.New("Location not found")

// ErrLocationInUse is returned when the location has orders and cannot be deleted.
var ErrLocationInUse = errors.New("Location has orders and cannot be deleted")

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

func (s *locationService) FindAllAdmin() ([]models.Location, error) {
	locations, err := s.locationRepo.FindAllAdmin()
	if err != nil {
		return nil, err
	}
	if locations == nil {
		locations = []models.Location{}
	}
	return locations, nil
}

func (s *locationService) Create(input LocationInput) (*models.Location, error) {
	name := strVal(input.Name)
	address := strVal(input.Address)
	if len([]rune(name)) < 2 {
		return nil, ErrLocationNameInvalid
	}
	if len([]rune(address)) < 2 {
		return nil, ErrLocationAddressInvalid
	}

	location := &models.Location{
		ID:        uuid.New().String(),
		Name:      name,
		Address:   address,
		City:      optStr(input.City),
		State:     optStr(input.State),
		ZipCode:   optStr(input.ZipCode),
		Latitude:  input.Latitude,
		Longitude: input.Longitude,
		IsActive:  true,
	}
	if err := s.locationRepo.Create(location); err != nil {
		return nil, err
	}
	return s.locationRepo.FindByIDAdmin(location.ID)
}

func (s *locationService) Update(id string, input LocationInput) (*models.Location, error) {
	location, err := s.locationRepo.FindByIDAdmin(id)
	if err != nil {
		return nil, ErrLocationNotFound
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if len([]rune(name)) < 2 {
			return nil, ErrLocationNameInvalid
		}
		location.Name = name
	}
	if input.Address != nil {
		address := strings.TrimSpace(*input.Address)
		if len([]rune(address)) < 2 {
			return nil, ErrLocationAddressInvalid
		}
		location.Address = address
	}
	if input.City != nil {
		location.City = optStr(input.City)
	}
	if input.State != nil {
		location.State = optStr(input.State)
	}
	if input.ZipCode != nil {
		location.ZipCode = optStr(input.ZipCode)
	}
	if input.Latitude != nil {
		location.Latitude = input.Latitude
	}
	if input.Longitude != nil {
		location.Longitude = input.Longitude
	}
	if input.IsActive != nil {
		location.IsActive = *input.IsActive
	}

	if err := s.locationRepo.Update(location); err != nil {
		return nil, err
	}
	return s.locationRepo.FindByIDAdmin(id)
}

func (s *locationService) SetActive(id string, isActive bool) (*models.Location, error) {
	location, err := s.locationRepo.FindByIDAdmin(id)
	if err != nil {
		return nil, ErrLocationNotFound
	}
	location.IsActive = isActive
	if err := s.locationRepo.Update(location); err != nil {
		return nil, err
	}
	return s.locationRepo.FindByIDAdmin(id)
}

func (s *locationService) Delete(id string) error {
	if _, err := s.locationRepo.FindByIDAdmin(id); err != nil {
		return ErrLocationNotFound
	}
	count, err := s.locationRepo.CountOrders(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrLocationInUse
	}
	return s.locationRepo.Delete(id)
}

func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}

func optStr(s *string) *string {
	if s == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*s)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
