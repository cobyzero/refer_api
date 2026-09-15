package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/uptrace/bun"
)

// JSONMap is a JSONB metadata bag for audit entries.
type JSONMap map[string]interface{}

func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	// string (not []byte): pgdriver would send []byte as bytea, which jsonb rejects.
	return string(raw), nil
}

func (m *JSONMap) Scan(src interface{}) error {
	if src == nil {
		*m = nil
		return nil
	}
	var raw []byte
	switch v := src.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return errors.New("invalid type for JSONMap scan")
	}
	if len(raw) == 0 {
		*m = nil
		return nil
	}
	return json.Unmarshal(raw, m)
}

// AuditLog is an immutable record of who did what. Only Go writes it;
// refer_web (admin) reads it via GET /api/audit-logs (ADMIN only).
type AuditLog struct {
	bun.BaseModel `bun:"table:audit_logs"`

	ID         string    `bun:"id,pk,type:char(36)" json:"id"`
	CreatedAt  time.Time `bun:"createdAt,notnull,default:current_timestamp" json:"createdAt"`
	UserID     *string   `bun:"userId" json:"userId,omitempty"`
	Email      *string   `bun:"email" json:"email,omitempty"`
	Roles      *string   `bun:"roles" json:"roles,omitempty"`           // comma-joined, e.g. "ADMIN,USER"
	Action     string    `bun:"action,notnull" json:"action"`           // e.g. "order.create", "PATCH /api/users/:id"
	EntityType *string   `bun:"entityType" json:"entityType,omitempty"` // e.g. "order"
	EntityID   *string   `bun:"entityId" json:"entityId,omitempty"`
	Status     int       `bun:"status,notnull,default:0" json:"status"` // HTTP status of the request
	IP         *string   `bun:"ip" json:"ip,omitempty"`
	Meta       JSONMap   `bun:"metadata,type:jsonb" json:"meta,omitempty"`
}
