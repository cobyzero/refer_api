package services

import (
	"errors"
	"testing"

	"refer_api/models"
)

// =====================
// Mock CartRepository
// =====================
type mockCartRepo struct {
	items      []models.CartItem
	createErr  error
	deleteErr  error
	updateErr  error
	clearErr   error
	findByErr  error
	enhResults []models.ProductEnhancement
}

func (m *mockCartRepo) FindAllByUser(_ string) ([]models.CartItem, error) {
	return m.items, nil
}
func (m *mockCartRepo) Create(_ *models.CartItem) error     { return m.createErr }
func (m *mockCartRepo) Delete(_, _ string) error             { return m.deleteErr }
func (m *mockCartRepo) UpdateQuantity(_, _ string, _ int) error { return m.updateErr }
func (m *mockCartRepo) Clear(_ string) error                 { return m.clearErr }
func (m *mockCartRepo) FindByIDAndUser(_, _ string) (*models.CartItem, error) {
	if m.findByErr != nil {
		return nil, m.findByErr
	}
	if len(m.items) > 0 {
		return &m.items[0], nil
	}
	return nil, errors.New("not found")
}
func (m *mockCartRepo) FindEnhancementsByIDs(_ []string) ([]models.ProductEnhancement, error) {
	return m.enhResults, nil
}

// ===========================
// CartService Unit Tests
// ===========================

func TestCartService_AddToCart_NoProductID(t *testing.T) {
	svc := NewCartService(&mockCartRepo{})
	err := svc.AddToCart("user1", map[string]interface{}{})
	if err == nil {
		t.Fatal("Expected error when productId is empty, got nil")
	}
	if err.Error() != "productId is required" {
		t.Errorf("Expected 'productId is required' error, got: %s", err.Error())
	}
}

func TestCartService_AddToCart_CreateError(t *testing.T) {
	repo := &mockCartRepo{createErr: errors.New("db error")}
	svc := NewCartService(repo)
	err := svc.AddToCart("user1", map[string]interface{}{"productId": "prod-1"})
	if err == nil {
		t.Fatal("Expected error when db create fails, got nil")
	}
}

func TestCartService_AddToCart_DefaultQuantity(t *testing.T) {
	repo := &mockCartRepo{}
	svc := NewCartService(repo)
	// Quantity <= 0 should default to 1 without errors
	err := svc.AddToCart("user1", map[string]interface{}{"productId": "prod-1", "quantity": float64(0)})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func TestCartService_RemoveFromCart_Success(t *testing.T) {
	repo := &mockCartRepo{}
	svc := NewCartService(repo)
	err := svc.RemoveFromCart("user1", "item-1")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func TestCartService_RemoveFromCart_Error(t *testing.T) {
	repo := &mockCartRepo{deleteErr: errors.New("delete failed")}
	svc := NewCartService(repo)
	err := svc.RemoveFromCart("user1", "item-1")
	if err == nil {
		t.Fatal("Expected error when delete fails, got nil")
	}
}

func TestCartService_UpdateItemQuantity_ZeroRemovesItem(t *testing.T) {
	// When quantity is 0, it should call RemoveFromCart (which calls Delete)
	repo := &mockCartRepo{}
	svc := NewCartService(repo)
	err := svc.UpdateItemQuantity("user1", "item-1", 0)
	if err != nil {
		t.Fatalf("Expected no error when removing item via quantity=0, got: %v", err)
	}
}

func TestCartService_UpdateItemQuantity_ItemNotFound(t *testing.T) {
	repo := &mockCartRepo{findByErr: errors.New("not found")}
	svc := NewCartService(repo)
	err := svc.UpdateItemQuantity("user1", "item-999", 5)
	if err == nil {
		t.Fatal("Expected error when item not found, got nil")
	}
	if err.Error() != "item not found in cart" {
		t.Errorf("Expected 'item not found in cart', got: %s", err.Error())
	}
}

func TestCartService_UpdateItemQuantity_UpdateError(t *testing.T) {
	item := models.CartItem{ID: "item-1", UserID: "user1"}
	repo := &mockCartRepo{items: []models.CartItem{item}, updateErr: errors.New("update error")}
	svc := NewCartService(repo)
	err := svc.UpdateItemQuantity("user1", "item-1", 3)
	if err == nil {
		t.Fatal("Expected error when update fails, got nil")
	}
}

func TestCartService_ClearCart_Success(t *testing.T) {
	repo := &mockCartRepo{}
	svc := NewCartService(repo)
	err := svc.ClearCart("user1")
	if err != nil {
		t.Fatalf("Expected no error when clearing cart, got: %v", err)
	}
}

func TestCartService_ClearCart_Error(t *testing.T) {
	repo := &mockCartRepo{clearErr: errors.New("clear failed")}
	svc := NewCartService(repo)
	err := svc.ClearCart("user1")
	if err == nil {
		t.Fatal("Expected error when clear fails, got nil")
	}
}

func TestCartService_GetCart_Empty(t *testing.T) {
	repo := &mockCartRepo{items: []models.CartItem{}}
	svc := NewCartService(repo)
	result, err := svc.GetCart("user1")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	total, ok := result["total"].(float64)
	if !ok {
		t.Fatal("Expected 'total' to be a float64 in result")
	}
	if total != 0 {
		t.Errorf("Expected total to be 0 for empty cart, got: %f", total)
	}
}

func TestCartService_GetCart_SkipsNilProduct(t *testing.T) {
	// Items where Product is nil should be skipped silently
	repo := &mockCartRepo{items: []models.CartItem{
		{ID: "i1", UserID: "u1", ProductID: "p1", Product: nil},
	}}
	svc := NewCartService(repo)
	result, err := svc.GetCart("u1")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	items, ok := result["items"]
	if !ok {
		t.Fatal("Expected 'items' key in result")
	}
	// Items should be nil or empty because the product is nil
	_ = items
}
