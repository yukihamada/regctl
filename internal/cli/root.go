package cli

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yukihamada/regctl/internal/config"
	cfprovider "github.com/yukihamada/regctl/internal/provider/cloudflare"
	"github.com/yukihamada/regctl/internal/provider/namecheap"
	"github.com/yukihamada/regctl/internal/provider/porkbun"
	"github.com/yukihamada/regctl/internal/provider/spaceship"
	"github.com/yukihamada/regctl/internal/provider/valuedomain"
)

var (
	cfg              *config.Config
	client           *valuedomain.Client
	porkbunClient    *porkbun.Client
	cloudflareClient *cfprovider.Client
	spaceshipClient  *spaceship.Client
	namecheapClient  *namecheap.Client
)

// NewRootCmd creates the root command for regctl.
func NewRootCmd(version string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "regctl",
		Short: "Multi-registrar domain management CLI",
		Long: `regctl — Domain Management Made Easy

  Manage domains across multiple registrars:
    Porkbun, Cloudflare, Value Domain

  Compare prices and register at the cheapest provider.

  First time? Run:
    regctl init

  Examples:
    regctl domains check example.com   Compare prices across registrars
    regctl domains list                List all your domains
    regctl domains register example.com Register at cheapest price
    regctl dns list example.com        Show DNS records
    regctl server                      Start API server

  AI-friendly output:
    regctl domains check example.com --format json`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Commands that don't need API key
			skip := map[string]bool{
				"init": true, "set": true, "show": true,
				"help": true, "version": true, "completion": true,
				"check": true, // check works with public pricing APIs
				"signup": true, "server": true,
				"providers": true, // sites providers doesn't need API key
				"launch":    true, // launch loads config internally
			}
			if skip[cmd.Name()] {
				return nil
			}

			// Billing commands need config but not providers
			billingCmds := map[string]bool{
				"billing": true, "balance": true, "topup": true,
			}

			var err error
			cfg, err = config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			if billingCmds[cmd.Name()] {
				return nil
			}

			// Initialize all configured providers
			initProviders(cfg)

			if !cfg.HasAnyProvider() {
				fmt.Println()
				color.New(color.FgYellow, color.Bold).Println("  No registrar API keys configured.")
				fmt.Println()
				fmt.Println("  Quick setup (interactive wizard):")
				color.Cyan("    regctl init")
				fmt.Println()
				fmt.Println("  Or set keys directly:")
				color.Cyan("    regctl config set porkbun_api_key YOUR_KEY")
				color.Cyan("    regctl config set porkbun_secret_key YOUR_SECRET")
				fmt.Println()
				fmt.Println("  Supported registrars:")
				fmt.Println("    Porkbun      — porkbun_api_key + porkbun_secret_key")
				fmt.Println("    Cloudflare   — cloudflare_token (or cloudflare_global_key + cloudflare_email)")
				fmt.Println("    Namecheap    — namecheap_api_key + namecheap_api_user")
				fmt.Println("    Value Domain — api_key")
				fmt.Println()
				os.Exit(1)
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			// No arguments → interactive mode
			runInteractiveMode()
		},
	}

	rootCmd.PersistentFlags().StringVar(&outputFormat, "format", "table", "Output format: table, json, ai")

	// Backward compat: --json is shorthand for --format json
	var jsonFlag bool
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Output as JSON (shorthand for --format json)")
	rootCmd.PersistentPreRun = nil // Handled via PersistentPreRunE

	// Resolve --json to --format
	origPreRunE := rootCmd.PersistentPreRunE
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if jsonFlag {
			outputFormat = "json"
		}
		if origPreRunE != nil {
			return origPreRunE(cmd, args)
		}
		return nil
	}

	rootCmd.AddCommand(
		newInitCmd(),
		newDomainsCmd(),
		newDNSCmd(),
		newSitesCmd(),
		newLaunchCmd(),
		newConfigCmd(),
		newServerCmd(),
		newBillingCmd(),
		newVersionCmd(version),
	)

	// Hide the completion command for cleaner help
	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	return rootCmd
}

func initProviders(cfg *config.Config) {
	if cfg.APIKey != "" {
		client = valuedomain.NewClient(cfg.APIKey)
	}
	if cfg.PorkbunAPIKey != "" && cfg.PorkbunSecretKey != "" {
		porkbunClient = porkbun.NewClient(cfg.PorkbunAPIKey, cfg.PorkbunSecretKey)
	}
	if cfg.SpaceshipAPIKey != "" && cfg.SpaceshipAPISecret != "" {
		spaceshipClient = spaceship.NewClient(cfg.SpaceshipAPIKey, cfg.SpaceshipAPISecret)
	}
	if cfg.NamecheapAPIKey != "" && cfg.NamecheapAPIUser != "" {
		userName := cfg.NamecheapUserName
		if userName == "" {
			userName = cfg.NamecheapAPIUser
		}
		namecheapClient = namecheap.NewClient(cfg.NamecheapAPIUser, cfg.NamecheapAPIKey, userName, cfg.NamecheapClientIP)
	}
	if cfg.CloudflareGlobalKey != "" && cfg.CloudflareEmail != "" {
		cloudflareClient = cfprovider.NewClientGlobal(cfg.CloudflareGlobalKey, cfg.CloudflareEmail, cfg.CloudflareAccountID)
	} else if cfg.CloudflareToken != "" {
		cloudflareClient = cfprovider.NewClient(cfg.CloudflareToken)
		if cfg.CloudflareAccountID != "" {
			cloudflareClient.AccountID = cfg.CloudflareAccountID
		}
	}
}

func newVersionCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Run: func(cmd *cobra.Command, args []string) {
			if isStructuredOutput() {
				printResult("version", map[string]string{"version": version}, "", nil)
				return
			}
			fmt.Printf("regctl %s\n", version)
		},
	}
}
