package routes

import (
	"refer_api/handlers"
	"refer_api/middleware"

	"github.com/gofiber/fiber/v2"
)

// RegisterOrderRoutes registers order endpoints on the router group.
func RegisterOrderRoutes(router fiber.Router, orderHandler *handlers.OrderHandler) {
	orders := router.Group("/orders", middleware.JWTProtected())

	// Fix #2: throttle order spam (totals are server-computed now, this only
	// bounds request volume / inventory noise).
	orders.Post("/", middleware.OrderCreateLimiter(), orderHandler.Create)
	// Order-first flow: confirm settles after Stripe verification, cancel
	// voids a pending order. Both are owner-or-ADMIN (checked in service).
	orders.Post("/:id/confirm", orderHandler.Confirm)
	orders.Post("/:id/cancel", orderHandler.Cancel)
	orders.Get("/", orderHandler.FindAll)
	orders.Patch("/:id/status", middleware.RequireRole("ADMIN"), orderHandler.UpdateStatus)
}
