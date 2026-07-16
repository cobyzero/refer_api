package handlers

import (
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

// Create places a new order.
func (h *OrderHandler) Create(c *fiber.Ctx) error {
	userID := c.Locals("userId").(string)

	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	order, err := h.orderService.Create(userID, body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

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
