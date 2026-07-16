package handlers

import (
	"refer_api/services"

	"github.com/gofiber/fiber/v2"
)

// DashboardHandler coordinates HTTP requests for dashboard summaries.
type DashboardHandler struct {
	dashboardService services.DashboardService
}

// NewDashboardHandler constructs a new DashboardHandler.
func NewDashboardHandler(dashboardService services.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboardService: dashboardService}
}

// GetSummary returns the authenticated user's home summary view.
func (h *DashboardHandler) GetSummary(c *fiber.Ctx) error {
	userID := c.Locals("userId").(string)

	summary, err := h.dashboardService.GetSummary(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	return c.JSON(summary)
}

// GetAdminSummary returns administrative dashboard metrics and recent orders.
func (h *DashboardHandler) GetAdminSummary(c *fiber.Ctx) error {
	summary, err := h.dashboardService.GetAdminSummary()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	return c.JSON(summary)
}
