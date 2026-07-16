package repository

import (
	"context"
	"refer_api/models"

	"github.com/uptrace/bun"
)

// LocationRepository defines database operations for physical locations.
type LocationRepository interface {
	FindAll() ([]models.Location, error)
	FindByID(id string) (*models.Location, error)
}

type locationRepository struct {
	db *bun.DB
}

// NewLocationRepository constructs a new LocationRepository.
func NewLocationRepository(db *bun.DB) LocationRepository {
	return &locationRepository{db: db}
}

func (r *locationRepository) FindAll() ([]models.Location, error) {
	var locations []models.Location
	err := r.db.NewSelect().Model(&locations).Where("isActive = ?", true).Scan(context.Background())
	return locations, err
}

func (r *locationRepository) FindByID(id string) (*models.Location, error) {
	var location models.Location
	err := r.db.NewSelect().Model(&location).Where("id = ? AND isActive = ?", id, true).Scan(context.Background())
	if err != nil {
		return nil, err
	}
	return &location, nil
}
