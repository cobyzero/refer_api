package models

import "github.com/uptrace/bun"

type Role struct {
	bun.BaseModel `bun:"table:roles"`

	ID   string `bun:"id,pk,type:char(36)" json:"id"`
	Name string `bun:"name,unique,notnull" json:"name"`
}
