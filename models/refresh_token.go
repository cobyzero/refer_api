package models

import (
	"time"

	"github.com/uptrace/bun"
)

// RefreshToken stores sha256 hashes of opaque API refresh tokens.
// Rotation: consumed on use (single-use), re-issued on every refresh.
// Logout / password change / reset revoke all tokens of the user.
type RefreshToken struct {
	bun.BaseModel `bun:"table:refresh_tokens"`

	TokenHash string    `bun:"token_hash,pk,type:char(64)" json:"-"`
	UserID    string    `bun:"user_id,notnull,type:char(36)" json:"userId"`
	ExpiresAt time.Time `bun:"expires_at,notnull" json:"expiresAt"`
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
}
