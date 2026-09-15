package routes

import (
	"refer_api/handlers"

	"github.com/gofiber/fiber/v2"
)

// RegisterLocationRoutes registers location endpoints on the router group.
// The admin list lives under /admin/all so the mobile contract
// (GET / active only, GET /:id) stays untouched.
func RegisterLocationRoutes(router fiber.Router, locationHandler *handlers.LocationHandler) {
	locations := router.Group("/locations")

	locations.Get("/", locationHandler.GetAll)
	locations.Get("/admin/all", locationHandler.GetAllAdmin)
	locations.Get("/:id", locationHandler.GetByID)
	locations.Post("/", locationHandler.CreateLocation)
	locations.Patch("/:id", locationHandler.UpdateLocation)
	locations.Delete("/:id", locationHandler.DeleteLocation)
}
