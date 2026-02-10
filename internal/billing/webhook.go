package billing

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/webhook"
)

// WebhookResult contains the result of processing a webhook event.
type WebhookResult struct {
	Type       string // "signup" or "topup"
	CustomerID string
	APIKey     string // only set for signup
}

// processedEvents tracks webhook event IDs to prevent duplicate processing.
var (
	processedEvents   = make(map[string]bool)
	processedEventsMu sync.Mutex
)

// HandleWebhook verifies and processes a Stripe webhook event.
func HandleWebhook(payload []byte, sigHeader, webhookSecret, signingSecret string, live bool) (*WebhookResult, error) {
	if webhookSecret == "" {
		return nil, errors.New("webhook secret not configured")
	}

	event, err := webhook.ConstructEvent(payload, sigHeader, webhookSecret)
	if err != nil {
		return nil, fmt.Errorf("verify webhook signature: %w", err)
	}

	if event.Type != "checkout.session.completed" {
		return nil, nil // ignore other events
	}

	// Idempotency: skip already-processed events
	processedEventsMu.Lock()
	if processedEvents[event.ID] {
		processedEventsMu.Unlock()
		return nil, nil
	}
	processedEvents[event.ID] = true
	processedEventsMu.Unlock()

	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		return nil, fmt.Errorf("parse session: %w", err)
	}

	eventType := session.Metadata["type"]
	amountStr := session.Metadata["amount_cents"]
	amountCents, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid amount_cents in metadata: %s", amountStr)
	}
	if amountCents <= 0 {
		return nil, errors.New("amount must be positive")
	}

	switch eventType {
	case "signup":
		customerID := ""
		if session.Customer != nil {
			customerID = session.Customer.ID
		}
		if customerID == "" {
			return nil, errors.New("no customer ID in signup session")
		}

		if err := AddBalance(customerID, amountCents, "Initial signup credit"); err != nil {
			return nil, fmt.Errorf("add signup balance: %w", err)
		}

		apiKey := GenerateAPIKey(customerID, signingSecret, live)
		return &WebhookResult{
			Type:       "signup",
			CustomerID: customerID,
			APIKey:     apiKey,
		}, nil

	case "topup":
		customerID := session.Metadata["customer_id"]
		if customerID == "" && session.Customer != nil {
			customerID = session.Customer.ID
		}
		if customerID == "" {
			return nil, errors.New("no customer ID in topup session")
		}

		if err := AddBalance(customerID, amountCents, fmt.Sprintf("Top-up: $%.2f", float64(amountCents)/100)); err != nil {
			return nil, fmt.Errorf("add topup balance: %w", err)
		}

		return &WebhookResult{
			Type:       "topup",
			CustomerID: customerID,
		}, nil

	default:
		return nil, fmt.Errorf("unknown session type: %s", eventType)
	}
}
