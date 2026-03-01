package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yukihamada/regctl/internal/config"
)

func newLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Sign in to regctl.sh and save your API key",
		Long: `Sign in to regctl.sh with your email address.

A 6-digit code will be sent to your email.
Your API key is saved automatically — no copy-paste needed.`,
		Example: `  regctl login`,
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := bufio.NewReader(os.Stdin)

			apiURL := "https://regctl-api.fly.dev"
			if cfg != nil && cfg.RegctlAPIURL != "" {
				apiURL = cfg.RegctlAPIURL
			}
			if cfg == nil {
				cfg, _ = config.Load()
			}

			fmt.Println()
			color.New(color.FgCyan, color.Bold).Println("  Sign in to regctl.sh")
			fmt.Println()

			// Prompt for email (show default if already set)
			var email string
			if cfg != nil && cfg.Email != "" {
				fmt.Printf("  Email [%s]: ", cfg.Email)
				input, _ := reader.ReadString('\n')
				input = strings.TrimSpace(input)
				if input == "" {
					email = cfg.Email
				} else {
					email = strings.ToLower(input)
				}
			} else {
				fmt.Print("  Email: ")
				raw, _ := reader.ReadString('\n')
				email = strings.ToLower(strings.TrimSpace(raw))
			}

			if email == "" || !strings.Contains(email, "@") || !strings.Contains(email, ".") {
				return fmt.Errorf("invalid email address")
			}

			ac := NewAPIClient(apiURL, "")

			// Send verification code
			fmt.Printf("\n  Sending code to %s...", email)
			if err := ac.RequestEmailAuth(email); err != nil {
				fmt.Println()
				return fmt.Errorf("failed to send code: %w", err)
			}
			color.Green(" sent ✓")
			fmt.Println()

			// Prompt for 6-digit code
			fmt.Print("  Verification code: ")
			codeRaw, _ := reader.ReadString('\n')
			code := strings.TrimSpace(codeRaw)

			if len(code) != 6 {
				return fmt.Errorf("code must be 6 digits")
			}

			// Verify and get api_key
			apiKey, retEmail, err := ac.VerifyEmailCode(email, code)
			if err != nil {
				return fmt.Errorf("verification failed: %w", err)
			}
			if apiKey == "" {
				return fmt.Errorf("server did not return an API key")
			}

			// Save both api-key and email
			if err := config.Set("regctl_billing_key", apiKey); err != nil {
				return fmt.Errorf("save api key: %w", err)
			}
			if err := config.Set("email", retEmail); err != nil {
				return fmt.Errorf("save email: %w", err)
			}

			fmt.Println()
			color.Green("  ✓ Logged in as %s", retEmail)
			color.Green("  ✓ API key saved to %s", config.GetConfigPath())
			fmt.Println()
			fmt.Println("  Next steps:")
			color.Cyan("    regctl domains list")
			color.Cyan("    regctl billing balance")
			color.Cyan("    regctl domains check yourdomain.com")
			fmt.Println()
			return nil
		},
	}
}
