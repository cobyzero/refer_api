package handlers

import (
	"github.com/gofiber/fiber/v2"
)

// ConfigHandler coordinates HTTP requests for app configuration.
type ConfigHandler struct{}

// NewConfigHandler constructs a new ConfigHandler.
func NewConfigHandler() *ConfigHandler {
	return &ConfigHandler{}
}

// GetConfig returns version, maintenance status, active features, and system links.
func (h *ConfigHandler) GetConfig(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"version":     "1.0.0",
		"minVersion":  "1.0.0",
		"maintenance": false,
		"features": fiber.Map{
			"starsEnabled":     true,
			"referralsEnabled": true,
			"authEnabled":      true,
		},
		"endpoints": fiber.Map{
			"termsUrl":   "https://example.com/terms",
			"privacyUrl": "https://example.com/privacy",
			"supportUrl": "https://example.com/support",
		},
	})
}
