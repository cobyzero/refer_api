package services

import (
	"errors"
	"log"
	"math"
	"os"
	"strings"

	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/paymentintent"
)

// PaymentService defines the business logic interface for processing payments.
type PaymentService interface {
	// CreatePaymentIntent creates a Stripe intent and returns
	// (clientSecret, intentID). The caller links intentID to its order.
	CreatePaymentIntent(amount float64, currency string) (clientSecret, intentID string, err error)
}

// ErrInvalidPayment marks client-correctable payment errors (bad amount or
// currency). Handlers map it to 400; Stripe failures stay 500.
var ErrInvalidPayment = errors.New("invalid payment")

type paymentService struct{}

// NewPaymentService constructs a new PaymentService.
func NewPaymentService() PaymentService {
	return &paymentService{}
}

// NormalizeCurrency validates the ISO code against the same allowlist as
// /config (case-insensitive) and returns it lowercased for Stripe.
func NormalizeCurrency(currency string) (string, error) {
	code := strings.ToUpper(strings.TrimSpace(currency))
	if code == "" {
		code = "USD"
	}
	for _, c := range SupportedCurrencies {
		if c == code {
			return strings.ToLower(code), nil
		}
	}
	return "", ErrInvalidPayment
}

func stripeKeyOrFail() (string, error) {
	stripeKey := os.Getenv("STRIPE_SECRET_KEY")
	if stripeKey == "" || stripeKey == "sk_test_placeholder" {
		return "", errors.New("payments are not configured")
	}
	return stripeKey, nil
}

func (s *paymentService) CreatePaymentIntent(amount float64, currency string) (string, string, error) {
	stripeKey, err := stripeKeyOrFail()
	if err != nil {
		return "", "", err
	}
	stripe.Key = stripeKey

	normalized, err := NormalizeCurrency(currency)
	if err != nil {
		return "", "", err
	}
	currency = normalized

	// Amount bounds: positive and capped. The upper cap bounds card-testing
	// damage and fat-finger charges; the order flow charges the server total
	// anyway (see payment handler orderId path).
	if math.IsNaN(amount) || math.IsInf(amount, 0) || amount <= 0 || amount > 50000 {
		return "", "", ErrInvalidPayment
	}

	amountCents := int64(math.Round(amount * 100))
	if amountCents <= 0 {
		return "", "", ErrInvalidPayment
	}

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amountCents),
		Currency: stripe.String(currency),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		return "", "", err
	}

	return pi.ClientSecret, pi.ID, nil
}

// VerifyPaymentSucceeded retrieves the intent from Stripe and reports whether
// it actually settled. Confirm endpoints must gate on this — the client
// telling us "it worked" is not proof.
func VerifyPaymentSucceeded(intentID string) (bool, error) {
	stripeKey, err := stripeKeyOrFail()
	if err != nil {
		return false, err
	}
	// Dev escape hatch: local stacks without a real Stripe key can still
	// exercise the order-first flow. NEVER enable in prod (fail closed by
	// default); every bypass is logged loudly.
	if os.Getenv("ALLOW_UNVERIFIED_PAYMENTS") == "true" {
		log.Printf("[payments] WARNING: ALLOW_UNVERIFIED_PAYMENTS=true, accepting intent %s without Stripe verification", intentID)
		return true, nil
	}
	if intentID == "" {
		return false, ErrInvalidPayment
	}
	stripe.Key = stripeKey
	pi, err := paymentintent.Get(intentID, nil)
	if err != nil {
		return false, err
	}
	return pi.Status == stripe.PaymentIntentStatusSucceeded, nil
}

// CancelPaymentIntent voids an intent best-effort (used when a pending order
// is cancelled). Errors are returned so callers can log them; money never
// moved on an unconfirmed intent, so failure here is non-fatal.
func CancelPaymentIntent(intentID string) error {
	stripeKey, err := stripeKeyOrFail()
	if err != nil || intentID == "" {
		return err
	}
	stripe.Key = stripeKey
	_, err = paymentintent.Cancel(intentID, nil)
	return err
}
