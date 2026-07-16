package handlers

import (
	"refer_api/services"

	"github.com/gofiber/fiber/v2"
)

// PaymentHandler coordinates HTTP requests for payment processing.
type PaymentHandler struct {
	paymentService services.PaymentService
}

// NewPaymentHandler constructs a new PaymentHandler.
func NewPaymentHandler(paymentService services.PaymentService) *PaymentHandler {
	return &PaymentHandler{paymentService: paymentService}
}

// CreatePaymentIntent processes request data and generates a payment intent client secret.
func (h *PaymentHandler) CreatePaymentIntent(c *fiber.Ctx) error {
	var body struct {
		Amount   float64 `json:"amount"`
		Currency string  `json:"currency"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	if body.Amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Amount must be greater than zero"})
	}

	clientSecret, err := h.paymentService.CreatePaymentIntent(body.Amount, body.Currency)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"clientSecret": clientSecret,
	})
}
