package routes

import (
	"refer_api/handlers"
	"refer_api/middleware"

	"github.com/gofiber/fiber/v2"
)

// RegisterAuthRoutes registers all authentication endpoints under the router group.
// Existing Flutter contract is preserved; /logout and /me are additive and
// safe (Flutter sin cambios sigue funcionando).
func RegisterAuthRoutes(router fiber.Router, authHandler *handlers.AuthHandler) {
	auth := router.Group("/auth")
	// Fix #1: brute-force throttles. Login/register/refresh share the
	// credential-stuffing bucket; OTP + recovery codes (6 digits) get the
	// strict bucket. Authenticated endpoints below stay unlimited.
	loginLimit := middleware.AuthLoginLimiter()
	sensitiveLimit := middleware.AuthSensitiveLimiter()
	auth.Post("/register", loginLimit, authHandler.Register)
	auth.Post("/login", loginLimit, authHandler.Login)
	auth.Post("/verify-email", sensitiveLimit, authHandler.VerifyEmail)
	auth.Post("/resend-verification", sensitiveLimit, authHandler.ResendVerification)
	auth.Post("/forgot-password", sensitiveLimit, authHandler.ForgotPassword)
	auth.Post("/reset-password", sensitiveLimit, authHandler.ResetPassword)
	auth.Post("/refresh", loginLimit, authHandler.Refresh)
	auth.Post("/logout", authHandler.Logout)
	auth.Post("/change-password", middleware.JWTProtected(), authHandler.ChangePassword)
	auth.Get("/validate", middleware.JWTProtected(), authHandler.Validate)
	auth.Get("/me", middleware.JWTProtected(), authHandler.Me)
}
