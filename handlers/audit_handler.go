package handlers

import (
	"strconv"
	"time"

	"refer_api/middleware"
	"refer_api/repository"
	"refer_api/services"

	"github.com/gofiber/fiber/v2"
)

// AuditHandler serves the admin audit-log module of refer_web.
type AuditHandler struct {
	auditService services.AuditService
}

// NewAuditHandler constructs the handler.
func NewAuditHandler(auditService services.AuditService) *AuditHandler {
	return &AuditHandler{auditService: auditService}
}

// List returns paginated audit entries. Filters (all optional):
// userId, action (exact), q (action/email ILIKE), entityType, entityId,
// from/to (RFC3339), page, limit (max 100).
// Response: {data, total, page, limit} — ready for the refer_web table.
func (h *AuditHandler) List(c *fiber.Ctx) error {
	middleware.SkipAudit(c)

	f := repository.AuditFilter{
		UserID:     c.Query("userId"),
		Action:     c.Query("action"),
		ActionLike: c.Query("q"),
		EntityType: c.Query("entityType"),
		EntityID:   c.Query("entityId"),
	}
	if v := c.Query("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.Page = n
		}
	}
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.Limit = n
		}
	}
	if v := c.Query("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.From = &t
		}
	}
	if v := c.Query("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.To = &t
		}
	}

	data, total, err := h.auditService.List(f)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	page, limit := f.Page, f.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return c.JSON(fiber.Map{
		"data":  data,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// Actions returns the distinct action names for the refer_web filter dropdown.
func (h *AuditHandler) Actions(c *fiber.Ctx) error {
	middleware.SkipAudit(c)

	actions, err := h.auditService.Actions()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	if actions == nil {
		actions = []string{}
	}
	return c.JSON(fiber.Map{"actions": actions})
}
