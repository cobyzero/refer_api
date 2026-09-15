package services

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"refer_api/gateway"
	"refer_api/models"
	"refer_api/repository"

	"github.com/google/uuid"
)

// OrderService defines the business logic interface for managing orders.
//
// Order-first flow (Fix #2): STRIPE orders are created PENDING_PAYMENT with
// server-computed totals, charged via POST /payments/create-payment-intent
// {orderId}, and only reach the kitchen on POST /orders/:id/confirm after
// Stripe reports the intent as succeeded. REWARD claims settle immediately.
type OrderService interface {
	Create(userID string, body map[string]interface{}) (*models.Order, error)
	FindAll() ([]models.Order, error)
	FindAllByUser(userID string) ([]models.Order, error)
	// FindByIDForUser returns the order only when it belongs to userID
	// (uniform "not found" otherwise, so IDs aren't enumerable across users).
	FindByIDForUser(orderID, userID string) (*models.Order, error)
	// AttachPaymentIntent links a Stripe intent to a pending order (owner only).
	AttachPaymentIntent(orderID, userID, intentID string) (*models.Order, error)
	// ConfirmOrder verifies the Stripe intent and settles the order
	// (owner or ADMIN). Awards stars and notifies kitchen + buyer.
	ConfirmOrder(orderID, userID string, roles []string) (*models.Order, error)
	// CancelOrder voids a pending order (owner or ADMIN).
	CancelOrder(orderID, userID string, roles []string) (*models.Order, error)
	UpdateStatus(orderID string, status string) (*models.Order, error)
}

// ErrInvalidOrder marks client-correctable order errors (unknown product,
// bad quantity, invalid location/method...). Handlers map it to 400;
// anything else is a 500.
var ErrInvalidOrder = errors.New("invalid order")

type orderService struct {
	orderRepo    repository.OrderRepository
	productRepo  repository.ProductRepository
	configRepo   repository.ConfigRepository
	locationRepo repository.LocationRepository
	starRepo     repository.StarRepository
}

// NewOrderService constructs a new OrderService. configRepo/locationRepo/
// starRepo may be nil in unit tests that only exercise FindAll paths; Create
// requires them and fails closed otherwise.
func NewOrderService(orderRepo repository.OrderRepository, productRepo repository.ProductRepository, configRepo repository.ConfigRepository, locationRepo repository.LocationRepository, starRepo repository.StarRepository) OrderService {
	return &orderService{
		orderRepo:    orderRepo,
		productRepo:  productRepo,
		configRepo:   configRepo,
		locationRepo: locationRepo,
		starRepo:     starRepo,
	}
}

// Server-side pricing bounds (Fix #2).
const (
	maxOrderItems      = 50
	maxQuantityPerItem = 20
)

// Allowed payment methods (refer_app sends STRIPE or REWARD).
var allowedPaymentMethods = map[string]bool{
	"STRIPE": true,
	"REWARD": true,
}

func round2(f float64) float64 {
	return math.Round(f*100) / 100
}

// pricingRates resolves tax/fee from app_settings (same source as GET
// /config, via the 30s cache) with safe defaults when unset or unreadable.
func (s *orderService) pricingRates() (taxEnabled bool, taxRate, feeRate float64) {
	taxEnabled, taxRate, feeRate = DefaultTaxEnabled, DefaultTaxRate, DefaultServiceFeeRate
	if s.configRepo == nil {
		return taxEnabled, taxRate, feeRate
	}
	settings, err := NewConfigService(s.configRepo).GetSettings()
	if err != nil {
		return taxEnabled, taxRate, feeRate
	}
	if v, err := strconv.ParseBool(strings.TrimSpace(settings["tax_enabled"])); err == nil {
		taxEnabled = v
	}
	if v, err := strconv.ParseFloat(strings.TrimSpace(settings["tax_rate"]), 64); err == nil && v >= 0 && v <= 1 {
		taxRate = v
	}
	if v, err := strconv.ParseFloat(strings.TrimSpace(settings["service_fee_rate"]), 64); err == nil && v >= 0 && v <= 1 {
		feeRate = v
	}
	return taxEnabled, taxRate, feeRate
}

func invalidOrder(format string, args ...interface{}) error {
	return fmt.Errorf("%w: %s", ErrInvalidOrder, fmt.Sprintf(format, args...))
}

