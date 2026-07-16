package routes

import (
	"refer_api/handlers"
	"refer_api/middleware"

	"github.com/gofiber/fiber/v2"
)

// RegisterOrderRoutes registers order endpoints on the router group.
func RegisterOrderRoutes(router fiber.Router, orderHandler *handlers.OrderHandler) {
	orders := router.Group("/orders", middleware.JWTProtected())

	orders.Post("/", orderHandler.Create)
	orders.Get("/", orderHandler.FindAll)
	orders.Patch("/:id/status", middleware.RequireRole("ADMIN"), orderHandler.UpdateStatus)
}
