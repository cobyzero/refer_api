package models

import (
	"time"

	"github.com/uptrace/bun"
)

type Reward struct {
	bun.BaseModel `bun:"table:rewards"`

	ID            string  `bun:"id,pk,type:char(36)" json:"id"`
	Title         string  `bun:"title,notnull" json:"title"`
	Description   string  `bun:"description,notnull" json:"description"`
	StarsRequired int     `bun:"starsRequired,notnull" json:"starsRequired"`
	ImageURL      *string `bun:"imageUrl" json:"imageUrl"`
	Icon          *string `bun:"icon" json:"icon"`
	ProductID     *string `bun:"productId" json:"productId"`
	IsActive      bool    `bun:"isActive,default:true" json:"isActive"`
}

type StarTransaction struct {
	bun.BaseModel `bun:"table:star_transactions"`

	ID     string    `bun:"id,pk,type:char(36)" json:"id"`
	UserID string    `bun:"userId,notnull" json:"userId"`
	Type   string    `bun:"type,notnull" json:"type"` // "earn" or "redeem"
	Title  string    `bun:"title,notnull" json:"title"`
	Stars  int       `bun:"stars,notnull" json:"stars"`
	Date   time.Time `bun:"date" json:"date"`

	User *User `bun:"rel:belongs-to,join:userId=id" json:"user,omitempty"`
}

type RedeemedReward struct {
	bun.BaseModel `bun:"table:redeemed_rewards"`

	ID          string    `bun:"id,pk,type:char(36)" json:"id"`
	UserID      string    `bun:"userId,notnull" json:"userId"`
	RewardID    string    `bun:"rewardId,notnull" json:"rewardId"`
	RewardTitle string    `bun:"rewardTitle,notnull" json:"rewardTitle"`
	IsClaimed   bool      `bun:"isClaimed,default:false" json:"isClaimed"`
	OrderID     *string   `bun:"orderId" json:"orderId"`
	CreatedAt   time.Time `bun:"createdAt" json:"createdAt"`

	User   *User   `bun:"rel:belongs-to,join:userId=id" json:"user,omitempty"`
	Reward *Reward `bun:"rel:belongs-to,join:rewardId=id" json:"reward,omitempty"`
}
