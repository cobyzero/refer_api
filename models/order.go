package models

import (
	"database/sql/driver"
	"errors"
	"strings"
	"time"

	"github.com/uptrace/bun"
)

// OrderStatus defines the lifecycle states of an order.
type OrderStatus string

const (
	OrderStatusOrdered   OrderStatus = "ORDERED"
	OrderStatusPreparing OrderStatus = "PREPARING"
	OrderStatusReady     OrderStatus = "READY"
	OrderStatusCompleted OrderStatus = "COMPLETED"
	OrderStatusCancelled OrderStatus = "CANCELLED"
	// OrderStatusPendingPayment is the order-first flow state: the order
	// exists with its server-computed total, but no money moved yet. It
	// only reaches the kitchen (ORDERED) after POST /orders/:id/confirm
	// verifies the Stripe intent as succeeded.
	OrderStatusPendingPayment OrderStatus = "PENDING_PAYMENT"
)

// StringArray is a custom type for mapping comma-separated strings to a Go slice.
type StringArray []string

// Scan implements the sql.Scanner interface.
func (a *StringArray) Scan(src interface{}) error {
	if src == nil {
		*a = nil
		return nil
	}
	var source []byte
	switch src := src.(type) {
	case []byte:
		source = src
	case string:
		source = []byte(src)
	default:
		return errors.New("invalid type for StringArray scan")
	}
	val := string(source)
	if val == "" {
		*a = []string{}
		return nil
	}
	*a = strings.Split(val, ",")
	return nil
}

// Value implements the driver.Valuer interface.
func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	return strings.Join(a, ","), nil
}

// Order represents an order record mapped to the orders table.
type Order struct {
	bun.BaseModel `bun:"table:orders"`

	ID            string      `bun:"id,pk,type:char(36)" json:"id"`
	OrderNumber   string      `bun:"orderNumber,notnull" json:"orderNumber"`
	UserID        string      `bun:"userId,notnull" json:"userId"`
	LocationID    string      `bun:"locationId,notnull" json:"locationId"`
	Total         float64     `bun:"total,notnull" json:"total"`
	ServiceFee    float64     `bun:"serviceFee,default:0" json:"serviceFee"`
	Status        OrderStatus `bun:"status,default:'ORDERED'" json:"status"`
	PaymentMethod string      `bun:"paymentMethod,notnull" json:"paymentMethod"`
	// PaymentIntentID links the Stripe intent created for this order
	// (order-first flow). PaidAt is set on confirm.
	PaymentIntentID *string    `bun:"paymentIntentId" json:"paymentIntentId,omitempty"`
	PaidAt          *time.Time `bun:"paidAt" json:"paidAt,omitempty"`
	PickupTime    *time.Time  `bun:"pickupTime" json:"pickupTime"`
	CreatedAt     time.Time   `bun:"createdAt" json:"createdAt"`

	User  *User       `bun:"rel:belongs-to,join:userId=id" json:"user,omitempty"`
	Items []OrderItem `bun:"rel:has-many,join:id=orderId" json:"items"`
}

// OrderItem represents an item within an order.
type OrderItem struct {
	bun.BaseModel `bun:"table:order_items"`

	ID           string      `bun:"id,pk,type:char(36)" json:"id"`
	OrderID      string      `bun:"orderId,notnull" json:"orderId"`
	ProductID    *string     `bun:"productId" json:"productId"`
	ImageURL     *string     `bun:"imageUrl" json:"imageUrl"`
	Name         string      `bun:"name,notnull" json:"name"`
	Quantity     int         `bun:"quantity,notnull" json:"quantity"`
	Price        float64     `bun:"price,notnull" json:"price"`
	Size         *string     `bun:"size" json:"size"`
	Type         *string     `bun:"type" json:"type"`
	Enhancements StringArray `bun:"enhancements,type:text" json:"enhancements"`
}
