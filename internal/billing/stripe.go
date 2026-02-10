package billing

import (
	"errors"
	"fmt"
	"sync"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
	"github.com/stripe/stripe-go/v81/customer"
	"github.com/stripe/stripe-go/v81/customerbalancetransaction"
)

// customerLocks provides per-customer mutex to prevent TOCTOU race
// conditions in balance check + deduct operations.
var customerLocks sync.Map

// Client wraps Stripe API operations for the billing system.
type Client struct {
	signingSecret string
	successURL    string
	cancelURL     string
	live          bool
}

// NewClient creates a new billing Client. stripeKey is the Stripe secret key.
func NewClient(stripeKey, signingSecret, successURL, cancelURL string) *Client {
	stripe.Key = stripeKey
	live := len(stripeKey) > 0 && stripeKey[:3] == "sk_" && len(stripeKey) > 7 && stripeKey[3:7] == "live"
	return &Client{
		signingSecret: signingSecret,
		successURL:    successURL,
		cancelURL:     cancelURL,
		live:          live,
	}
}

// GetBalance returns the customer's balance in cents (positive = credit available).
// Stripe stores credit as negative balance, so we negate it.
func GetBalance(customerID string) (int64, error) {
	c, err := customer.Get(customerID, nil)
	if err != nil {
		return 0, fmt.Errorf("get customer: %w", err)
	}
	return -c.Balance, nil // Stripe: negative = credit
}

// DeductBalance subtracts amountCents from the customer's balance.
// Returns an error if the balance is insufficient.
// Uses per-customer locking to prevent TOCTOU race conditions.
func DeductBalance(customerID string, amountCents int64, description string) error {
	if amountCents <= 0 {
		return errors.New("deduct amount must be positive")
	}

	// Acquire per-customer lock to prevent concurrent overdraft
	mu := getCustomerLock(customerID)
	mu.Lock()
	defer mu.Unlock()

	balance, err := GetBalance(customerID)
	if err != nil {
		return err
	}
	if balance < amountCents {
		return errors.New("insufficient balance")
	}

	// Positive amount = debit (reduces credit)
	params := &stripe.CustomerBalanceTransactionParams{
		Customer:    stripe.String(customerID),
		Amount:      stripe.Int64(amountCents), // positive = debit in Stripe
		Currency:    stripe.String(string(stripe.CurrencyUSD)),
		Description: stripe.String(description),
	}
	_, err = customerbalancetransaction.New(params)
	if err != nil {
		return fmt.Errorf("deduct balance: %w", err)
	}
	return nil
}

func getCustomerLock(customerID string) *sync.Mutex {
	v, _ := customerLocks.LoadOrStore(customerID, &sync.Mutex{})
	return v.(*sync.Mutex)
}

// AddBalance adds amountCents to the customer's balance (as credit).
func AddBalance(customerID string, amountCents int64, description string) error {
	if amountCents <= 0 {
		return errors.New("add amount must be positive")
	}
	// Negative amount = credit in Stripe
	params := &stripe.CustomerBalanceTransactionParams{
		Customer:    stripe.String(customerID),
		Amount:      stripe.Int64(-amountCents), // negative = credit
		Currency:    stripe.String(string(stripe.CurrencyUSD)),
		Description: stripe.String(description),
	}
	_, err := customerbalancetransaction.New(params)
	if err != nil {
		return fmt.Errorf("add balance: %w", err)
	}
	return nil
}

// CreateCustomer creates a new Stripe customer with the given email.
func CreateCustomer(email string) (*stripe.Customer, error) {
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
	}
	c, err := customer.New(params)
	if err != nil {
		return nil, fmt.Errorf("create customer: %w", err)
	}
	return c, nil
}

// CreateTopUpSession creates a Stripe Checkout session for topping up balance.
func (c *Client) CreateTopUpSession(customerID string, amountCents int64) (*stripe.CheckoutSession, error) {
	if amountCents < int64(MinTopUpCents) {
		return nil, fmt.Errorf("minimum top-up is $%.2f", float64(MinTopUpCents)/100)
	}

	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(customerID),
		Mode:     stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency:   stripe.String(string(stripe.CurrencyUSD)),
					UnitAmount: stripe.Int64(amountCents),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String("regctl API Credit"),
					},
				},
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(c.successURL + "?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(c.cancelURL),
	}
	params.AddMetadata("type", "topup")
	params.AddMetadata("customer_id", customerID)
	params.AddMetadata("amount_cents", fmt.Sprintf("%d", amountCents))

	s, err := session.New(params)
	if err != nil {
		return nil, fmt.Errorf("create checkout session: %w", err)
	}
	return s, nil
}

// CreateSignUpSession creates a Stripe Checkout session for new user signup.
func (c *Client) CreateSignUpSession(email string, initialAmountCents int64) (*stripe.CheckoutSession, error) {
	if initialAmountCents < int64(MinTopUpCents) {
		return nil, errors.New("minimum initial amount is $5.00")
	}

	params := &stripe.CheckoutSessionParams{
		CustomerEmail: stripe.String(email),
		Mode:          stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency:   stripe.String(string(stripe.CurrencyUSD)),
					UnitAmount: stripe.Int64(initialAmountCents),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String("regctl API Credit (Initial)"),
					},
				},
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(c.successURL + "?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(c.cancelURL),
	}
	params.AddMetadata("type", "signup")
	params.AddMetadata("amount_cents", fmt.Sprintf("%d", initialAmountCents))

	s, err := session.New(params)
	if err != nil {
		return nil, fmt.Errorf("create signup session: %w", err)
	}
	return s, nil
}

// IsLive returns true if the client is configured with a live Stripe key.
func (c *Client) IsLive() bool {
	return c.live
}

// GenerateAPIKeyForCustomer creates an API key for the given customer.
func (c *Client) GenerateAPIKeyForCustomer(customerID string) string {
	return GenerateAPIKey(customerID, c.signingSecret, c.live)
}
