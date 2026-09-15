package repository

import (
	"context"
	"time"

	"refer_api/models"

	"github.com/uptrace/bun"
)

// ConfigRepository defines persistence for generic app settings.
type ConfigRepository interface {
	FindAll() ([]models.AppSetting, error)
	Upsert(key, value string) (*models.AppSetting, error)
}

type configRepository struct {
	db *bun.DB
}

// NewConfigRepository constructs a new ConfigRepository.
func NewConfigRepository(db *bun.DB) ConfigRepository {
	return &configRepository{db: db}
}

func (r *configRepository) FindAll() ([]models.AppSetting, error) {
	settings := []models.AppSetting{}
	err := r.db.NewSelect().Model(&settings).Scan(context.Background())
	return settings, err
}

func (r *configRepository) Upsert(key, value string) (*models.AppSetting, error) {
	setting := &models.AppSetting{
		Key:       key,
		Value:     value,
		UpdatedAt: time.Now(),
	}
	_, err := r.db.NewInsert().Model(setting).
		On(`CONFLICT ("key") DO UPDATE`).
		Set(`"value" = EXCLUDED."value"`).
		Set(`"updatedAt" = EXCLUDED."updatedAt"`).
		Exec(context.Background())
	if err != nil {
		return nil, err
	}
	return setting, nil
}
