package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yukihamada/regctl/internal/server"
)

func newServerCmd() *cobra.Command {
	var port int

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
  DELETE /v1/dns/{domain}/{id}    Delete DNS record`,
		Example: `  regctl server
  regctl server --port 3000

  # Then use the API:
  curl http://localhost:8080/health
  curl -H "Authorization: Bearer YOUR_KEY" http://localhost:8080/v1/domains`,
		RunE: func(cmd *cobra.Command, args []string) error {
			srv := server.New(client, cfg.RegctlKey)

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
			fmt.Println()
			fmt.Println("  Press Ctrl+C to stop.")
			fmt.Println()

			return srv.ListenAndServe(port)
		},
	}

	cmd.Flags().IntVar(&port, "port", 8080, "Port to listen on")
	return cmd
}
