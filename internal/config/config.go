package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const (
	configDir  = ".regctl"
	configFile = "config"
	configType = "yaml"
)

// Config holds the application configuration.
type Config struct {
	// Value Domain
	APIKey string `mapstructure:"api_key"`

	// Porkbun
	PorkbunAPIKey    string `mapstructure:"porkbun_api_key"`
	PorkbunSecretKey string `mapstructure:"porkbun_secret_key"`

	// Cloudflare
	CloudflareToken     string `mapstructure:"cloudflare_token"`
	CloudflareGlobalKey string `mapstructure:"cloudflare_global_key"`
	CloudflareEmail     string `mapstructure:"cloudflare_email"`
	CloudflareAccountID string `mapstructure:"cloudflare_account_id"`

	// General
	RegctlKey  string `mapstructure:"regctl_api_key"`
	OutputJSON bool   `mapstructure:"output_json"`
	ServerPort int    `mapstructure:"server_port"`
}

// Load reads configuration from file and environment variables.
func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home directory: %w", err)
	}

	configPath := filepath.Join(home, configDir)

	// Ensure config directory exists
	if err := os.MkdirAll(configPath, 0700); err != nil {
		return nil, fmt.Errorf("create config directory: %w", err)
	}

	viper.SetConfigName(configFile)
	viper.SetConfigType(configType)
	viper.AddConfigPath(configPath)

	// Environment variable bindings
	viper.BindEnv("api_key", "VALUEDOMAIN_API_KEY")
	viper.BindEnv("porkbun_api_key", "PORKBUN_API_KEY")
	viper.BindEnv("porkbun_secret_key", "PORKBUN_SECRET_KEY")
	viper.BindEnv("cloudflare_token", "CLOUDFLARE_API_TOKEN")
	viper.BindEnv("cloudflare_global_key", "CLOUDFLARE_GLOBAL_KEY")
	viper.BindEnv("cloudflare_email", "CLOUDFLARE_EMAIL")
	viper.BindEnv("cloudflare_account_id", "CLOUDFLARE_ACCOUNT_ID")
	viper.BindEnv("regctl_api_key", "REGCTL_API_KEY")

	// Defaults
	viper.SetDefault("server_port", 8080)
	viper.SetDefault("output_json", false)

	// Read config file (ignore not-found)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}

// Set writes a configuration key-value pair to the config file.
func Set(key, value string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home directory: %w", err)
	}

	configPath := filepath.Join(home, configDir)
	if err := os.MkdirAll(configPath, 0700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	viper.SetConfigName(configFile)
	viper.SetConfigType(configType)
	viper.AddConfigPath(configPath)

	// Read existing config
	_ = viper.ReadInConfig()

	viper.Set(key, value)

	configFilePath := filepath.Join(configPath, configFile+"."+configType)
	return viper.WriteConfigAs(configFilePath)
}

// GetConfigPath returns the path to the config file.
func GetConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, configDir, configFile+"."+configType)
}

// HasAnyProvider returns true if at least one provider is configured.
func (c *Config) HasAnyProvider() bool {
	return c.APIKey != "" || c.PorkbunAPIKey != "" || c.CloudflareToken != "" || c.CloudflareGlobalKey != ""
}