// EnrichedItem matches the item schema expected by the database stored procedure sp_create_order.
type EnrichedItem struct {
	ID           string   `json:"id"`
	ProductID    string   `json:"productId"`
	Name         string   `json:"name"`
	Quantity     int      `json:"quantity"`
	Price        float64  `json:"price"`
	ImageURL     *string  `json:"imageUrl"`
	Size         *string  `json:"size"`
	Type         *string  `json:"type"`
	Enhancements *string  `json:"enhancements"`
	StarsReward  int      `json:"starsReward"`
}

// parsePickupTime accepts RFC3339 with offset (current apps send UTC with `Z`)
// and, for older clients, offset-less local ISO strings interpreted in server time.
func parsePickupTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	for _, layout := range []string{
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("invalid pickupTime")
}

// validateRedeemedReward ensures the claim belongs to the buyer and is still
// unclaimed. Without this, any user could consume someone else's reward by ID.
func (s *orderService) validateRedeemedReward(userID, rrID string) (*models.RedeemedReward, error) {
	if s.starRepo == nil {
		return nil, invalidOrder("rewards unavailable")
	}
	rr, err := s.starRepo.FindRedeemedRewardByID(rrID)
	if err != nil || rr == nil {
		return nil, invalidOrder("invalid redeemed reward")
	}
	if rr.UserID != userID {
		return nil, invalidOrder("invalid redeemed reward")
	}
	if rr.IsClaimed {
		return nil, invalidOrder("reward already claimed")
	}
	return rr, nil
}

// priceItem resolves one client line against the DB catalog and returns the
// server-priced line. Names AND ids are accepted for size/type/enhancements
// (the app sends names; sockets carry ids). unit = base + size + type +
// Σ enhancements — the same formula as refer_app's _calculateCurrentPrice.
func priceItem(product *models.Product, sizeVal, typeVal string, enhVals []string) (unit float64, canonSize, canonType *string, canonEnh []string) {
	unit = product.Price
	if sizeVal != "" {
		for _, so := range product.SizeOptions {
			if so.Size.ID == sizeVal || so.Size.Name == sizeVal {
				unit += so.Price
				name := so.Size.Name
				canonSize = &name
				break
			}
		}
	}
	if typeVal != "" {
		for _, opt := range product.Options {
			if opt.ID == typeVal || opt.Name == typeVal {
				unit += opt.Price
				name := opt.Name
				canonType = &name
				break
			}
		}
	}
	for _, ev := range enhVals {
		for _, enh := range product.Enhancements {
			if enh.ID == ev || enh.Name == ev {
				unit += enh.Price
				canonEnh = append(canonEnh, enh.Name)
				break
			}
		}
	}
	return unit, canonSize, canonType, canonEnh
}

