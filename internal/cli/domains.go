package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newDomainsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domains",
		Short: "Manage domains",
		Long: `Manage your Value Domain domains.

Examples:
  regctl domains list                      List all domains
  regctl domains list --format json        List as JSON (AI-friendly)
  regctl domains info example.com          Show domain details
  regctl domains check example.com         Check availability
  regctl domains register example.com      Register a domain`,
	}

	cmd.AddCommand(
		newDomainsListCmd(),
		newDomainsInfoCmd(),
		newDomainsCheckCmd(),
		newDomainsRegisterCmd(),
	)

	return cmd
}

func newDomainsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all domains in your account",
		Example: `  regctl domains list
  regctl domains list --format json
  regctl domains list --format ai`,
		RunE: func(cmd *cobra.Command, args []string) error {
			domains, err := client.ListDomains()
			if err != nil {
				printErrorResult("domains list", err, "Check your API key with: regctl config show")
				return nil
			}

			if isStructuredOutput() {
				printResult("domains list", domains,
					fmt.Sprintf("Found %d domain(s)", len(domains)),
					[]string{
						"regctl domains info <domain> — View details for a specific domain",
						"regctl dns list <domain> — View DNS records",
					},
				)
				return nil
			}

			if len(domains) == 0 {
				fmt.Println("No domains found in your account.")
				fmt.Println()
				fmt.Println("  Check availability: regctl domains check example.com")
				fmt.Println("  Register a domain:  regctl domains register example.com")
				return nil
			}

			printSection(fmt.Sprintf("Domains (%d)", len(domains)))
			var rows [][]string
			for _, d := range domains {
				name := d.DomainName
				if name == "" {
					name = d.Name
				}
				rows = append(rows, []string{
					fmt.Sprintf("%d", d.ID), name, d.Status, d.ExpiresAt,
				})
			}
			renderTable([]string{"ID", "Domain", "Status", "Expires"}, rows)
			return nil
		},
	}
}

func newDomainsInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <domain>",
		Short: "Show detailed domain information",
		Example: `  regctl domains info example.com
  regctl domains info example.com --format json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			detail, err := client.GetDomainInfo(args[0])
			if err != nil {
				printErrorResult("domains info", err, "Make sure the domain is in your account: regctl domains list")
				return nil
			}

			d := detail.Domain
			if isStructuredOutput() {
				printResult("domains info", d,
					fmt.Sprintf("Domain: %s, Status: %s", d.DomainName, d.Status),
					[]string{
						fmt.Sprintf("regctl dns list %s — View DNS records", d.DomainName),
						fmt.Sprintf("regctl dns add %s -t A -n @ -c <ip> — Add a DNS record", d.DomainName),
					},
				)
				return nil
			}

			printSection(d.DomainName)
			printKeyValue("Status", d.Status)
			printKeyValue("Expires", d.ExpirationDate)
			printKeyValue("Auto-renew", fmt.Sprintf("%v", d.AutoRenew))
			printKeyValue("Locked", fmt.Sprintf("%v", d.Locked))
			printKeyValue("Privacy", fmt.Sprintf("%v", d.Privacy))
			if len(d.Nameservers) > 0 {
				printKeyValue("Nameservers", "")
				for _, ns := range d.Nameservers {
					fmt.Printf("                 %s\n", ns)
				}
			}
			return nil
		},
	}
}

func newDomainsCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check <domain>",
		Short: "Check if a domain is available for registration",
		Example: `  regctl domains check example.com
  regctl domains check mybrand.jp --format json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]
			avail, err := client.CheckAvailability(domain)
			if err != nil {
				printErrorResult("domains check", err, "Check your network connection")
				return nil
			}

			if isStructuredOutput() {
				data := map[string]interface{}{
					"domain":    domain,
					"available": avail.Available,
					"premium":   avail.Premium,
					"price":     avail.Price,
					"currency":  avail.Currency,
				}
				next := []string{}
				if avail.Available {
					next = append(next, fmt.Sprintf("regctl domains register %s — Register this domain", domain))
				} else {
					next = append(next, "Try a different domain name or TLD (.net, .jp, .io, etc.)")
				}
				summary := fmt.Sprintf("%s is available", domain)
				if !avail.Available {
					summary = fmt.Sprintf("%s is NOT available", domain)
				}
				printResult("domains check", data, summary, next)
				return nil
			}

			fmt.Println()
			if avail.Available {
				color.New(color.FgGreen, color.Bold).Printf("  %s is available!\n", domain)
				if avail.Price > 0 {
					fmt.Printf("  Price: %.0f %s\n", avail.Price, avail.Currency)
				}
				fmt.Println()
				fmt.Printf("  Register: regctl domains register %s\n", domain)
			} else {
				color.New(color.FgRed, color.Bold).Printf("  %s is not available.\n", domain)
				fmt.Println()
				fmt.Println("  Try a different name or TLD (.net, .jp, .io, .dev, etc.)")
			}
			fmt.Println()
			return nil
		},
	}
}

func newDomainsRegisterCmd() *cobra.Command {
	var years int
	cmd := &cobra.Command{
		Use:   "register <domain>",
		Short: "Register a new domain",
		Example: `  regctl domains register example.com
  regctl domains register example.com --years 2`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]
			if err := client.RegisterDomain(domain, years); err != nil {
				printErrorResult("domains register", err,
					"Check availability first: regctl domains check "+domain)
				return nil
			}

			if isStructuredOutput() {
				printResult("domains register",
					map[string]interface{}{"domain": domain, "years": years, "status": "registered"},
					fmt.Sprintf("Domain %s registered for %d year(s)", domain, years),
					[]string{
						fmt.Sprintf("regctl dns list %s — View DNS records", domain),
						fmt.Sprintf("regctl domains info %s — View domain details", domain),
					},
				)
				return nil
			}

			printSuccess(fmt.Sprintf("  Domain %s registered successfully!", domain))
			fmt.Println()
			fmt.Printf("  Next: regctl dns list %s\n", domain)
			fmt.Println()
			return nil
		},
	}
	cmd.Flags().IntVar(&years, "years", 1, "Registration period in years")
	return cmd
}
