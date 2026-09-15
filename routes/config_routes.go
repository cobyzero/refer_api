package routes

import (
	"refer_api/handlers"
	"refer_api/middleware"

	"github.com/gofiber/fiber/v2"
)

// RegisterConfigRoutes registers configuration endpoints on the router group.
// GET is public (mobile needs it pre-login at splash); PUT is ADMIN only —
// it controls prices, versions and the direct-confirm testing toggle.
func RegisterConfigRoutes(router fiber.Router, configHandler *handlers.ConfigHandler) {
	router.Get("/config", configHandler.GetConfig)
	router.Put("/config", middleware.JWTProtected(), middleware.RequireRole("ADMIN"), configHandler.UpdateConfig)
}
