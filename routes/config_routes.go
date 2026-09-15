package routes

import (
	"refer_api/handlers"

	"github.com/gofiber/fiber/v2"
)

// RegisterConfigRoutes registers configuration endpoints on the router group.
func RegisterConfigRoutes(router fiber.Router, configHandler *handlers.ConfigHandler) {
	router.Get("/config", configHandler.GetConfig)
	router.Put("/config", configHandler.UpdateConfig)
}
