package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/yukihamada/regctl/internal/billing"
)

// lowBalanceThresholdCents is the balance threshold below which a warning is shown.
const lowBalanceThresholdCents = 100 // $1.00

// holdResult represents an active balance hold.
type holdResult struct {
	HoldID       string
	CostCents    int64
	BalanceAfter int64
	client       *APIClient
}

// Confirm finalizes the hold (charge is kept).
func (h *holdResult) Confirm() error {
	return h.client.ConfirmHold(h.HoldID)
}

// Release cancels the hold (refunds the amount).
func (h *holdResult) Release() error {
	return h.client.ReleaseHold(h.HoldID)
}

// billingGuard checks balance and creates a hold before a billable operation.
// If billing key is not configured, returns (nil, nil) — caller should proceed without billing.
// If balance is insufficient, prints an error and returns a non-nil error.
func billingGuard(op billing.OperationType, baseCostCents int64, domain string) (*holdResult, error) {
	if cfg == nil || cfg.RegctlBillingKey == "" || cfg.RegctlAPIURL == "" {
		return nil, nil
	}

	costCents := billing.CalculateCostCents(op, baseCostCents)
	if costCents == 0 {
		return nil, nil
	}

	ac := NewAPIClient(cfg.RegctlAPIURL, cfg.RegctlBillingKey)
	description := fmt.Sprintf("%s: %s", op, domain)

	result, err := ac.HoldBalance(costCents, description)
	if err != nil {
		color.Red("    Billing: insufficient balance or billing error")
		fmt.Printf("    Estimated cost: $%.2f\n", float64(costCents)/100)
		fmt.Printf("    Error: %v\n", err)
		fmt.Println()
		fmt.Println("    Top up: regctl billing topup 10")
		return nil, fmt.Errorf("billing hold failed: %w", err)
	}

	return &holdResult{
		HoldID:       result.HoldID,
		CostCents:    costCents,
		BalanceAfter: result.BalanceAfter,
		client:       ac,
	}, nil
}

// checkLowBalance prints a warning if balance is below the threshold.
func checkLowBalance(balanceCents int64) {
	if balanceCents <= lowBalanceThresholdCents {
		fmt.Println()
		color.Yellow("    ⚠ Balance is low ($%.2f). Top up: regctl billing topup 10", float64(balanceCents)/100)
	}
}

// printBillingInfo prints the billing summary after a successful operation.
func printBillingInfo(hold *holdResult) {
	if hold == nil {
		return
	}
	fmt.Printf("    Cost: $%.2f  |  Balance: $%.2f remaining\n",
		float64(hold.CostCents)/100, float64(hold.BalanceAfter)/100)
	checkLowBalance(hold.BalanceAfter)
}
