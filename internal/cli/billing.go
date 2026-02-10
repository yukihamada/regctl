package cli

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newBillingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "billing",
		Short: "Manage billing and API credits",
		Long: `Manage your regctl API billing account.

  regctl billing balance   Check your current balance
  regctl billing topup     Add credit to your account
  regctl billing signup    Open the signup page`,
	}

	cmd.AddCommand(
		newBillingBalanceCmd(),
		newBillingTopUpCmd(),
		newBillingSignUpCmd(),
	)
	return cmd
}

func newBillingBalanceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "balance",
		Short: "Check your API credit balance",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := NewAPIClient(cfg.RegctlAPIURL, cfg.RegctlBillingKey)
			data, err := ac.GetBalance()
			if err != nil {
				return err
			}

			if isStructuredOutput() {
				printResult("billing balance", json.RawMessage(data), "", nil)
				return nil
			}

			var bal struct {
				Balance  float64 `json:"balance"`
				Currency string  `json:"currency"`
			}
			json.Unmarshal(data, &bal)

			fmt.Println()
			color.New(color.FgCyan, color.Bold).Println("  Account Balance")
			fmt.Println()
			color.New(color.FgGreen, color.Bold).Printf("  $%.2f %s\n", bal.Balance, bal.Currency)
			fmt.Println()
			return nil
		},
	}
}

func newBillingTopUpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "topup <amount_usd>",
		Short: "Add credit to your account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			amount, err := strconv.ParseFloat(args[0], 64)
			if err != nil || amount < 5 {
				return fmt.Errorf("amount must be at least $5.00")
			}
			amountCents := int64(amount * 100)

			ac := NewAPIClient(cfg.RegctlAPIURL, cfg.RegctlBillingKey)
			data, err := ac.CreateTopUp(amountCents)
			if err != nil {
				return err
			}

			var result struct {
				CheckoutURL string `json:"checkout_url"`
			}
			json.Unmarshal(data, &result)

			if isStructuredOutput() {
				printResult("billing topup", json.RawMessage(data), "", nil)
				return nil
			}

			fmt.Println()
			color.New(color.FgCyan, color.Bold).Println("  Opening Stripe Checkout...")
			fmt.Println()
			fmt.Println("  " + result.CheckoutURL)
			fmt.Println()

			openBrowser(result.CheckoutURL)
			return nil
		},
	}
}

func newBillingSignUpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "signup",
		Short: "Open the signup page in your browser",
		Run: func(cmd *cobra.Command, args []string) {
			url := "https://regctl.sh/#signup"
			fmt.Println()
			color.New(color.FgCyan, color.Bold).Println("  Opening signup page...")
			fmt.Println()
			fmt.Println("  " + url)
			fmt.Println()
			openBrowser(url)
		},
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		cmd = exec.Command("cmd", "/c", "start", url)
	}
	cmd.Start()
}
