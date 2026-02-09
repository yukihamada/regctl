package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yukihamada/regctl/internal/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long: `Manage regctl configuration.

Configuration is stored in ~/.regctl/config.yaml.
Environment variables take precedence over the config file.

Environment variables:
  VALUEDOMAIN_API_KEY   Your Value Domain API key
  REGCTL_API_KEY        API key for the regctl HTTP server`,
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
  api_key         Your Value Domain API key
  regctl_api_key  API key for the regctl HTTP server
  server_port     Port for the HTTP server (default: 8080)`,
		Example: `  regctl config set api_key YOUR_API_KEY
  regctl config set server_port 3000`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]

			validKeys := map[string]bool{
				"api_key":        true,
				"regctl_api_key": true,
				"server_port":    true,
			}
			if !validKeys[key] {
				return fmt.Errorf("unknown key: %s\n\nValid keys: api_key, regctl_api_key, server_port", key)
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

			keys := []string{"api_key", "regctl_api_key", "server_port"}

			if isStructuredOutput() {
				data := map[string]interface{}{
					"config_file": config.GetConfigPath(),
				}
				for _, key := range keys {
					val := viper.GetString(key)
					if (key == "api_key" || key == "regctl_api_key") && len(val) > 8 {
						val = val[:4] + "****" + val[len(val)-4:]
					}
					data[key] = val
				}
				printResult("config show", data, "Current configuration", nil)
				return nil
			}

			printSection("Configuration")
			fmt.Printf("  File: %s\n\n", config.GetConfigPath())

			for _, key := range keys {
				val := viper.GetString(key)
				if (key == "api_key" || key == "regctl_api_key") && len(val) > 8 {
					val = val[:4] + "****" + val[len(val)-4:]
				}
				if val == "" {
					val = color.New(color.FgYellow).Sprint("(not set)")
				}
				printKeyValue(key, val)
			}
			fmt.Println()
			return nil
		},
	}
}
