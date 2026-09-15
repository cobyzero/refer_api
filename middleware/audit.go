package middleware

import (
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// auditLogger is the subset of AuditService used here. Defined locally to
// avoid an import cycle (services -> gateway -> middleware).
type auditLogger interface {
	Log(userID, email string, roles []string, action, entityType, entityID string, status int, ip string, meta map[string]interface{}) error
}

// Audit detail keys set by handlers via AttachAudit (rich context).
// Anything not set falls back to generic METHOD + route pattern.
const (
	auditActionKey = "auditAction"
	auditEntityKey = "auditEntityType"
	auditEntityID  = "auditEntityID"
	auditMetaKey   = "auditMeta"
	auditSkipKey   = "auditSkip"
)

// AttachAudit lets a handler describe its domain action before returning.
// Example: AttachAudit(c, "order.create", "order", order.ID, map[string]any{"total": order.Total})
func AttachAudit(c *fiber.Ctx, action, entityType, entityID string, meta map[string]interface{}) {
	c.Locals(auditActionKey, action)
	c.Locals(auditEntityKey, entityType)
	c.Locals(auditEntityID, entityID)
	if meta != nil {
		c.Locals(auditMetaKey, meta)
	}
}

// SkipAudit disables the automatic entry for this request (e.g. /audit-logs
// reads, health, or noisy polling) to avoid feedback loops and bloat.
func SkipAudit(c *fiber.Ctx) {
	c.Locals(auditSkipKey, true)
}

func auditString(c *fiber.Ctx, key string) string {
	if v, ok := c.Locals(key).(string); ok {
		return v
	}
	return ""
}

// DefaultAuditAction builds "METHOD /api/resource/:id" from the route.
// Pure helper, unit-tested.
func DefaultAuditAction(method, routePath string) string {
	if routePath == "" {
		routePath = "/"
	}
	return method + " " + routePath
}

// AuditMiddleware records every mutating HTTP request (POST/PUT/PATCH/DELETE)
// after it completes: actor (JWT locals, may be anonymous on public routes),
// action (explicit AttachAudit or METHOD+route), entity, status, IP.
// It NEVER breaks the request: audit failures are only logged server-side.
// Register it on the /api group BEFORE route registration — its post-Next
// logic runs after inner JWT + handlers, so locals are already populated.
func AuditMiddleware(auditSvc auditLogger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()

		if c.Locals(auditSkipKey) == true {
			return err
		}
		switch c.Method() {
		case fiber.MethodPost, fiber.MethodPut, fiber.MethodPatch, fiber.MethodDelete:
		default:
			return err
		}

		status := c.Response().StatusCode()
		action := auditString(c, auditActionKey)
		if action == "" {
			routePath := ""
			if r := c.Route(); r != nil {
				routePath = r.Path
			}
			action = DefaultAuditAction(c.Method(), routePath)
		}
		var meta map[string]interface{}
		if m, ok := c.Locals(auditMetaKey).(map[string]interface{}); ok {
			meta = m
		}
		userID := auditString(c, "userId")
		email := auditString(c, "email")
		var roles []string
		if r, ok := c.Locals("roles").([]string); ok {
			roles = r
		}
		entityType := auditString(c, auditEntityKey)
		if entityType == "" {
			entityType = entityFromPath(c.Path())
		}
		entityID := auditString(c, auditEntityID)
		if entityID == "" {
			entityID = firstParam(c, "id", "userId", "orderId", "productId", "rewardId", "collectionId", "locationId")
		}

		if logErr := auditSvc.Log(userID, email, roles, action, entityType, entityID, status, c.IP(), meta); logErr != nil {
			log.Printf("[audit] failed to persist entry: %v", logErr)
		}
		return err
	}
}

func firstParam(c *fiber.Ctx, names ...string) string {
	for _, n := range names {
		if v := c.Params(n); v != "" {
			return v
		}
	}
	return ""
}

// entityFromPath derives "users" from "/api/users/123".
func entityFromPath(p string) string {
	parts := strings.Split(strings.Trim(p, "/"), "/")
	if len(parts) >= 2 && parts[0] == "api" {
		return parts[1]
	}
	return ""
}
