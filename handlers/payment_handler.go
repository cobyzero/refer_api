package handlers

import (
	"errors"

	"refer_api/middleware"
	"refer_api/models"
	"refer_api/services"

	"github.com/gofiber/fiber/v2"
)

// PaymentHandler coordinates HTTP requests for payment processing.
type PaymentHandler struct {
	paymentService services.PaymentService
	orderService   services.OrderService
}

// NewPaymentHandler constructs a new PaymentHandler. orderService may be nil
// in tests that only cover raw amounts; the orderId path requires it.
func NewPaymentHandler(paymentService services.PaymentService, orderService services.OrderService) *PaymentHandler {
	return &PaymentHandler{paymentService: paymentService, orderService: orderService}
}

// CreatePaymentIntent generates the client secret for a pending order.
//
// Order-first flow (Fix #2): orderId is REQUIRED. The charged amount always
// comes from the server-side order total (ownership-checked, still pending);
// raw client amounts are no longer accepted — that was the card-testing and
// underpayment vector.
func (h *PaymentHandler) CreatePaymentIntent(c *fiber.Ctx) error {
	var body struct {
		Currency string `json:"currency"`
		OrderID  string `json:"orderId"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}
	if body.OrderID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "orderId is required"})
	}
	if h.orderService == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "payments unavailable"})
	}

	userID, _ := c.Locals("userId").(string)
	order, err := h.orderService.FindByIDForUser(body.OrderID, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "order not found"})
	}
	if order.Status != models.OrderStatusPendingPayment {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": "order cannot be charged"})
	}

	clientSecret, intentID, err := h.paymentService.CreatePaymentIntent(order.Total, body.Currency)
	if err != nil {
		if errors.Is(err, services.ErrInvalidPayment) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	if _, err := h.orderService.AttachPaymentIntent(order.ID, userID, intentID); err != nil {
		// The intent exists at Stripe but isn't linked: harmless orphan
		// (auto-voided by Stripe), surfaced so the client retries cleanly.
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": err.Error()})
	}

	// Never log clientSecret — order/total only.
	middleware.AttachAudit(c, "payment.intent", "payment", "", map[string]interface{}{
		"orderId": order.ID, "amount": order.Total, "currency": body.Currency,
	})
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"clientSecret": clientSecret,
		"orderId":      order.ID,
	})
}
