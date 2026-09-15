package models

import (
	"time"

	"github.com/uptrace/bun"
)

// AppSetting is a generic key/value row in the app_settings table.
// Used for operator-controlled configuration such as the display currency.
type AppSetting struct {
	bun.BaseModel `bun:"table:app_settings"`

	Key       string    `bun:"key,pk" json:"key"`
	Value     string    `bun:"value,notnull" json:"value"`
	UpdatedAt time.Time `bun:"updatedAt,nullzero,notnull,default:current_timestamp" json:"updatedAt"`
}
