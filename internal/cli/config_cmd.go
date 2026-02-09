package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yukihamada/regctl/internal/config"
)

var allConfigKeys = []string{
	"api_key", "porkbun_api_key", "porkbun_secret_key",
	"spaceship_api_key", "spaceship_api_secret",
	"cloudflare_token", "cloudflare_global_key", "cloudflare_email", "cloudflare_account_id",
	"regctl_api_key", "server_port",
}

var secretKeys = map[string]bool{
	"api_key": true, "porkbun_api_key": true, "porkbun_secret_key": true,
	"spaceship_api_key": true, "spaceship_api_secret": true,
	"cloudflare_token": true, "cloudflare_global_key": true, "regctl_api_key": true,
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long: `Manage regctl configuration.

Configuration is stored in ~/.regctl/config.yaml.
Environment variables take precedence over the config file.

Environment variables:
  VALUEDOMAIN_API_KEY    Value Domain API key
  PORKBUN_API_KEY        Porkbun API key
  PORKBUN_SECRET_KEY     Porkbun secret key
  SPACESHIP_API_KEY      Spaceship API key
  SPACESHIP_API_SECRET   Spaceship API secret
  CLOUDFLARE_API_TOKEN   Cloudflare API token
  CLOUDFLARE_GLOBAL_KEY  Cloudflare Global API key
  CLOUDFLARE_EMAIL       Cloudflare account email
  CLOUDFLARE_ACCOUNT_ID  Cloudflare account ID
  REGCTL_API_KEY         API key for the regctl HTTP server`,
	}

	cmd.AddCommand(
		newConfigSetCmd(),
		newConfigShowCmd(),
	)

	return cmd
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long: `Set a configuration value.

Valid keys:
  api_key               Value Domain API key
  porkbun_api_key       Porkbun API key
  porkbun_secret_key    Porkbun secret key
  spaceship_api_key     Spaceship API key
  spaceship_api_secret  Spaceship API secret
  cloudflare_token      Cloudflare API token (Bearer)
  cloudflare_global_key Cloudflare Global API key
  cloudflare_email      Cloudflare account email
  cloudflare_account_id Cloudflare account ID
  regctl_api_key        API key for the regctl HTTP server
  server_port           Port for the HTTP server (default: 8080)`,
		Example: `  regctl config set porkbun_api_key pk1_xxxx
  regctl config set porkbun_secret_key sk1_xxxx
  regctl config set cloudflare_global_key xxxx
  regctl config set cloudflare_email you@example.com`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]

			validKeys := make(map[string]bool)
			for _, k := range allConfigKeys {
				validKeys[k] = true
			}
			if !validKeys[key] {
				return fmt.Errorf("unknown key: %s\n\nValid keys: %v", key, allConfigKeys)
			}

			if err := config.Set(key, value); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			if isStructuredOutput() {
				printResult("config set", map[string]string{"key": key, "status": "updated"}, "Configuration updated", nil)
				return nil
			}

			color.Green("  Configuration updated: %s", key)
			fmt.Printf("  Saved to: %s\n", config.GetConfigPath())
			fmt.Println()
			return nil
		},
	}
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "show",
		Short:   "Show current configuration",
		Example: "  regctl config show\n  regctl config show --format json",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.Load()
			if err != nil {
				return err
			}

			if isStructuredOutput() {
				data := map[string]interface{}{
					"config_file": config.GetConfigPath(),
				}
				for _, key := range allConfigKeys {
					val := viper.GetString(key)
					if secretKeys[key] && len(val) > 8 {
						val = val[:4] + "****" + val[len(val)-4:]
					}
					data[key] = val
				}
				printResult("config show", data, "Current configuration", nil)
				return nil
			}

			printSection("Configuration")
			fmt.Printf("  File: %s\n\n", config.GetConfigPath())

			sections := []struct {
				name string
				keys []string
			}{
				{"Value Domain", []string{"api_key"}},
				{"Porkbun", []string{"porkbun_api_key", "porkbun_secret_key"}},
				{"Spaceship", []string{"spaceship_api_key", "spaceship_api_secret"}},
				{"Cloudflare", []string{"cloudflare_token", "cloudflare_global_key", "cloudflare_email", "cloudflare_account_id"}},
				{"General", []string{"regctl_api_key", "server_port"}},
			}

			for _, s := range sections {
				color.New(color.Bold).Printf("  [%s]\n", s.name)
				for _, key := range s.keys {
					val := viper.GetString(key)
					if secretKeys[key] && len(val) > 8 {
						val = val[:4] + "****" + val[len(val)-4:]
					}
					if val == "" {
						val = color.New(color.FgYellow).Sprint("(not set)")
					}
					printKeyValue(key, val)
				}
				fmt.Println()
			}
			return nil
		},
	}
}
