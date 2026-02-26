package cli

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yukihamada/regctl/internal/billing"
	"github.com/yukihamada/regctl/internal/config"
	"github.com/yukihamada/regctl/internal/email"
	"github.com/yukihamada/regctl/internal/notify"
	"github.com/yukihamada/regctl/internal/provider"
	"github.com/yukihamada/regctl/internal/server"
	"github.com/yukihamada/regctl/internal/storage"
)

func newServerCmd() *cobra.Command {
	var port int
	var staticDir string

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the HTTP API server",
		Long: `Start the regctl HTTP API server.

The server exposes a REST API for managing domains and DNS records.
All endpoints (except /health) require Bearer token authentication.

Endpoints:
  GET  /health                    Health check (no auth)
  GET  /v1/domains                List domains
  GET  /v1/domains/{domain}       Domain details
  GET  /v1/domains/check/{domain} Check availability
  POST /v1/domains                Register domain
  GET  /v1/dns/{domain}           List DNS records
  POST /v1/dns/{domain}           Add DNS record
  DELETE /v1/dns/{domain}/{id}    Delete DNS record

Billing endpoints (when STRIPE_SECRET_KEY is set):
  POST /v1/billing/signup         Create account
  POST /v1/billing/topup          Add credit
  GET  /v1/billing/balance        Check balance
  POST /webhooks/stripe           Stripe webhook`,
		Example: `  regctl server
  regctl server --port 3000

  # Then use the API:
  curl http://localhost:8080/health
  curl -H "Authorization: Bearer YOUR_KEY" http://localhost:8080/v1/domains`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Server command skips PersistentPreRunE, so load config here
			if cfg == nil {
				var err error
				cfg, err = config.Load()
				if err != nil {
					cfg = &config.Config{}
				}
				initProviders(cfg)
			}

			// Auto-detect static dir
			if staticDir == "" {
				if _, err := os.Stat("/static/index.html"); err == nil {
					staticDir = "/static"
				}
			}

			// Initialize SQLite store for search logging
			dbPath := os.Getenv("REGCTL_DB_PATH")
			if dbPath == "" {
				dbPath = "/data/regctl.db"
			}
			var store *storage.Store
			store, err := storage.New(dbPath)
			if err != nil {
				log.Printf("WARN: failed to open database at %s: %v (search logging disabled)", dbPath, err)
			} else {
				// Start background cleanup goroutine
				go func() {
					ticker := time.NewTicker(6 * time.Hour)
					defer ticker.Stop()
					for range ticker.C {
						if err := store.CleanupOldData(); err != nil {
							log.Printf("WARN: cleanup: %v", err)
						}
					}
				}()
			}

			srvCfg := server.Config{
				Client:    client,
				APIKey:    cfg.RegctlKey,
				StaticDir: staticDir,
				Store:     store,
			}

			// Initialize billing if Stripe env vars are set
			stripeKey := os.Getenv("STRIPE_SECRET_KEY")
			signingSecret := os.Getenv("REGCTL_SIGNING_SECRET")
			webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
			successURL := os.Getenv("REGCTL_SUCCESS_URL")
			cancelURL := os.Getenv("REGCTL_CANCEL_URL")

			if stripeKey != "" && signingSecret != "" {
				if successURL == "" {
					successURL = "https://regctl.sh"
				}
				if cancelURL == "" {
					cancelURL = "https://regctl.sh"
				}
				srvCfg.BillingClient = billing.NewClient(stripeKey, signingSecret, successURL, cancelURL)
				srvCfg.SigningSecret = signingSecret
				srvCfg.WebhookSecret = webhookSecret
			}

			// Auth providers
			resendKey := os.Getenv("RESEND_API_KEY")
			if resendKey != "" {
				srvCfg.EmailClient = email.NewClient(resendKey, "noreply@regctl.sh")
			}
			srvCfg.GitHubClientID = os.Getenv("GITHUB_CLIENT_ID")
			srvCfg.GitHubClientSecret = os.Getenv("GITHUB_CLIENT_SECRET")
			srvCfg.GoogleClientID = os.Getenv("GOOGLE_CLIENT_ID")
			srvCfg.GoogleClientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
			baseURL := os.Getenv("REGCTL_BASE_URL")
			if baseURL == "" {
				baseURL = "https://regctl-api.fly.dev"
			}
			srvCfg.BaseURL = baseURL
			srvCfg.GoogleRedirectURI = baseURL + "/v1/auth/google/callback"

			// LINE Notify (残高不足アラート)
			srvCfg.LineNotify = notify.NewLineClient(os.Getenv("LINE_NOTIFY_TOKEN"))

			// Fly Machines hosting
			srvCfg.FlyAPIToken = os.Getenv("FLY_API_TOKEN")
			srvCfg.FlyAppName = os.Getenv("FLY_APP_NAME")
			srvCfg.FlyRegion = os.Getenv("FLY_REGION")
			srvCfg.InternalSecret = os.Getenv("REGCTL_INTERNAL_SECRET")

			// Multi-registrar support
			var registrars []provider.Registrar
			if spaceshipClient != nil {
				registrars = append(registrars, spaceshipClient)
			}
			if porkbunClient != nil {
				registrars = append(registrars, porkbunClient)
			}
			srvCfg.Registrars = registrars

			// DNS & NS providers (Porkbun supports both)
			if porkbunClient != nil {
				srvCfg.DNSProviders = append(srvCfg.DNSProviders, porkbunClient)
				srvCfg.NSProviders = append(srvCfg.NSProviders, porkbunClient)
			}

			srv := server.New(srvCfg)

			fmt.Println()
			color.New(color.FgCyan, color.Bold).Printf("  regctl API server starting on port %d\n", port)
			fmt.Println()
			fmt.Printf("  Health:  http://localhost:%d/health\n", port)
			fmt.Printf("  API:     http://localhost:%d/v1/\n", port)
			fmt.Println()
			if cfg.RegctlKey != "" {
				fmt.Println("  Auth:    Authorization: Bearer <REGCTL_API_KEY>")
			} else {
				color.Yellow("  Warning: No REGCTL_API_KEY set. API endpoints are unprotected.")
				fmt.Println("  Set one: regctl config set regctl_api_key <key>")
			}
			if srvCfg.BillingClient != nil {
				color.Green("  Billing: enabled (Stripe)")
			}
			if store != nil {
				color.Green("  Store:   enabled (%s)", dbPath)
			}
			if len(registrars) > 0 {
				names := make([]string, len(registrars))
				for i, r := range registrars {
					names[i] = r.Name()
				}
				color.Green("  Registrars: %s", fmt.Sprintf("%v", names))
			}
			fmt.Println()
			fmt.Println("  Press Ctrl+C to stop.")
			fmt.Println()

			return srv.ListenAndServe(port)
		},
	}

	cmd.Flags().IntVar(&port, "port", 8080, "Port to listen on")
	cmd.Flags().StringVar(&staticDir, "static-dir", "", "Directory with static files (auto-detects /static)")
	return cmd
}
