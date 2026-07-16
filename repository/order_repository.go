package repository

import (
	"context"
	"database/sql"
	"refer_api/models"
	"time"

	"github.com/uptrace/bun"
)

// OrderRepository defines database operations for orders.
type OrderRepository interface {
	FindAll() ([]models.Order, error)
	FindAllByUser(userID string) ([]models.Order, error)
	FindByID(orderID string) (*models.Order, error)
	UpdateStatus(orderID string, status models.OrderStatus) error
	CreateOrderViaSP(
		orderID string,
		userID string,
		locationID string,
		orderNumber string,
		total float64,
		serviceFee float64,
		paymentMethod string,
		pickupTime *time.Time,
		itemsJSON string,
		redeemedRewardID *string,
	) (int, error)
}

type orderRepository struct {
	db *bun.DB
}

// NewOrderRepository constructs a new OrderRepository.
func NewOrderRepository(db *bun.DB) OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) FindAll() ([]models.Order, error) {
	var orders []models.Order
	err := r.db.NewSelect().Model(&orders).
		Relation("Items").
		Relation("User").
		OrderExpr("?TableAlias.createdAt DESC").
		Scan(context.Background())
	return orders, err
}

func (r *orderRepository) FindAllByUser(userID string) ([]models.Order, error) {
	var orders []models.Order
	err := r.db.NewSelect().Model(&orders).
		Where("?TableAlias.userId = ?", userID).
		Relation("Items").
		Relation("User").
		OrderExpr("?TableAlias.createdAt DESC").
		Scan(context.Background())
	return orders, err
}

func (r *orderRepository) FindByID(orderID string) (*models.Order, error) {
	var order models.Order
	err := r.db.NewSelect().Model(&order).
		Where("?TableAlias.id = ?", orderID).
		Relation("Items").
		Relation("User").
		Scan(context.Background())
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *orderRepository) UpdateStatus(orderID string, status models.OrderStatus) error {
	_, err := r.db.NewUpdate().Model((*models.Order)(nil)).
		Set("status = ?", status).
		Where("id = ?", orderID).
		Exec(context.Background())
	return err
}

func (r *orderRepository) CreateOrderViaSP(
	orderID string,
	userID string,
	locationID string,
	orderNumber string,
	total float64,
	serviceFee float64,
	paymentMethod string,
	pickupTime *time.Time,
	itemsJSON string,
	redeemedRewardID *string,
) (int, error) {
	ctx := context.Background()

	var pickupVal interface{}
	if pickupTime != nil {
		pickupVal = *pickupTime
	} else {
		pickupVal = nil
	}

	// Execute the stored procedure inside an explicit transaction so the session
	// variable @totalStars is readable in the same connection.
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(ctx,
		"CALL sp_create_order(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, @totalStars)",
		orderID, userID, locationID, orderNumber,
		total, serviceFee, paymentMethod, pickupVal,
		itemsJSON, redeemedRewardID,
	); err != nil {
		return 0, err
	}

	var result struct {
		TotalStars sql.NullInt64 `bun:"total_stars"`
	}
	if err = tx.QueryRowContext(ctx, "SELECT @totalStars AS total_stars").
		Scan(&result.TotalStars); err != nil {
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return int(result.TotalStars.Int64), nil
}
