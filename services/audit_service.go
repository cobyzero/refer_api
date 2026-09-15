package services

import (
	"strings"
	"time"

	"refer_api/models"
	"refer_api/repository"

	"github.com/google/uuid"
)

// AuditService records domain events and serves the admin listing.
type AuditService interface {
	// Log persists one immutable entry. Empty strings become NULL-ish
	// (omitted JSON fields); it never fails the caller loudly — returns err
	// so middleware can decide (it swallows it to never break the request).
	Log(userID, email string, roles []string, action, entityType, entityID string, status int, ip string, meta map[string]interface{}) error
	List(f repository.AuditFilter) ([]models.AuditLog, int64, error)
	Actions() ([]string, error)
}

type auditService struct {
	repo repository.AuditRepository
}

// NewAuditService constructs the service.
func NewAuditService(repo repository.AuditRepository) AuditService {
	return &auditService{repo: repo}
}

func strPtr(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

func (s *auditService) Log(userID, email string, roles []string, action, entityType, entityID string, status int, ip string, meta map[string]interface{}) error {
	var rolesStr *string
	if len(roles) > 0 {
		rolesStr = strPtr(strings.Join(roles, ","))
	}
	entry := &models.AuditLog{
		ID:         uuid.New().String(),
		CreatedAt:  time.Now(),
		UserID:     strPtr(userID),
		Email:      strPtr(email),
		Roles:      rolesStr,
		Action:     action,
		EntityType: strPtr(entityType),
		EntityID:   strPtr(entityID),
		Status:     status,
		IP:         strPtr(ip),
		Meta:       models.JSONMap(meta),
	}
	return s.repo.Create(entry)
}

func (s *auditService) List(f repository.AuditFilter) ([]models.AuditLog, int64, error) {
	return s.repo.List(f)
}

func (s *auditService) Actions() ([]string, error) {
	return s.repo.DistinctActions(100)
}
