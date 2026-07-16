package routes

import (
	"refer_api/handlers"

	"github.com/gofiber/fiber/v2"
)

// RegisterLocationRoutes registers location endpoints on the router group.
func RegisterLocationRoutes(router fiber.Router, locationHandler *handlers.LocationHandler) {
	locations := router.Group("/locations")

	locations.Get("/", locationHandler.GetAll)
	locations.Get("/:id", locationHandler.GetByID)
}
