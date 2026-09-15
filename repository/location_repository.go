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
	FindAllAdmin() ([]models.Location, error)
	FindByIDAdmin(id string) (*models.Location, error)
	Create(location *models.Location) error
	Update(location *models.Location) error
	Delete(id string) error
	CountOrders(locationID string) (int, error)
}

type locationRepository struct {
	db *bun.DB
}

// NewLocationRepository constructs a new LocationRepository.
func NewLocationRepository(db *bun.DB) LocationRepository {
	return &locationRepository{db: db}
}

func (r *locationRepository) FindAll() ([]models.Location, error) {
	locations := []models.Location{}
	err := r.db.NewSelect().Model(&locations).Where("\"isActive\" = ?", true).Scan(context.Background())
	return locations, err
}

func (r *locationRepository) FindByID(id string) (*models.Location, error) {
	var location models.Location
	err := r.db.NewSelect().Model(&location).Where("id = ? AND \"isActive\" = ?", id, true).Scan(context.Background())
	if err != nil {
		return nil, err
	}
	return &location, nil
}

// FindAllAdmin returns every location for back-office management (no active filter).
func (r *locationRepository) FindAllAdmin() ([]models.Location, error) {
	locations := []models.Location{}
	err := r.db.NewSelect().Model(&locations).OrderExpr("\"location\".\"name\" ASC").Scan(context.Background())
	return locations, err
}

func (r *locationRepository) FindByIDAdmin(id string) (*models.Location, error) {
	var location models.Location
	err := r.db.NewSelect().Model(&location).Where("\"location\".\"id\" = ?", id).Scan(context.Background())
	if err != nil {
		return nil, err
	}
	return &location, nil
}

func (r *locationRepository) Create(location *models.Location) error {
	_, err := r.db.NewInsert().Model(location).Exec(context.Background())
	return err
}

func (r *locationRepository) Update(location *models.Location) error {
	_, err := r.db.NewUpdate().Model(location).WherePK().Exec(context.Background())
	return err
}

func (r *locationRepository) Delete(id string) error {
	_, err := r.db.NewDelete().Model((*models.Location)(nil)).Where("id = ?", id).Exec(context.Background())
	return err
}

// CountOrders reports how many orders reference a location.
// Lives here (not in the order repository) so the location service can
// refuse deletions without new cross-service wiring.
func (r *locationRepository) CountOrders(locationID string) (int, error) {
	count, err := r.db.NewSelect().Model((*models.Order)(nil)).
		Where("\"order\".\"locationId\" = ?", locationID).
		Count(context.Background())
	return count, err
}
