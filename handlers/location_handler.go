package handlers

import (
	"errors"

	"refer_api/middleware"
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
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(location)
}

// GetAllAdmin returns every location for back-office management (no active filter).
func (h *LocationHandler) GetAllAdmin(c *fiber.Ctx) error {
	locations, err := h.locationService.FindAllAdmin()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(locations)
}

// locationBody mirrors the operator-editable fields (pointers = partial update).
type locationBody struct {
	Name      *string  `json:"name"`
	Address   *string  `json:"address"`
	City      *string  `json:"city"`
	State     *string  `json:"state"`
	ZipCode   *string  `json:"zipCode"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	IsActive  *bool    `json:"isActive"`
}

func toLocationInput(body locationBody) services.LocationInput {
	return services.LocationInput{
		Name:      body.Name,
		Address:   body.Address,
		City:      body.City,
		State:     body.State,
		ZipCode:   body.ZipCode,
		Latitude:  body.Latitude,
		Longitude: body.Longitude,
		IsActive:  body.IsActive,
	}
}

// CreateLocation inserts a new branch / pickup point.
func (h *LocationHandler) CreateLocation(c *fiber.Ctx) error {
	var body locationBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	location, err := h.locationService.Create(toLocationInput(body))
	if err != nil {
		switch {
		case errors.Is(err, services.ErrLocationNameInvalid),
			errors.Is(err, services.ErrLocationAddressInvalid):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
		}
	}

	middleware.AttachAudit(c, "location.create", "location", location.ID, map[string]interface{}{"name": location.Name})
	return c.Status(fiber.StatusCreated).JSON(location)
}

// UpdateLocation patches an existing location.
func (h *LocationHandler) UpdateLocation(c *fiber.Ctx) error {
	var body locationBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	location, err := h.locationService.Update(c.Params("id"), toLocationInput(body))
	if err != nil {
		switch {
		case errors.Is(err, services.ErrLocationNotFound):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		case errors.Is(err, services.ErrLocationNameInvalid),
			errors.Is(err, services.ErrLocationAddressInvalid):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
		}
	}
	middleware.AttachAudit(c, "location.update", "location", c.Params("id"), nil)

	return c.JSON(location)
}

// DeleteLocation removes a location unless orders reference it (409).
func (h *LocationHandler) DeleteLocation(c *fiber.Ctx) error {
	if err := h.locationService.Delete(c.Params("id")); err != nil {
		switch {
		case errors.Is(err, services.ErrLocationNotFound):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		case errors.Is(err, services.ErrLocationInUse):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": err.Error()})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
		}
	}

	middleware.AttachAudit(c, "location.delete", "location", c.Params("id"), nil)
	return c.SendStatus(fiber.StatusNoContent)
}
