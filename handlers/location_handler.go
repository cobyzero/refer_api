package handlers

import (
	"refer_api/services"

	"github.com/gofiber/fiber/v2"
)

// LocationHandler coordinates HTTP requests for physical locations.
type LocationHandler struct {
	locationService services.LocationService
}

// NewLocationHandler constructs a new LocationHandler.
func NewLocationHandler(locationService services.LocationService) *LocationHandler {
	return &LocationHandler{locationService: locationService}
}

// GetAll returns all active physical locations.
func (h *LocationHandler) GetAll(c *fiber.Ctx) error {
	locations, err := h.locationService.FindAll()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(locations)
}

// GetByID returns detailed information for a single location.
func (h *LocationHandler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	location, err := h.locationService.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Location not found"})
	}
	return c.JSON(location)
}
