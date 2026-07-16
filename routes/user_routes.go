package routes

import (
	"refer_api/handlers"
	"refer_api/middleware"

	"github.com/gofiber/fiber/v2"
)

// RegisterUserRoutes registers all user-related endpoints on the router.
func RegisterUserRoutes(router fiber.Router, userHandler *handlers.UserHandler) {
	users := router.Group("/users", middleware.JWTProtected())

	// Profile Operations
	users.Get("/profile", userHandler.GetProfile)
	users.Patch("/profile", userHandler.UpdateProfile)
	users.Post("/upload", userHandler.UploadPhoto)

	// Admin Operations
	adminUsers := users.Group("/", middleware.RequireRole("ADMIN"))
	adminUsers.Get("/", userHandler.FindAll)
	adminUsers.Get("/:id", userHandler.FindByID)
	adminUsers.Post("/", userHandler.Create)
	adminUsers.Patch("/:id", userHandler.Update)
	adminUsers.Delete("/:id", userHandler.Delete)
}
