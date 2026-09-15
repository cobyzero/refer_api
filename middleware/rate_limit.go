package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// Rate limiting (Fix #1, #2): per-IP fixed windows on auth and money
// endpoints. In-memory store (single instance); when we scale to >1 replica
// (Fase 1 Redis) swap Storage for the Redis store — call sites stay the same.

func limitReached(c *fiber.Ctx) error {
	return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
		"message": "Too many requests. Inténtalo de nuevo en un minuto.",
		"code":    "RATE_LIMITED",
	})
}

// AuthLoginLimiter throttles credential-stuffing on login/register/refresh.
func AuthLoginLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        20,
		Expiration: 60 * time.Second,
		LimitReached: limitReached,
	})
}

// AuthSensitiveLimiter throttles OTP/reset brute force: verify, resend,
// forgot-password and reset-password (6-digit codes).
func AuthSensitiveLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        8,
		Expiration: 60 * time.Second,
		LimitReached: limitReached,
	})
}

// PaymentLimiter throttles Stripe intent creation (card-testing abuse).
func PaymentLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        15,
		Expiration: 60 * time.Second,
		LimitReached: limitReached,
	})
}

// OrderCreateLimiter throttles order spam (DoS / inventory noise).
func OrderCreateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        30,
		Expiration: 60 * time.Second,
		LimitReached: limitReached,
	})
}
