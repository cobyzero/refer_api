package services

import (
	"math"
	"os"

	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/paymentintent"
)

// PaymentService defines the business logic interface for processing payments.
type PaymentService interface {
	CreatePaymentIntent(amount float64, currency string) (string, error)
}

type paymentService struct{}

// NewPaymentService constructs a new PaymentService.
func NewPaymentService() PaymentService {
	return &paymentService{}
}

func (s *paymentService) CreatePaymentIntent(amount float64, currency string) (string, error) {
	stripeKey := os.Getenv("STRIPE_SECRET_KEY")
	if stripeKey == "" {
		stripeKey = "sk_test_placeholder"
	}
	stripe.Key = stripeKey

	if currency == "" {
		currency = "usd"
	}

	amountCents := int64(math.Round(amount * 100))

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amountCents),
		Currency: stripe.String(currency),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		return "", err
	}

	return pi.ClientSecret, nil
}
