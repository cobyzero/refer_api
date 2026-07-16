package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"refer_api/gateway"
	"refer_api/models"
	"refer_api/repository"

	"github.com/google/uuid"
)

// OrderService defines the business logic interface for managing orders.
type OrderService interface {
	Create(userID string, body map[string]interface{}) (*models.Order, error)
	FindAll() ([]models.Order, error)
	FindAllByUser(userID string) ([]models.Order, error)
	UpdateStatus(orderID string, status string) (*models.Order, error)
}

type orderService struct {
	orderRepo   repository.OrderRepository
	productRepo repository.ProductRepository
}

// NewOrderService constructs a new OrderService.
func NewOrderService(orderRepo repository.OrderRepository, productRepo repository.ProductRepository) OrderService {
	return &orderService{
		orderRepo:   orderRepo,
		productRepo: productRepo,
	}
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

func (s *orderService) Create(userID string, body map[string]interface{}) (*models.Order, error) {
	locationID, _ := body["locationId"].(string)
	if locationID == "" {
		return nil, errors.New("locationId is required")
	}

	total, _ := body["total"].(float64)
	serviceFee, _ := body["serviceFee"].(float64)
	paymentMethod, _ := body["paymentMethod"].(string)

	var pickupTime *time.Time
	if pickupStr, ok := body["pickupTime"].(string); ok && pickupStr != "" {
		parsed, err := time.Parse(time.RFC3339, pickupStr)
		if err == nil {
			pickupTime = &parsed
		}
	}

	var redeemedRewardID *string
	if rrID, ok := body["redeemedRewardId"].(string); ok && rrID != "" {
		redeemedRewardID = &rrID
	}

	itemsRaw, ok := body["items"].([]interface{})
	if !ok || len(itemsRaw) == 0 {
		return nil, errors.New("items are required")
	}

	var enrichedItems []EnrichedItem
	for _, itemVal := range itemsRaw {
		itemMap, ok := itemVal.(map[string]interface{})
		if !ok {
			continue
		}

		pID, _ := itemMap["productId"].(string)
		name, _ := itemMap["name"].(string)
		qtyFloat, _ := itemMap["quantity"].(float64)
		qty := int(qtyFloat)
		price, _ := itemMap["price"].(float64)

		var imgURL *string
		if url, ok := itemMap["imageUrl"].(string); ok && url != "" {
			imgURL = &url
		}

		var sizeVal *string
		if sz, ok := itemMap["size"].(string); ok && sz != "" {
			sizeVal = &sz
		}

		var typeVal *string
		if tp, ok := itemMap["type"].(string); ok && tp != "" {
			typeVal = &tp
		}

		var enhancementsVal *string
		if enhSlice, ok := itemMap["enhancements"].([]interface{}); ok {
			var enhStrings []string
			for _, e := range enhSlice {
				if eStr, ok := e.(string); ok {
					enhStrings = append(enhStrings, eStr)
				}
			}
			if len(enhStrings) > 0 {
				joined := strings.Join(enhStrings, ",")
				enhancementsVal = &joined
			}
		}

		starsReward := 0
		if paymentMethod != "REWARD" && redeemedRewardID == nil {
			product, err := s.productRepo.FindByID(pID)
			if err == nil && product != nil {
				starsReward = product.StarsReward
			}
		}

		enrichedItems = append(enrichedItems, EnrichedItem{
			ID:           uuid.New().String(),
			ProductID:    pID,
			Name:         name,
			Quantity:     qty,
			Price:        price,
			ImageURL:     imgURL,
			Size:         sizeVal,
			Type:         typeVal,
			Enhancements: enhancementsVal,
			StarsReward:  starsReward,
		})
	}

	itemsJSONBytes, err := json.Marshal(enrichedItems)
	if err != nil {
		return nil, err
	}

	orderID := uuid.New().String()
	rand.Seed(time.Now().UnixNano())
	orderNumber := fmt.Sprintf("ER-%d", 1000+rand.Intn(9000))

	totalStars, err := s.orderRepo.CreateOrderViaSP(
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
	)
	if err != nil {
		return nil, err
	}

	fmt.Printf("[sp_create_order] Order %s created — %d stars awarded to user %s\n", orderNumber, totalStars, userID)

	fullOrder, err := s.orderRepo.FindByID(orderID)
	if err != nil {
		return nil, err
	}

	if gateway.Instance != nil {
		gateway.Instance.SendNewOrderNotification(fullOrder)
	}

	return fullOrder, nil
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
