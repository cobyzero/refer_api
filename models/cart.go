package models

import "github.com/uptrace/bun"

// CartItem represents a single item stored in a user's shopping cart.
type CartItem struct {
	bun.BaseModel `bun:"table:cart_items"`

	ID        string  `bun:"id,pk,type:char(36)" json:"id"`
	UserID    string  `bun:"userId,notnull" json:"userId"`
	ProductID string  `bun:"productId,notnull" json:"productId"`
	SizeID    *string `bun:"sizeId" json:"sizeId"`
	TypeID    *string `bun:"typeId" json:"typeId"`
	Quantity  int     `bun:"quantity,default:1" json:"quantity"`

	User         *User                `bun:"rel:belongs-to,join:userId=id" json:"-"`
	Product      *Product             `bun:"rel:belongs-to,join:productId=id" json:"product,omitempty"`
	Size         *ProductSize         `bun:"rel:belongs-to,join:sizeId=id" json:"size,omitempty"`
	Type         *ProductType         `bun:"rel:belongs-to,join:typeId=id" json:"type,omitempty"`
	Enhancements []ProductEnhancement `bun:"m2m:cart_item_enhancements,join:CartItem=ProductEnhancement" json:"enhancements"`
}

// CartItemEnhancement is the pivot struct for the cart_item_enhancements many-to-many join.
type CartItemEnhancement struct {
	bun.BaseModel `bun:"table:cart_item_enhancements"`

	CartItemID           string              `bun:"cartItemId,pk"`
	CartItem             *CartItem           `bun:"rel:belongs-to,join:cartItemId=id"`
	ProductEnhancementID string              `bun:"enhancementId,pk"`
	ProductEnhancement   *ProductEnhancement `bun:"rel:belongs-to,join:enhancementId=id"`
}