func (s *orderService) Create(userID string, body map[string]interface{}) (*models.Order, error) {
	if s.configRepo == nil || s.locationRepo == nil || s.starRepo == nil || s.productRepo == nil {
		return nil, errors.New("order service misconfigured")
	}

	locationID, _ := body["locationId"].(string)
	if locationID == "" {
		return nil, invalidOrder("locationId is required")
	}
	// Only active pickup points can receive orders (FindByID filters isActive).
	if _, err := s.locationRepo.FindByID(locationID); err != nil {
		return nil, invalidOrder("invalid location")
	}

	paymentMethodRaw, _ := body["paymentMethod"].(string)
	paymentMethod := strings.ToUpper(strings.TrimSpace(paymentMethodRaw))
	if !allowedPaymentMethods[paymentMethod] {
		return nil, invalidOrder("unsupported payment method")
	}

	var pickupTime *time.Time
	if pickupStr, ok := body["pickupTime"].(string); ok && pickupStr != "" {
		if parsed, err := parsePickupTime(pickupStr); err == nil {
			pickupTime = &parsed
		}
	}

	var redeemedRewardID *string
	if rrID, ok := body["redeemedRewardId"].(string); ok && rrID != "" {
		redeemedRewardID = &rrID
	}

	// Client-supplied amounts are informational only (fraud telemetry below):
	// the charged total is ALWAYS recomputed from the DB catalog (Fix #2).
	clientTotal, _ := body["total"].(float64)

	var enrichedItems []EnrichedItem
	var subtotal float64

	if paymentMethod == "REWARD" {
		// Reward-claim flow: the app sends a synthetic 'reward-item' line.
		// Ignore client lines entirely; the single zero-price line comes
		// from the validated claim.
		if redeemedRewardID == nil {
			return nil, invalidOrder("redeemedRewardId is required")
		}
		rr, err := s.validateRedeemedReward(userID, *redeemedRewardID)
		if err != nil {
			return nil, err
		}
		enrichedItems = []EnrichedItem{{
			ID:          uuid.New().String(),
			ProductID:   "",
			Name:        rr.RewardTitle,
			Quantity:    1,
			Price:       0,
			StarsReward: 0,
		}}
	} else {
		itemsRaw, ok := body["items"].([]interface{})
		if !ok || len(itemsRaw) == 0 {
			return nil, invalidOrder("items are required")
		}
		if len(itemsRaw) > maxOrderItems {
			return nil, invalidOrder("too many items")
		}

		// No STRIPE+redeemed combo: reward claims settle immediately via the
		// REWARD flow below. Accepting a claim on an unpaid order would need
		// the claim id persisted past creation with no column for it.
		if redeemedRewardID != nil {
			return nil, invalidOrder("redeemed rewards require REWARD payment")
		}

		for i, itemVal := range itemsRaw {
			itemMap, ok := itemVal.(map[string]interface{})
			if !ok {
				return nil, invalidOrder("invalid item at index %d", i)
			}

			pID, _ := itemMap["productId"].(string)
			if pID == "" || pID == "reward-item" {
				return nil, invalidOrder("invalid product at index %d", i)
			}
			qtyFloat, _ := itemMap["quantity"].(float64)
			qty := int(qtyFloat)
			if qty < 1 || qty > maxQuantityPerItem {
				return nil, invalidOrder("invalid quantity at index %d", i)
			}

			product, err := s.productRepo.FindByID(pID)
			if err != nil || product == nil {
				return nil, invalidOrder("product unavailable at index %d", i)
			}
			if !product.IsAvailable {
				return nil, invalidOrder("product unavailable at index %d", i)
			}

			sizeVal, _ := itemMap["size"].(string)
			typeVal, _ := itemMap["type"].(string)
			var enhVals []string
			if enhSlice, ok := itemMap["enhancements"].([]interface{}); ok {
				for _, e := range enhSlice {
					if eStr, ok := e.(string); ok && eStr != "" {
						enhVals = append(enhVals, eStr)
					}
				}
			}

			unit, canonSize, canonType, canonEnh := priceItem(product, sizeVal, typeVal, enhVals)
			// Unknown modifiers are rejected: silently dropping them would
			// charge less than displayed, accepting them blindly more.
			if sizeVal != "" && canonSize == nil {
				return nil, invalidOrder("invalid size at index %d", i)
			}
			if typeVal != "" && canonType == nil {
				return nil, invalidOrder("invalid type at index %d", i)
			}
			if len(enhVals) != len(canonEnh) {
				return nil, invalidOrder("invalid enhancement at index %d", i)
			}

			lineTotal := round2(unit * float64(qty))
			subtotal += lineTotal

			var enhJoined *string
			if len(canonEnh) > 0 {
				joined := strings.Join(canonEnh, ",")
				enhJoined = &joined
			}
			enrichedItems = append(enrichedItems, EnrichedItem{
				ID:           uuid.New().String(),
				ProductID:    product.ID,
				Name:         product.Name,
				Quantity:     qty,
				Price:        lineTotal,
				ImageURL:     product.ImageURL,
				Size:         canonSize,
				Type:         canonType,
				Enhancements: enhJoined,
				StarsReward:  product.StarsReward,
			})
		}
	}

	// Server-side totals: subtotal + tax (if enabled) + service fee, all from
	// app_settings — the same rates refer_app caches from GET /config.
	taxEnabled, taxRate, feeRate := s.pricingRates()
	tax := 0.0
	if taxEnabled {
		tax = subtotal * taxRate
	}
	serviceFee := round2(subtotal * feeRate)
	total := round2(subtotal + tax + serviceFee)
	if paymentMethod == "REWARD" {
		total, serviceFee = 0, 0
	}

	if math.Abs(clientTotal-total) > 0.01 && paymentMethod != "REWARD" {
		// Don't reject (client may run a stale catalog/tax version), but
		// leave a trail: systematic mismatches mean tampering or skew.
		fmt.Printf("[order] total mismatch user=%s client=%.2f server=%.2f items=%d\n", userID, clientTotal, total, len(enrichedItems))
	}

	orderID := uuid.New().String()
	n, err := rand.Int(rand.Reader, big.NewInt(9000))
	if err != nil {
		return nil, err
	}
	orderNumber := fmt.Sprintf("ER-%d", 1000+n.Int64())

	if paymentMethod == "REWARD" {
		// Immediate flow: no money moves, the claim is consumed now.
		itemsJSONBytes, err := json.Marshal(enrichedItems)
		if err != nil {
			return nil, err
		}
		if _, err := s.orderRepo.CreateOrderViaSP(
			orderID,
			userID,
			locationID,
			orderNumber,
			total,
			serviceFee,
			paymentMethod,
			pickupTime,
			string(itemsJSONBytes),
			redeemedRewardID,
		); err != nil {
			return nil, err
		}
		fmt.Printf("[order] reward claimed %s user=%s\n", orderNumber, userID)
		fullOrder, err := s.orderRepo.FindByID(orderID)
		if err != nil {
			return nil, err
		}
		if gateway.Instance != nil {
			gateway.Instance.SendNewOrderNotification(fullOrder)
		}
		return fullOrder, nil
	}

	// Order-first flow: persist PENDING_PAYMENT with server totals. No stars
	// and no kitchen notification — both happen on confirm after Stripe
	// reports the intent as succeeded.
	items := make([]models.OrderItem, 0, len(enrichedItems))
	for _, e := range enrichedItems {
		pid := e.ProductID
		var enh models.StringArray
		if e.Enhancements != nil && *e.Enhancements != "" {
			enh = strings.Split(*e.Enhancements, ",")
		}
		items = append(items, models.OrderItem{
			ID:           e.ID,
			OrderID:      orderID,
			ProductID:    &pid,
			ImageURL:     e.ImageURL,
			Name:         e.Name,
			Quantity:     e.Quantity,
			Price:        e.Price,
			Size:         e.Size,
			Type:         e.Type,
			Enhancements: enh,
		})
	}
	pending := &models.Order{
		ID:            orderID,
		OrderNumber:   orderNumber,
		UserID:        userID,
		LocationID:    locationID,
		Total:         total,
		ServiceFee:    serviceFee,
		Status:        models.OrderStatusPendingPayment,
		PaymentMethod: paymentMethod,
		PickupTime:    pickupTime,
		CreatedAt:     time.Now(),
	}
	if err := s.orderRepo.CreatePendingOrder(pending, items); err != nil {
		return nil, err
	}
	fmt.Printf("[order] pending %s user=%s total=%.2f items=%d\n", orderNumber, userID, total, len(items))

	fullOrder, err := s.orderRepo.FindByID(orderID)
	if err != nil {
		return nil, err
	}
	return fullOrder, nil
}

