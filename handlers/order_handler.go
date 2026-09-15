package handlers

import (
	"errors"

	"refer_api/middleware"
	"refer_api/models"
	"refer_api/services"

	"github.com/gofiber/fiber/v2"
)

// OrderHandler coordinates HTTP requests for orders.
type OrderHandler struct {
	orderService services.OrderService
}

// NewOrderHandler constructs a new OrderHandler.
func NewOrderHandler(orderService services.OrderService) *OrderHandler {
	return &OrderHandler{orderService: orderService}
}

// Create places a new order. Totals are computed server-side from the DB
// catalog; client totals are ignored for charging (fraud telemetry only).
func (h *OrderHandler) Create(c *fiber.Ctx) error {
	userID := c.Locals("userId").(string)

	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}
	clientTotal, _ := body["total"].(float64)

	order, err := h.orderService.Create(userID, body)
	if err != nil {
		if errors.Is(err, services.ErrInvalidOrder) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	meta := map[string]interface{}{"total": order.Total}
	if diff := clientTotal - order.Total; diff > 0.01 || diff < -0.01 {
		meta["clientTotal"] = clientTotal
		meta["totalMismatch"] = true
	}
	middleware.AttachAudit(c, "order.create", "order", order.ID, meta)
	return c.Status(fiber.StatusCreated).JSON(order)
}

// UpdateStatus updates the order stage (Admin only).
func (h *OrderHandler) UpdateStatus(c *fiber.Ctx) error {
	id := c.Params("id")

	var body struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	order, err := h.orderService.UpdateStatus(id, body.Status)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	middleware.AttachAudit(c, "order.status", "order", id, map[string]interface{}{"status": body.Status})
	return c.JSON(order)
}

// Confirm settles a pending order after Stripe verification (owner or ADMIN).
// Body: {} — proof comes from Stripe server-side, never from the client.
func (h *OrderHandler) Confirm(c *fiber.Ctx) error {
	userID, _ := c.Locals("userId").(string)
	var roles []string
	if r, ok := c.Locals("roles").([]string); ok {
		roles = r
	}

	order, err := h.orderService.ConfirmOrder(c.Params("id"), userID, roles)
	if err != nil {
		if errors.Is(err, services.ErrInvalidOrder) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": err.Error()})
		}
		if err.Error() == "order not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "order not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	middleware.AttachAudit(c, "order.confirm", "order", order.ID, map[string]interface{}{"total": order.Total})
	return c.JSON(order)
}

// Cancel voids a pending order (owner or ADMIN).
func (h *OrderHandler) Cancel(c *fiber.Ctx) error {
	userID, _ := c.Locals("userId").(string)
	var roles []string
	if r, ok := c.Locals("roles").([]string); ok {
		roles = r
	}

	order, err := h.orderService.CancelOrder(c.Params("id"), userID, roles)
	if err != nil {
		if errors.Is(err, services.ErrInvalidOrder) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": err.Error()})
		}
		if err.Error() == "order not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "order not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	middleware.AttachAudit(c, "order.cancel", "order", order.ID, nil)
	return c.JSON(order)
}

// FindAll returns all orders for admins, or the user's personal orders list.
func (h *OrderHandler) FindAll(c *fiber.Ctx) error {
	userID := c.Locals("userId").(string)
	rolesVal := c.Locals("roles")

	isAdmin := false
	if roles, ok := rolesVal.([]string); ok {
		for _, r := range roles {
			if r == "ADMIN" {
				isAdmin = true
				break
			}
		}
	}

	var orders []models.Order
	var err error
	if isAdmin {
		orders, err = h.orderService.FindAll()
	} else {
		orders, err = h.orderService.FindAllByUser(userID)
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	return c.JSON(orders)
}
