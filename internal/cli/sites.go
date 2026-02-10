package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yukihamada/regctl/internal/hosting"
	"github.com/yukihamada/regctl/internal/provider"
	"github.com/yukihamada/regctl/internal/provider/valuedomain"
)

func newSitesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sites",
		Short: "Connect domains to hosting services",
		Long: `Set up DNS records to connect a domain to a hosting provider.

  Supports Vercel, Netlify, Cloudflare Pages, GitHub Pages, and Fly.io.

Examples:
  regctl sites create example.com                      Interactive provider selection
  regctl sites create example.com --provider vercel     Set up DNS for Vercel
  regctl sites providers                                List supported providers`,
	}

	cmd.AddCommand(
		newSitesCreateCmd(),
		newSitesProvidersCmd(),
	)

	return cmd
}

func newSitesProvidersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "providers",
		Short: "List supported hosting providers",
		Run: func(cmd *cobra.Command, args []string) {
			if isStructuredOutput() {
				var list []map[string]string
				for _, p := range hosting.Providers {
					list = append(list, map[string]string{
						"name":        p.Name,
						"slug":        p.Slug,
						"description": p.Description,
					})
				}
				printResult("sites providers", list,
					fmt.Sprintf("%d hosting providers supported", len(hosting.Providers)),
					[]string{"regctl sites create <domain> --provider <slug>"},
				)
				return
			}

			fmt.Println()
			color.New(color.Bold).Println("  Supported hosting providers")
			fmt.Println()
			for _, p := range hosting.Providers {
				fmt.Printf("  %-20s %s\n", color.CyanString(p.Slug), p.Description)
			}
			fmt.Println()
			fmt.Println("  Usage: regctl sites create <domain> --provider <slug>")
			fmt.Println()
		},
	}
}

func newSitesCreateCmd() *cobra.Command {
	var providerFlag string

	cmd := &cobra.Command{
		Use:   "create <domain>",
		Short: "Set up DNS records for a hosting provider",
		Long: `Connect a domain to a hosting service by adding the required DNS records.

  The hosting project must already exist on the provider's side.
  This command only configures DNS — no API keys for hosting services are needed.`,
		Example: `  regctl sites create example.com
  regctl sites create example.com --provider vercel
  regctl sites create example.com --provider github-pages`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]

			// Select provider
			var hp *hosting.Provider
			if providerFlag != "" {
				hp = hosting.FindBySlug(providerFlag)
				if hp == nil {
					printErrorResult("sites create", fmt.Errorf("unknown provider: %s", providerFlag),
						"List providers: regctl sites providers")
					return nil
				}
			} else {
				// Interactive selection
				hp = promptProviderSelection(domain)
				if hp == nil {
					return nil
				}
			}

			// Collect additional inputs if needed
			reader := bufio.NewReader(os.Stdin)
			var answers []string
			for _, prompt := range hp.Prompts {
				fmt.Printf("  %s: ", prompt)
				answer, _ := reader.ReadString('\n')
				answers = append(answers, strings.TrimSpace(answer))
			}

			// Generate DNS records
			records := hp.Records(domain, answers)
			if len(records) == 0 {
				printErrorResult("sites create", fmt.Errorf("no DNS records to add"), "Check your inputs")
				return nil
			}

			fmt.Println()
			color.New(color.Bold).Printf("  Setting up DNS for %s...\n", hp.Name)
			fmt.Println()

			// Add records via available DNS provider
			var added int
			for _, rec := range records {
				label := fmt.Sprintf("%-5s %-4s → %s", rec.Type, rec.Name, rec.Content)
				err := addDNSRecord(domain, rec)
				if err != nil {
					color.Red("    Adding %s  ✗ %v", label, err)
				} else {
					color.Green("    Adding %s  ✓", label)
					added++
				}
			}

			fmt.Println()
			if added == len(records) {
				printSuccess(fmt.Sprintf("  Done! %d DNS record(s) added.", added))
			} else {
				color.Yellow("  %d/%d DNS record(s) added.", added, len(records))
			}

			// Print next steps
			steps := hp.NextSteps(domain)
			if len(steps) > 0 {
				fmt.Println()
				color.New(color.Bold).Println("  Next steps:")
				for i, step := range steps {
					fmt.Printf("    %d. %s\n", i+1, step)
				}
			}
			fmt.Println()

			return nil
		},
	}

	cmd.Flags().StringVar(&providerFlag, "provider", "", "Hosting provider (vercel, netlify, cloudflare-pages, github-pages, flyio)")

	return cmd
}

// promptProviderSelection shows an interactive menu and returns the chosen provider.
func promptProviderSelection(domain string) *hosting.Provider {
	fmt.Println()
	color.New(color.Bold).Printf("  Connect %s to a hosting service\n", domain)
	fmt.Println()
	fmt.Println("  Which provider?")
	fmt.Println()

	for i, p := range hosting.Providers {
		fmt.Printf("    %d) %s\n", i+1, p.Name)
	}

	fmt.Println()
	fmt.Printf("  Choose [1-%d]: ", len(hosting.Providers))

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	var choice int
	if _, err := fmt.Sscanf(input, "%d", &choice); err != nil || choice < 1 || choice > len(hosting.Providers) {
		printErrorResult("sites create", fmt.Errorf("invalid choice: %s", input), "Enter a number between 1 and "+fmt.Sprintf("%d", len(hosting.Providers)))
		return nil
	}

	return &hosting.Providers[choice-1]
}

// addDNSRecord adds a single DNS record using the first available DNS provider.
func addDNSRecord(domain string, rec provider.DNSRecord) error {
	if namecheapClient != nil {
		return namecheapClient.AddRecord(domain, rec)
	}
	if porkbunClient != nil {
		return porkbunClient.AddRecord(domain, rec)
	}
	if cloudflareClient != nil {
		return cloudflareClient.AddRecord(domain, rec)
	}
	if spaceshipClient != nil {
		return spaceshipClient.AddRecord(domain, rec)
	}
	if client != nil {
		vdRecord := valuedomain.DNSRecord{
			Type:     rec.Type,
			Name:     rec.Name,
			Content:  rec.Content,
			TTL:      rec.TTL,
			Priority: rec.Priority,
		}
		return client.AddDNSRecord(domain, vdRecord)
	}
	return fmt.Errorf("no DNS provider configured — run: regctl init")
}
