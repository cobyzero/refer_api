package routes

import (
	"refer_api/handlers"
	"refer_api/middleware"

	"github.com/gofiber/fiber/v2"
)

// RegisterDashboardRoutes registers dashboard endpoints on the router group.
func RegisterDashboardRoutes(router fiber.Router, dashboardHandler *handlers.DashboardHandler) {
	dashboard := router.Group("/dashboard", middleware.JWTProtected())

	dashboard.Get("/summary", dashboardHandler.GetSummary)
	dashboard.Get("/admin/summary", middleware.RequireRole("ADMIN"), dashboardHandler.GetAdminSummary)
}
