package repository

import (
	"context"
	"encoding/json"
	"errors"
	"refer_api/models"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ErrOrderNotPending is returned when a payment-flow transition targets an
// order that is no longer awaiting payment (already paid/cancelled).
var ErrOrderNotPending = errors.New("order is not awaiting payment")

// OrderRepository defines database operations for orders.
type OrderRepository interface {
	FindAll() ([]models.Order, error)
	FindAllByUser(userID string) ([]models.Order, error)
	FindByID(orderID string) (*models.Order, error)
	UpdateStatus(orderID string, status models.OrderStatus) error
	// CreatePendingOrder inserts a PENDING_PAYMENT order with its items.
	// No stars are awarded here — that happens on MarkPaidWithStars.
	CreatePendingOrder(order *models.Order, items []models.OrderItem) error
	// SetPaymentIntent links a Stripe intent to a pending order.
	SetPaymentIntent(orderID, intentID string) error
	// MarkPaidWithStars flips a pending order to ORDERED, stamps paidAt and
	// awards the star earn transaction (when stars > 0), atomically.
	MarkPaidWithStars(orderID, userID string, stars int, title string) error
	// CancelPending flips a pending order to CANCELLED.
	CancelPending(orderID string) error
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
	orders := []models.Order{}
	err := r.db.NewSelect().Model(&orders).
		Relation("Items").
		Relation("User").
		OrderExpr("?TableAlias.\"createdAt\" DESC").
		Scan(context.Background())
	return orders, err
}

func (r *orderRepository) FindAllByUser(userID string) ([]models.Order, error) {
	orders := []models.Order{}
	err := r.db.NewSelect().Model(&orders).
		Where("?TableAlias.\"userId\" = ?", userID).
		Relation("Items").
		Relation("User").
		OrderExpr("?TableAlias.\"createdAt\" DESC").
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

// CreatePendingOrder inserts the order row plus its items in one transaction.
func (r *orderRepository) CreatePendingOrder(order *models.Order, items []models.OrderItem) error {
	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if _, err = tx.NewInsert().Model(order).Exec(ctx); err != nil {
		return err
	}
	for i := range items {
		items[i].OrderID = order.ID
		if _, err = tx.NewInsert().Model(&items[i]).Exec(ctx); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SetPaymentIntent links a Stripe intent id to a still-pending order.
func (r *orderRepository) SetPaymentIntent(orderID, intentID string) error {
	res, err := r.db.NewUpdate().Model((*models.Order)(nil)).
		Set("\"paymentIntentId\" = ?", intentID).
		Where("id = ?", orderID).
		Where("\"status\" = ?", models.OrderStatusPendingPayment).
		Exec(context.Background())
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrOrderNotPending
	}
	return nil
}

// MarkPaidWithStars confirms payment: ORDERED + paidAt + earn transaction.
func (r *orderRepository) MarkPaidWithStars(orderID, userID string, stars int, title string) error {
	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	res, err := tx.NewUpdate().Model((*models.Order)(nil)).
		Set("\"status\" = ?", models.OrderStatusOrdered).
		Set("\"paidAt\" = ?", time.Now()).
		Where("id = ?", orderID).
		Where("\"status\" = ?", models.OrderStatusPendingPayment).
		Exec(ctx)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrOrderNotPending
	}
	if stars > 0 {
		st := &models.StarTransaction{
			ID:     uuid.New().String(),
			UserID: userID,
			Type:   "earn",
			Title:  title,
			Stars:  stars,
			Date:   time.Now(),
		}
		if _, err = tx.NewInsert().Model(st).Exec(ctx); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// CancelPending moves a still-pending order to CANCELLED.
func (r *orderRepository) CancelPending(orderID string) error {
	res, err := r.db.NewUpdate().Model((*models.Order)(nil)).
		Set("\"status\" = ?", models.OrderStatusCancelled).
		Where("id = ?", orderID).
		Where("\"status\" = ?", models.OrderStatusPendingPayment).
		Exec(context.Background())
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrOrderNotPending
	}
	return nil
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

	type enrichedItem struct {
		ID           string  `json:"id"`
		ProductID    string  `json:"productId"`
		Name         string  `json:"name"`
		Quantity     int     `json:"quantity"`
		Price        float64 `json:"price"`
		ImageURL     *string `json:"imageUrl"`
		Size         *string `json:"size"`
		Type         *string `json:"type"`
		Enhancements *string `json:"enhancements"`
		StarsReward  int     `json:"starsReward"`
	}

	var items []enrichedItem
	if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
		// If unmarshalling fails, treat as empty to avoid transaction failure
		items = []enrichedItem{}
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Insert order
	order := &models.Order{
		ID:            orderID,
		OrderNumber:   orderNumber,
		UserID:        userID,
		LocationID:    locationID,
		Total:         total,
		ServiceFee:    serviceFee,
		Status:        models.OrderStatusOrdered,
		PaymentMethod: paymentMethod,
		PickupTime:    pickupTime,
		CreatedAt:     time.Now(),
	}
	if _, err = tx.NewInsert().Model(order).Exec(ctx); err != nil {
		return 0, err
	}

	// Insert order items
	totalStars := 0
	for _, it := range items {
		enh := models.StringArray{}
		if it.Enhancements != nil && *it.Enhancements != "" {
			// Already comma-joined string, let StringArray handle it via Value()
			// Split by comma to create slice, then StringArray will re-join
			// Simple: keep as single string slice element? Actually StringArray expects slice of strings
			// The service already joined enhancements with "," so we need to split
			// We'll parse it back
			// For simplicity, store as StringArray with one element being the joined string? Better split
			// Let's split by comma
			var parts []string
			// manual split
			s := *it.Enhancements
			start := 0
			for i := 0; i <= len(s); i++ {
				if i == len(s) || s[i] == ',' {
					if i > start {
						parts = append(parts, s[start:i])
					}
					start = i + 1
				}
			}
			enh = parts
		}
		oi := &models.OrderItem{
			ID:           it.ID,
			OrderID:      orderID,
			ProductID:    &it.ProductID,
			ImageURL:     it.ImageURL,
			Name:         it.Name,
			Quantity:     it.Quantity,
			Price:        it.Price,
			Size:         it.Size,
			Type:         it.Type,
			Enhancements: enh,
		}
		if oi.ID == "" {
			oi.ID = uuid.New().String()
		}
		if _, err = tx.NewInsert().Model(oi).Exec(ctx); err != nil {
			return 0, err
		}
		totalStars += it.StarsReward * it.Quantity
	}

	// If payment is via reward or redeemed reward present, don't award stars
	if paymentMethod == "REWARD" || redeemedRewardID != nil {
		totalStars = 0
	}

	// Handle redeemed reward marking
	if redeemedRewardID != nil && *redeemedRewardID != "" {
		if _, err = tx.NewUpdate().Model((*models.RedeemedReward)(nil)).
			Set("\"isClaimed\" = ?", true).
			Where("id = ?", *redeemedRewardID).
			Exec(ctx); err != nil {
			return 0, err
		}
		// Also create a redeem star transaction (negative stars) if reward has cost
		// The reward cost is not directly available here; star handling for redeem is done elsewhere (star service)
		// So we just return 0 stars for this order
	}

	// Award stars if applicable
	if totalStars > 0 {
		st := &models.StarTransaction{
			ID:     uuid.New().String(),
			UserID: userID,
			Type:   "earn",
			Title:  "Order " + orderNumber,
			Stars:  totalStars,
			Date:   time.Now(),
		}
		if _, err = tx.NewInsert().Model(st).Exec(ctx); err != nil {
			return 0, err
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return totalStars, nil
}
