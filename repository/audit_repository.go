package repository

import (
	"context"
	"strings"
	"time"

	"refer_api/models"

	"github.com/uptrace/bun"
)

// AuditFilter constrains the admin audit listing (all fields optional).
type AuditFilter struct {
	UserID     string
	Action     string // exact match, e.g. "order.create"
	ActionLike string // ILIKE %...% over action+email
	EntityType string
	EntityID   string
	From       *time.Time
	To         *time.Time
	Page       int // 1-based
	Limit      int // max 100
}

func (f AuditFilter) page() int {
	if f.Page < 1 {
		return 1
	}
	return f.Page
}

func (f AuditFilter) limit() int {
	if f.Limit < 1 {
		return 20
	}
	if f.Limit > 100 {
		return 100
	}
	return f.Limit
}

// AuditRepository persists and queries audit entries.
type AuditRepository interface {
	Create(entry *models.AuditLog) error
	List(filter AuditFilter) ([]models.AuditLog, int64, error)
	DistinctActions(limit int) ([]string, error)
}

type auditRepository struct {
	db *bun.DB
}

// NewAuditRepository creates a Bun-backed AuditRepository.
func NewAuditRepository(db *bun.DB) AuditRepository {
	return &auditRepository{db: db}
}

func (r *auditRepository) Create(entry *models.AuditLog) error {
	_, err := r.db.NewInsert().Model(entry).Exec(context.Background())
	return err
}

func applyAuditFilter(q *bun.SelectQuery, f AuditFilter) *bun.SelectQuery {
	if f.UserID != "" {
		q = q.Where("\"userId\" = ?", f.UserID)
	}
	if f.Action != "" {
		q = q.Where("action = ?", f.Action)
	}
	if s := strings.TrimSpace(f.ActionLike); s != "" {
		q = q.Where("(action ILIKE ? OR COALESCE(email,'') ILIKE ?)",
			"%"+s+"%", "%"+s+"%")
	}
	if f.EntityType != "" {
		q = q.Where("\"entityType\" = ?", f.EntityType)
	}
	if f.EntityID != "" {
		q = q.Where("\"entityId\" = ?", f.EntityID)
	}
	if f.From != nil {
		q = q.Where("\"createdAt\" >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("\"createdAt\" <= ?", *f.To)
	}
	return q
}

func (r *auditRepository) List(f AuditFilter) ([]models.AuditLog, int64, error) {
	ctx := context.Background()
	out := []models.AuditLog{}
	countQ := r.db.NewSelect().Model((*models.AuditLog)(nil))
	countQ = applyAuditFilter(countQ, f)
	total, err := countQ.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	q := r.db.NewSelect().Model(&out).Order("createdAt DESC").
		Limit(f.limit()).Offset((f.page() - 1) * f.limit())
	q = applyAuditFilter(q, f)
	if err := q.Scan(ctx); err != nil {
		return nil, 0, err
	}
	return out, int64(total), nil
}

func (r *auditRepository) DistinctActions(limit int) ([]string, error) {
	if limit < 1 || limit > 200 {
		limit = 100
	}
	var actions []string
	err := r.db.NewSelect().Model((*models.AuditLog)(nil)).
		Column("action").Distinct().
		Order("action ASC").Limit(limit).
		Scan(context.Background(), &actions)
	return actions, err
}
