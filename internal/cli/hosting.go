package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newHostingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hosting",
		Short: "Manage hosted sites on regctl infrastructure",
		Long: `Deploy and manage static sites with regctl hosting.

  regctl hosting create <domain>       Create a new site
  regctl hosting deploy <domain> [dir] Deploy files (default: current dir)
  regctl hosting status <domain>       Show site status + usage
  regctl hosting list                  List your sites
  regctl hosting delete <domain>       Delete a site`,
	}

	cmd.AddCommand(
		newHostingCreateCmd(),
		newHostingDeployCmd(),
		newHostingStatusCmd(),
		newHostingListCmd(),
		newHostingDeleteCmd(),
	)
	return cmd
}

func newHostingCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <domain>",
		Short: "Create a new hosted site",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]
			ac := NewAPIClient(cfg.RegctlAPIURL, cfg.RegctlBillingKey)

			data, err := ac.CreateSite(domain)
			if err != nil {
				return err
			}

			if isStructuredOutput() {
				printResult("hosting create", json.RawMessage(data), "", nil)
				return nil
			}

			var site struct {
				Domain string `json:"domain"`
				Status string `json:"status"`
			}
			json.Unmarshal(data, &site)

			fmt.Println()
			color.New(color.FgGreen, color.Bold).Printf("  Site created: %s\n", site.Domain)
			fmt.Printf("  Status: %s\n", site.Status)
			fmt.Println()
			fmt.Println("  Next: deploy your files")
			color.Cyan("    regctl hosting deploy %s .", domain)
			fmt.Println()
			return nil
		},
	}
}

func newHostingDeployCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "deploy <domain> [path]",
		Short: "Deploy files to a hosted site",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]
			dir := "."
			if len(args) > 1 {
				dir = args[1]
			}

			// Create tar.gz of the directory
			var buf bytes.Buffer
			if err := createTarGz(&buf, dir); err != nil {
				return fmt.Errorf("create archive: %w", err)
			}

			ac := NewAPIClient(cfg.RegctlAPIURL, cfg.RegctlBillingKey)
			data, err := ac.DeploySite(domain, buf.Bytes())
			if err != nil {
				return err
			}

			if isStructuredOutput() {
				printResult("hosting deploy", json.RawMessage(data), "", nil)
				return nil
			}

			var result struct {
				Deployed string `json:"deployed"`
				Size     int    `json:"size"`
			}
			json.Unmarshal(data, &result)

			fmt.Println()
			color.New(color.FgGreen, color.Bold).Printf("  Deployed to %s\n", result.Deployed)
			fmt.Printf("  Upload size: %d bytes\n", result.Size)
			fmt.Println()
			return nil
		},
	}
}

func newHostingStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <domain>",
		Short: "Show site status and usage",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]
			ac := NewAPIClient(cfg.RegctlAPIURL, cfg.RegctlBillingKey)

			data, err := ac.GetSiteStatus(domain)
			if err != nil {
				return err
			}

			if isStructuredOutput() {
				printResult("hosting status", json.RawMessage(data), "", nil)
				return nil
			}

			var result struct {
				Site struct {
					Domain    string `json:"domain"`
					Status    string `json:"status"`
					Tier      string `json:"tier"`
					MaxReqDay int    `json:"max_req_day"`
					CreatedAt string `json:"created_at"`
				} `json:"site"`
				TodayUsage struct {
					RequestCount int64 `json:"request_count"`
				} `json:"today_usage"`
			}
			json.Unmarshal(data, &result)

			fmt.Println()
			color.New(color.FgCyan, color.Bold).Printf("  %s\n", result.Site.Domain)
			fmt.Println()
			fmt.Printf("  Status:     %s\n", colorStatus(result.Site.Status))
			fmt.Printf("  Tier:       %s\n", result.Site.Tier)
			fmt.Printf("  Requests:   %d / %d today\n", result.TodayUsage.RequestCount, result.Site.MaxReqDay)
			fmt.Printf("  Created:    %s\n", result.Site.CreatedAt)
			fmt.Println()
			return nil
		},
	}
}

func newHostingListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List your hosted sites",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := NewAPIClient(cfg.RegctlAPIURL, cfg.RegctlBillingKey)

			data, err := ac.ListSites()
			if err != nil {
				return err
			}

			if isStructuredOutput() {
				printResult("hosting list", json.RawMessage(data), "", nil)
				return nil
			}

			var sites []struct {
				Domain string `json:"domain"`
				Status string `json:"status"`
				Tier   string `json:"tier"`
			}
			json.Unmarshal(data, &sites)

			if len(sites) == 0 {
				fmt.Println()
				fmt.Println("  No sites yet.")
				fmt.Println()
				fmt.Println("  Create one:")
				color.Cyan("    regctl hosting create example.com")
				fmt.Println()
				return nil
			}

			fmt.Println()
			color.New(color.Bold).Printf("  %-30s %-12s %s\n", "DOMAIN", "STATUS", "TIER")
			for _, s := range sites {
				fmt.Printf("  %-30s %-12s %s\n", s.Domain, colorStatus(s.Status), s.Tier)
			}
			fmt.Println()
			return nil
		},
	}
}

func newHostingDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <domain>",
		Short: "Delete a hosted site",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]
			ac := NewAPIClient(cfg.RegctlAPIURL, cfg.RegctlBillingKey)

			if err := ac.DeleteSite(domain); err != nil {
				return err
			}

			if isStructuredOutput() {
				printResult("hosting delete", map[string]string{"deleted": domain}, "", nil)
				return nil
			}

			fmt.Println()
			color.New(color.FgYellow).Printf("  Deleted site: %s\n", domain)
			fmt.Println()
			return nil
		},
	}
}

func colorStatus(status string) string {
	switch status {
	case "active":
		return color.GreenString(status)
	case "suspended":
		return color.RedString(status)
	case "provisioning":
		return color.YellowString(status)
	default:
		return status
	}
}

// createTarGz creates a gzipped tar archive of a directory.
func createTarGz(w io.Writer, dir string) error {
	gz := gzip.NewWriter(w)
	defer gz.Close()

	tw := tar.NewWriter(gz)
	defer tw.Close()

	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden dirs and common non-deploy directories
		name := info.Name()
		if info.IsDir() && (strings.HasPrefix(name, ".") || name == "node_modules") {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasPrefix(name, ".") {
			return nil
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(tw, f)
		return err
	})
}