func isAdminRole(roles []string) bool {
	for _, r := range roles {
		if r == "ADMIN" {
			return true
		}
	}
	return false
}

// loadOwnedForPayment loads an order for payment-flow transitions: owner or
// ADMIN, uniform "not found" otherwise.
func (s *orderService) loadOwnedForPayment(orderID, userID string, roles []string) (*models.Order, error) {
	if orderID == "" || userID == "" {
		return nil, errors.New("order not found")
	}
	order, err := s.orderRepo.FindByID(orderID)
	if err != nil || order == nil {
		return nil, errors.New("order not found")
	}
	if order.UserID != userID && !isAdminRole(roles) {
		return nil, errors.New("order not found")
	}
	return order, nil
}

// AttachPaymentIntent links a Stripe intent to a pending order (owner only:
// the intent will move real money, so admins can't attach it to strangers'
// orders from this path — they can still confirm).
func (s *orderService) AttachPaymentIntent(orderID, userID, intentID string) (*models.Order, error) {
	order, err := s.FindByIDForUser(orderID, userID)
	if err != nil {
		return nil, err
	}
	if order.Status != models.OrderStatusPendingPayment {
		return nil, invalidOrder("order cannot be charged")
	}
	if order.Total <= 0 {
		return nil, invalidOrder("order total must be greater than zero")
	}
	if intentID == "" {
		return nil, invalidOrder("payment intent is required")
	}
	if err := s.orderRepo.SetPaymentIntent(orderID, intentID); err != nil {
		if errors.Is(err, repository.ErrOrderNotPending) {
			return nil, invalidOrder("order cannot be charged")
		}
		return nil, err
	}
	return s.orderRepo.FindByID(orderID)
}

// computeEarnStars re-sums StarsReward from the current catalog at confirm
// time (products may have changed since creation; missing ones earn 0).
func (s *orderService) computeEarnStars(items []models.OrderItem) int {
	total := 0
	for _, it := range items {
		if it.ProductID == nil || *it.ProductID == "" {
			continue
		}
		product, err := s.productRepo.FindByID(*it.ProductID)
		if err != nil || product == nil {
			continue
		}
		total += product.StarsReward * it.Quantity
	}
	return total
}

