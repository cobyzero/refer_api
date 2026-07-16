package routes

import (
	"refer_api/handlers"
	"refer_api/middleware"

	"github.com/gofiber/fiber/v2"
)

// RegisterAuthRoutes registers all authentication endpoints under the router group.
func RegisterAuthRoutes(router fiber.Router, authHandler *handlers.AuthHandler) {
	auth := router.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)
	auth.Post("/forgot-password", authHandler.ForgotPassword)
	auth.Post("/reset-password", authHandler.ResetPassword)
	auth.Post("/refresh", authHandler.Refresh)
	auth.Post("/change-password", middleware.JWTProtected(), authHandler.ChangePassword)
	auth.Get("/validate", middleware.JWTProtected(), authHandler.Validate)
}
