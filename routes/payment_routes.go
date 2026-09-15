package routes

import (
	"refer_api/handlers"
	"refer_api/middleware"

	"github.com/gofiber/fiber/v2"
)

// RegisterPaymentRoutes registers payment endpoints on the router group.
func RegisterPaymentRoutes(router fiber.Router, paymentHandler *handlers.PaymentHandler) {
	payments := router.Group("/payments", middleware.JWTProtected())

	// Fix #2: throttle intent creation (card-testing abuse).
	payments.Post("/create-payment-intent", middleware.PaymentLimiter(), paymentHandler.CreatePaymentIntent)
}
