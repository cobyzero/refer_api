package models

import "github.com/uptrace/bun"

type Location struct {
	bun.BaseModel `bun:"table:locations"`

	ID        string   `bun:"id,pk,type:char(36)" json:"id"`
	Name      string   `bun:"name,notnull" json:"name"`
	Address   string   `bun:"address,notnull" json:"address"`
	City      *string  `bun:"city" json:"city"`
	State     *string  `bun:"state" json:"state"`
	ZipCode   *string  `bun:"zipCode" json:"zipCode"`
	Latitude  *float64 `bun:"latitude" json:"latitude"`
	Longitude *float64 `bun:"longitude" json:"longitude"`
	IsActive  bool     `bun:"isActive,default:true" json:"isActive"`
}
