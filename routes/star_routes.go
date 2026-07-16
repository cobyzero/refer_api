package routes

import (
	"refer_api/handlers"
	"refer_api/middleware"

	"github.com/gofiber/fiber/v2"
)

// RegisterStarRoutes registers all star-related and perk-related endpoints on the router.
func RegisterStarRoutes(router fiber.Router, starHandler *handlers.StarHandler) {
	stars := router.Group("/stars", middleware.JWTProtected())

	// User Star Operations
	stars.Get("/balance", starHandler.GetBalance)
	stars.Get("/history", starHandler.GetHistory)
	stars.Get("/rewards", starHandler.GetRewards)
	stars.Post("/redeem", starHandler.RedeemReward)
	stars.Get("/redeemed", starHandler.GetRedeemedRewards)

	// Admin Operations
	adminStars := stars.Group("/", middleware.RequireRole("ADMIN"))
	adminStars.Get("/transactions", starHandler.GetAllTransactions)
	adminStars.Post("/rewards", starHandler.CreateReward)
	adminStars.Patch("/rewards/:id", starHandler.UpdateReward)
	adminStars.Delete("/rewards/:id", starHandler.DeleteReward)

	// Perks Operations (Public)
	router.Get("/perks", starHandler.GetPerks)
}
