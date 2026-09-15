package repository

import (
	"context"
	"refer_api/models"

	"github.com/uptrace/bun"
)

// StarRepository defines database operations for rewards and star transactions.
type StarRepository interface {
	GetStarBalance(userID string) (int, error)
	FindHistory(userID string) ([]models.StarTransaction, error)
	// FindHistoryRecent returns at most limit rows (for dashboard recent activity).
	FindHistoryRecent(userID string, limit int) ([]models.StarTransaction, error)
	FindRewards() ([]models.Reward, error)
	FindRewardByID(id string) (*models.Reward, error)
	CreateTransaction(tx *models.StarTransaction) error
	CreateRedeemedReward(rr *models.RedeemedReward) error
	FindRedeemedRewards(userID string) ([]models.RedeemedReward, error)
	FindRedeemedRewardByID(id string) (*models.RedeemedReward, error)
	UpdateRedeemedReward(rr *models.RedeemedReward) error
	GetTotalStarsAwarded() (int, error)
	GetTotalStarsRedeemed() (int, error)
	GetAllTransactions() ([]models.StarTransaction, error)
	CreateReward(reward *models.Reward) error
	UpdateReward(reward *models.Reward) error
	DeleteReward(id string) error
}

type starRepository struct {
	db *bun.DB
}

// NewStarRepository constructs a new StarRepository.
func NewStarRepository(db *bun.DB) StarRepository {
	return &starRepository{db: db}
}

func (r *starRepository) GetStarBalance(userID string) (int, error) {
	var result struct {
		Total int `bun:"total"`
	}
	err := r.db.NewSelect().
		TableExpr("\"star_transactions\"").
		ColumnExpr("COALESCE(SUM(stars), 0) AS total").
		Where("\"userId\" = ?", userID).
		Scan(context.Background(), &result)
	return result.Total, err
}

func (r *starRepository) FindHistory(userID string) ([]models.StarTransaction, error) {
	return r.FindHistoryRecent(userID, 0)
}

func (r *starRepository) FindHistoryRecent(userID string, limit int) ([]models.StarTransaction, error) {
	txs := []models.StarTransaction{}
	q := r.db.NewSelect().Model(&txs).
		Where("\"userId\" = ?", userID).
		OrderExpr("date DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Scan(context.Background())
	return txs, err
}

func (r *starRepository) FindRewards() ([]models.Reward, error) {
	rewards := []models.Reward{}
	err := r.db.NewSelect().Model(&rewards).Where("\"isActive\" = ?", true).Scan(context.Background())
	return rewards, err
}

func (r *starRepository) FindRewardByID(id string) (*models.Reward, error) {
	var reward models.Reward
	err := r.db.NewSelect().Model(&reward).Where("id = ?", id).Scan(context.Background())
	if err != nil {
		return nil, err
	}
	return &reward, nil
}

func (r *starRepository) CreateTransaction(tx *models.StarTransaction) error {
	_, err := r.db.NewInsert().Model(tx).Exec(context.Background())
	return err
}

func (r *starRepository) CreateRedeemedReward(rr *models.RedeemedReward) error {
	_, err := r.db.NewInsert().Model(rr).Exec(context.Background())
	return err
}

func (r *starRepository) FindRedeemedRewards(userID string) ([]models.RedeemedReward, error) {
	rrs := []models.RedeemedReward{}
	err := r.db.NewSelect().Model(&rrs).
		Where("\"userId\" = ?", userID).
		OrderExpr("\"createdAt\" DESC").
		Scan(context.Background())
	return rrs, err
}

func (r *starRepository) FindRedeemedRewardByID(id string) (*models.RedeemedReward, error) {
	var rr models.RedeemedReward
	err := r.db.NewSelect().Model(&rr).Where("id = ?", id).Scan(context.Background())
	if err != nil {
		return nil, err
	}
	return &rr, nil
}

func (r *starRepository) UpdateRedeemedReward(rr *models.RedeemedReward) error {
	_, err := r.db.NewUpdate().Model(rr).WherePK().Exec(context.Background())
	return err
}

func (r *starRepository) GetTotalStarsAwarded() (int, error) {
	var result struct {
		Total int `bun:"total"`
	}
	err := r.db.NewSelect().
		TableExpr("\"star_transactions\"").
		ColumnExpr("COALESCE(SUM(stars), 0) AS total").
		Where("type = ?", "earn").
		Scan(context.Background(), &result)
	return result.Total, err
}

func (r *starRepository) GetTotalStarsRedeemed() (int, error) {
	var result struct {
		Total int `bun:"total"`
	}
	err := r.db.NewSelect().
		TableExpr("\"star_transactions\"").
		ColumnExpr("COALESCE(SUM(ABS(stars)), 0) AS total").
		Where("type = ?", "redeem").
		Scan(context.Background(), &result)
	return result.Total, err
}

func (r *starRepository) GetAllTransactions() ([]models.StarTransaction, error) {
	txs := []models.StarTransaction{}
	err := r.db.NewSelect().Model(&txs).
		Relation("User").
		OrderExpr("date DESC").
		Scan(context.Background())
	return txs, err
}

func (r *starRepository) CreateReward(reward *models.Reward) error {
	_, err := r.db.NewInsert().Model(reward).Exec(context.Background())
	return err
}

func (r *starRepository) UpdateReward(reward *models.Reward) error {
	_, err := r.db.NewUpdate().Model(reward).WherePK().Exec(context.Background())
	return err
}

func (r *starRepository) DeleteReward(id string) error {
	_, err := r.db.NewDelete().Model((*models.Reward)(nil)).Where("id = ?", id).Exec(context.Background())
	return err
}
