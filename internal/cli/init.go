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

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Set up regctl (first-time setup wizard)",
		Long: `Interactive setup wizard for first-time users.

This will guide you through:
  1. Setting your registrar API keys
  2. Verifying the connection
  3. Listing your domains

Supported registrars:
  Porkbun, Spaceship, Namecheap, Cloudflare, Value Domain`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInitWizard()
		},
	}
}

func runInitWizard() error {
	reader := bufio.NewReader(os.Stdin)

	printWelcomeBanner()

	// Step 1: API Key
	printStep(1, 3, "Value Domain API Key")
	fmt.Println()
	fmt.Println("  To use regctl, you need a Value Domain API key.")
	fmt.Println()
	fmt.Println("  Get your API key here:")
	color.New(color.FgCyan, color.Underline).Println("  https://www.value-domain.com/api/")
	fmt.Println()

	// Check if already configured
	existing, _ := config.Load()
	if existing != nil && existing.APIKey != "" {
		masked := existing.APIKey[:4] + "****" + existing.APIKey[len(existing.APIKey)-4:]
		fmt.Printf("  Current API key: %s\n", masked)
		fmt.Print("  Keep this key? [Y/n]: ")
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer == "" || answer == "y" || answer == "yes" {
			fmt.Println()
			color.Green("  Keeping existing API key.")
			goto step2
		}
	}

	{
		fmt.Print("  Enter your API key: ")
		apiKey, _ := reader.ReadString('\n')
		apiKey = strings.TrimSpace(apiKey)

		if apiKey == "" {
			return fmt.Errorf("API key is required. Get one at https://www.value-domain.com/api/")
		}

		if err := config.Set("api_key", apiKey); err != nil {
			return fmt.Errorf("save API key: %w", err)
		}
		color.Green("  API key saved!")
	}

step2:
	// Step 2: Verify connection
	printStep(2, 3, "Verify connection")
	fmt.Println()
	fmt.Println("  Testing your API key...")

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	testClient := newClientFromConfig(cfg)
	domains, err := testClient.ListDomains()
	if err != nil {
		color.Red("  Connection failed: %s", err)
		fmt.Println()
		fmt.Println("  Possible causes:")
		fmt.Println("    - Invalid API key")
		fmt.Println("    - Network issue")
		fmt.Println("    - Value Domain API is down")
		fmt.Println()
		fmt.Println("  You can update your key later with:")
		color.Cyan("    regctl config set api_key <your-key>")
		return nil
	}

	color.Green("  Connected successfully!")
	fmt.Printf("  Found %d domain(s) in your account.\n", len(domains))

	// Step 3: Summary
	printStep(3, 3, "You're all set!")
	fmt.Println()

	printUsageGuide()

	return nil
}

func printWelcomeBanner() {
	cyan := color.New(color.FgCyan, color.Bold)
	fmt.Println()
	cyan.Println("  ┌─────────────────────────────────────┐")
	cyan.Println("  │         Welcome to regctl!           │")
	cyan.Println("  │    Domain Management Made Easy       │")
	cyan.Println("  └─────────────────────────────────────┘")
	fmt.Println()
}

func printStep(current, total int, title string) {
	fmt.Println()
	color.New(color.FgCyan, color.Bold).Printf("  [%d/%d] %s\n", current, total, title)
}

func printUsageGuide() {
	bold := color.New(color.Bold)

	bold.Println("  Quick Start:")
	fmt.Println()
	fmt.Println("    List your domains:")
	color.Cyan("      regctl domains list")
	fmt.Println()
	fmt.Println("    Check if a domain is available:")
	color.Cyan("      regctl domains check example.com")
	fmt.Println()
	fmt.Println("    View DNS records:")
	color.Cyan("      regctl dns list example.com")
	fmt.Println()
	fmt.Println("    Add a DNS record:")
	color.Cyan("      regctl dns add example.com -t A -n @ -c 1.2.3.4")
	fmt.Println()
	fmt.Println("    Get AI-friendly output (JSON):")
	color.Cyan("      regctl domains list --format json")
	fmt.Println()
	fmt.Println("    Start API server:")
	color.Cyan("      regctl server")
	fmt.Println()
	fmt.Println("    See all commands:")
	color.Cyan("      regctl --help")
	fmt.Println()
}
