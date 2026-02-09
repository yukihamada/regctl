package cli

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yukihamada/regctl/internal/config"
	"github.com/yukihamada/regctl/internal/provider/valuedomain"
)

var (
	cfg    *config.Config
	client *valuedomain.Client
)

// NewRootCmd creates the root command for regctl.
func NewRootCmd(version string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "regctl",
		Short: "Domain management CLI for Value Domain",
		Long: `regctl — Domain Management Made Easy

  Manage domains and DNS records via the Value Domain API.
  Supports table, JSON, and AI-friendly output formats.

  First time? Run:
    regctl init

  Examples:
    regctl domains list              List all your domains
    regctl domains check example.com Check if a domain is available
    regctl dns list example.com      Show DNS records
    regctl dns add example.com -t A -n @ -c 1.2.3.4
    regctl server                    Start API server

  AI-friendly output:
    regctl domains list --format json
    regctl dns list example.com --format ai`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Commands that don't need API key
			skip := map[string]bool{
				"init": true, "set": true, "show": true,
				"help": true, "version": true, "completion": true,
			}
			if skip[cmd.Name()] {
				return nil
			}

			var err error
			cfg, err = config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			if cfg.APIKey == "" {
				fmt.Println()
				color.New(color.FgYellow, color.Bold).Println("  API key is not configured.")
				fmt.Println()
				fmt.Println("  Quick setup (interactive wizard):")
				color.Cyan("    regctl init")
				fmt.Println()
				fmt.Println("  Or set it directly:")
				color.Cyan("    regctl config set api_key YOUR_KEY")
				fmt.Println()
				fmt.Println("  Or use an environment variable:")
				color.Cyan("    export VALUEDOMAIN_API_KEY=YOUR_KEY")
				fmt.Println()
				fmt.Println("  Get your API key at:")
				color.New(color.FgCyan, color.Underline).Println("    https://www.value-domain.com/api/")
				fmt.Println()
				os.Exit(1)
			}

			client = valuedomain.NewClient(cfg.APIKey)
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
		newConfigCmd(),
		newServerCmd(),
		newVersionCmd(version),
	)

	// Hide the completion command for cleaner help
	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	return rootCmd
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
