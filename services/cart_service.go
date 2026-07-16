package services

import (
	"errors"
	"log"
	"refer_api/gateway"
	"refer_api/models"
	"refer_api/repository"

	"github.com/google/uuid"
)

// CartService defines the business logic interface for managing the user's shopping cart.
type CartService interface {
	GetCart(userID string) (map[string]interface{}, error)
	AddToCart(userID string, data map[string]interface{}) error
	RemoveFromCart(userID, itemID string) error
	UpdateItemQuantity(userID, itemID string, quantity int) error
	ClearCart(userID string) error
}

type cartService struct {
	cartRepo repository.CartRepository
}

// NewCartService constructs a new CartService.
func NewCartService(cartRepo repository.CartRepository) CartService {
	return &cartService{cartRepo: cartRepo}
}

func (s *cartService) notifyCartUpdate(userID string) {
	log.Printf("[CartService] notifyCartUpdate initiated for user: %s", userID)
	cart, err := s.GetCart(userID)
	if err != nil {
		log.Printf("[CartService] notifyCartUpdate failed to get cart: %v", err)
		return
	}
	if gateway.Instance == nil {
		log.Printf("[CartService] notifyCartUpdate failed: gateway.Instance is nil")
		return
	}
	gateway.Instance.SendCartUpdate(userID, cart)
	log.Printf("[CartService] notifyCartUpdate successfully sent cartUpdate event for user: %s", userID)
}

func (s *cartService) GetCart(userID string) (map[string]interface{}, error) {
	items, err := s.cartRepo.FindAllByUser(userID)
	if err != nil {
		return nil, err
	}

	type MappedCartItem struct {
		ID           string   `json:"id"`
		ProductID    string   `json:"productId"`
		Name         string   `json:"name"`
		ImageURL     *string  `json:"imageUrl"`
		Category     string   `json:"category"`
		Size         string   `json:"size"`
		Type         string   `json:"type"`
		Enhancements []string `json:"enhancements"`
		Quantity     int      `json:"quantity"`
		UnitPrice    float64  `json:"unitPrice"`
		TotalPrice   float64  `json:"totalPrice"`
		StarsReward  int      `json:"starsReward"`
	}

	var mappedItems []MappedCartItem
	var grandTotal float64

	for _, item := range items {
		if item.Product == nil {
			continue
		}

		unitPrice := item.Product.Price

		if item.SizeID != nil && item.Size != nil {
			for _, sizeOpt := range item.Product.SizeOptions {
				if sizeOpt.SizeID == *item.SizeID {
					unitPrice += sizeOpt.Price
					break
				}
			}
		}

		if item.TypeID != nil && item.Type != nil {
			unitPrice += item.Type.Price
		}

		for _, enh := range item.Enhancements {
			unitPrice += enh.Price
		}

		var categoryName string
		if item.Product.Category != nil {
			categoryName = item.Product.Category.Name
		} else {
			categoryName = "All"
		}

		var sizeName string
		if item.Size != nil {
			sizeName = item.Size.Name
		}

		var typeName string
		if item.Type != nil {
			typeName = item.Type.Name
		}

		var enhancementNames []string
		for _, enh := range item.Enhancements {
			enhancementNames = append(enhancementNames, enh.Name)
		}

		totalPrice := unitPrice * float64(item.Quantity)
		grandTotal += totalPrice

		mappedItems = append(mappedItems, MappedCartItem{
			ID:           item.ID,
			ProductID:    item.ProductID,
			Name:         item.Product.Name,
			ImageURL:     item.Product.ImageURL,
			Category:     categoryName,
			Size:         sizeName,
			Type:         typeName,
			Enhancements: enhancementNames,
			Quantity:     item.Quantity,
			UnitPrice:    unitPrice,
			TotalPrice:   totalPrice,
			StarsReward:  item.Product.StarsReward * item.Quantity,
		})
	}

	return map[string]interface{}{
		"items": mappedItems,
		"total": grandTotal,
	}, nil
}

func (s *cartService) AddToCart(userID string, data map[string]interface{}) error {
	productID, _ := data["productId"].(string)
	if productID == "" {
		return errors.New("productId is required")
	}

	var sizeID *string
	if sz, ok := data["sizeId"].(string); ok && sz != "" {
		sizeID = &sz
	}

	var typeID *string
	if tp, ok := data["typeId"].(string); ok && tp != "" {
		typeID = &tp
	}

	qtyFloat, _ := data["quantity"].(float64)
	qty := int(qtyFloat)
	if qty <= 0 {
		qty = 1
	}

	var enhancementIDs []string
	if enhRaw, ok := data["enhancementIds"].([]interface{}); ok {
		for _, e := range enhRaw {
			if eStr, ok := e.(string); ok && eStr != "" {
				enhancementIDs = append(enhancementIDs, eStr)
			}
		}
	}

	var enhancements []models.ProductEnhancement
	if len(enhancementIDs) > 0 {
		var err error
		enhancements, err = s.cartRepo.FindEnhancementsByIDs(enhancementIDs)
		if err != nil {
			return err
		}
	}

	cartItem := &models.CartItem{
		ID:           uuid.New().String(),
		UserID:       userID,
		ProductID:    productID,
		SizeID:       sizeID,
		TypeID:       typeID,
		Quantity:     qty,
		Enhancements: enhancements,
	}

	if err := s.cartRepo.Create(cartItem); err != nil {
		return err
	}

	s.notifyCartUpdate(userID)
	return nil
}

func (s *cartService) RemoveFromCart(userID, itemID string) error {
	if err := s.cartRepo.Delete(itemID, userID); err != nil {
		return err
	}
	s.notifyCartUpdate(userID)
	return nil
}

func (s *cartService) UpdateItemQuantity(userID, itemID string, quantity int) error {
	if quantity <= 0 {
		return s.RemoveFromCart(userID, itemID)
	}

	_, err := s.cartRepo.FindByIDAndUser(itemID, userID)
	if err != nil {
		return errors.New("item not found in cart")
	}

	if err := s.cartRepo.UpdateQuantity(itemID, userID, quantity); err != nil {
		return err
	}

	s.notifyCartUpdate(userID)
	return nil
}

func (s *cartService) ClearCart(userID string) error {
	if err := s.cartRepo.Clear(userID); err != nil {
		return err
	}
	s.notifyCartUpdate(userID)
	return nil
}
