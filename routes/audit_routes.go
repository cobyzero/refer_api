package routes

import (
	"refer_api/handlers"
	"refer_api/middleware"

	"github.com/gofiber/fiber/v2"
)

// RegisterAuditRoutes exposes the admin audit-log module (refer_web).
// ADMIN only; reads are excluded from auto-audit (SkipAudit) to avoid loops.
func RegisterAuditRoutes(router fiber.Router, auditHandler *handlers.AuditHandler) {
	audit := router.Group("/audit-logs", middleware.JWTProtected(), middleware.RequireRole("ADMIN"))

	audit.Get("/", auditHandler.List)
	audit.Get("/actions", auditHandler.Actions)
}
