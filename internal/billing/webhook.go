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
	Type       string // "signup", "topup", or "site_sponsor"
	CustomerID string
	APIKey     string // only set for signup

	// site_sponsor fields
	Domain             string
	OwnerCustomerID    string
	SponsorEmail       string
	SponsorCustomerID  string
	AmountCents        int64
	SiteCreditCents    int64
	SponsorTokenCents  int64
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

	event, err := webhook.ConstructEventWithOptions(payload, sigHeader, webhookSecret,
		webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true})
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

	case "site_sponsor":
		domain := session.Metadata["domain"]
		ownerCustomerID := session.Metadata["owner_customer_id"]
		sponsorEmail := session.Metadata["sponsor_email"]
		if domain == "" || ownerCustomerID == "" || sponsorEmail == "" {
			return nil, errors.New("missing site_sponsor metadata")
		}

		siteCreditCents := amountCents * int64(SponsorSitePct) / 100
		sponsorTokenCents := amountCents * int64(SponsorTokenPct) / 100

		// Add credit to site owner
		if err := AddBalance(ownerCustomerID, siteCreditCents,
			fmt.Sprintf("Sponsor from %s for %s", sponsorEmail, domain)); err != nil {
			return nil, fmt.Errorf("add site credit: %w", err)
		}

		// Find or create sponsor customer and add tokens
		sponsorCustomer, err := FindOrCreateCustomer(sponsorEmail)
		if err != nil {
			return nil, fmt.Errorf("find/create sponsor: %w", err)
		}
		if err := AddBalance(sponsorCustomer.ID, sponsorTokenCents,
			fmt.Sprintf("Sponsor token for %s", domain)); err != nil {
			return nil, fmt.Errorf("add sponsor token: %w", err)
		}

		return &WebhookResult{
			Type:              "site_sponsor",
			Domain:            domain,
			OwnerCustomerID:   ownerCustomerID,
			SponsorEmail:      sponsorEmail,
			SponsorCustomerID: sponsorCustomer.ID,
			AmountCents:       amountCents,
			SiteCreditCents:   siteCreditCents,
			SponsorTokenCents: sponsorTokenCents,
		}, nil

	default:
		return nil, fmt.Errorf("unknown session type: %s", eventType)
	}
}
