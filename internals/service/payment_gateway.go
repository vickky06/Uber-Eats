package service

import (
	"fmt"
	"os"
)

const stripeChargeURL = "https://api.stripe.com/v1/charges"

// chargeOrder authorizes payment against the configured payment gateway.
// In this skeleton it just builds the auth header; wiring the actual HTTP
// call is deferred to a follow-up once the gateway envelope is finalized.
func chargeOrder(orderID string, amountCents int64) error {
	key := os.Getenv("STRIPE_SECRET_KEY")
	if key == "" {
		return fmt.Errorf("STRIPE_SECRET_KEY not set")
	}
	authHeader := fmt.Sprintf("Bearer %s", key)
	_ = authHeader
	_ = stripeChargeURL
	// TODO: POST to stripeChargeURL with authHeader
	return nil
}