// ConfirmOrder settles a pending order after verifying the Stripe intent as
// succeeded. Client claims of "I paid" are never trusted on their own.
func (s *orderService) ConfirmOrder(orderID, userID string, roles []string) (*models.Order, error) {
	order, err := s.loadOwnedForPayment(orderID, userID, roles)
	if err != nil {
		return nil, err
	}
	if order.Status != models.OrderStatusPendingPayment {
		return nil, invalidOrder("order cannot be confirmed")
	}
	intentID := ""
	if order.PaymentIntentID != nil {
		intentID = *order.PaymentIntentID
	}
	if intentID == "" {
		return nil, invalidOrder("payment not initialized")
	}
	verified, err := VerifyPaymentSucceeded(intentID)
	if err != nil {
		return nil, err
	}
	if !verified {
		return nil, invalidOrder("payment not completed")
	}
	stars := s.computeEarnStars(order.Items)
	if err := s.orderRepo.MarkPaidWithStars(orderID, order.UserID, stars, "Order "+order.OrderNumber); err != nil {
		if errors.Is(err, repository.ErrOrderNotPending) {
			return nil, invalidOrder("order cannot be confirmed")
		}
		return nil, err
	}
	fmt.Printf("[order] confirmed %s user=%s total=%.2f stars=%d\n", order.OrderNumber, order.UserID, order.Total, stars)

	fullOrder, err := s.orderRepo.FindByID(orderID)
	if err != nil {
		return nil, err
	}
	if gateway.Instance != nil {
		gateway.Instance.SendNewOrderNotification(fullOrder)
		gateway.Instance.SendOrderStatusUpdate(fullOrder.UserID, fullOrder)
	}
	return fullOrder, nil
}

// CancelOrder voids a pending order (owner or ADMIN). The Stripe intent, if
// any, is voided best-effort — an unconfirmed intent moves no money, so a
// failure here never fails the cancellation.
func (s *orderService) CancelOrder(orderID, userID string, roles []string) (*models.Order, error) {
	order, err := s.loadOwnedForPayment(orderID, userID, roles)
	if err != nil {
		return nil, err
	}
	if order.Status != models.OrderStatusPendingPayment {
		return nil, invalidOrder("order cannot be cancelled")
	}
	if order.PaymentIntentID != nil && *order.PaymentIntentID != "" {
		if err := CancelPaymentIntent(*order.PaymentIntentID); err != nil {
			fmt.Printf("[order] cancel %s: stripe void best-effort failed: %v\n", order.OrderNumber, err)
		}
	}
	if err := s.orderRepo.CancelPending(orderID); err != nil {
		if errors.Is(err, repository.ErrOrderNotPending) {
			return nil, invalidOrder("order cannot be cancelled")
		}
		return nil, err
	}
	fmt.Printf("[order] cancelled %s user=%s\n", order.OrderNumber, order.UserID)
	return s.orderRepo.FindByID(orderID)
}

// FindByIDForUser scopes a single order to its owner (used by payments to
// charge the server total instead of a client-supplied amount).
func (s *orderService) FindByIDForUser(orderID, userID string) (*models.Order, error) {
	if orderID == "" || userID == "" {
		return nil, errors.New("order not found")
	}
	order, err := s.orderRepo.FindByID(orderID)
	if err != nil || order == nil || order.UserID != userID {
		return nil, errors.New("order not found")
	}
	return order, nil
}

func (s *orderService) FindAll() ([]models.Order, error) {
	orders, err := s.orderRepo.FindAll()
	if err != nil {
		return nil, err
	}
	if orders == nil {
		orders = []models.Order{}
	}
	return orders, nil
}

func (s *orderService) FindAllByUser(userID string) ([]models.Order, error) {
	orders, err := s.orderRepo.FindAllByUser(userID)
	if err != nil {
		return nil, err
	}
	if orders == nil {
		orders = []models.Order{}
	}
	return orders, nil
}

func (s *orderService) UpdateStatus(orderID string, status string) (*models.Order, error) {
	err := s.orderRepo.UpdateStatus(orderID, models.OrderStatus(status))
	if err != nil {
		return nil, err
	}

	fullOrder, err := s.orderRepo.FindByID(orderID)
	if err != nil {
		return nil, err
	}

	if gateway.Instance != nil {
		gateway.Instance.SendOrderStatusUpdate(fullOrder.UserID, fullOrder)
	}

	return fullOrder, nil
}
