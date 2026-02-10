package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yukihamada/regctl/internal/provider"
	"github.com/yukihamada/regctl/internal/provider/valuedomain"
)

func newDNSCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dns",
		Short: "Manage DNS records",
		Long: `Manage DNS records for your domains.

Examples:
  regctl dns list example.com                         List all records
  regctl dns add example.com -t A -n @ -c 1.2.3.4    Add an A record
  regctl dns add example.com -t CNAME -n www -c @     Add a CNAME
  regctl dns add example.com -t MX -n @ -c mail.example.com --priority 10
  regctl dns delete example.com 123                   Delete record #123`,
	}

	cmd.AddCommand(
		newDNSListCmd(),
		newDNSAddCmd(),
		newDNSDeleteCmd(),
	)

	return cmd
}

func newDNSListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <domain>",
		Short: "List all DNS records for a domain",
		Example: `  regctl dns list example.com
  regctl dns list example.com --format json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]

			var records []provider.DNSRecord
			var err error

			// Try providers that support DNS
			if namecheapClient != nil {
				records, err = namecheapClient.ListRecords(domain)
				if err == nil {
					return printDNSRecords(domain, records)
				}
			}
			if porkbunClient != nil {
				records, err = porkbunClient.ListRecords(domain)
				if err == nil {
					return printDNSRecords(domain, records)
				}
			}
			if cloudflareClient != nil {
				records, err = cloudflareClient.ListRecords(domain)
				if err == nil {
					return printDNSRecords(domain, records)
				}
			}
			if spaceshipClient != nil {
				records, err = spaceshipClient.ListRecords(domain)
				if err == nil {
					return printDNSRecords(domain, records)
				}
			}
			if client != nil {
				vdRecords, vdErr := client.GetDNSRecords(domain)
				if vdErr == nil {
					for _, r := range vdRecords {
						records = append(records, provider.DNSRecord{
							ID:       fmt.Sprintf("%d", r.ID),
							Type:     r.Type,
							Name:     r.Name,
							Content:  r.Content,
							TTL:      r.TTL,
							Priority: r.Priority,
						})
					}
					return printDNSRecords(domain, records)
				}
				err = vdErr
			}

			if err != nil {
				printErrorResult("dns list", err, "Make sure you own this domain: regctl domains list")
			} else {
				printErrorResult("dns list", fmt.Errorf("no DNS provider configured"), "regctl config set namecheap_api_key YOUR_KEY")
			}
			return nil
		},
	}
}

func printDNSRecords(domain string, records []provider.DNSRecord) error {
	if isStructuredOutput() {
		printResult("dns list", map[string]interface{}{
			"domain":  domain,
			"records": records,
			"count":   len(records),
		},
			fmt.Sprintf("%d DNS record(s) for %s", len(records), domain),
			[]string{
				fmt.Sprintf("regctl dns add %s -t A -n @ -c <ip> — Add a record", domain),
				fmt.Sprintf("regctl dns delete %s <id> — Delete a record", domain),
			},
		)
		return nil
	}

	if len(records) == 0 {
		fmt.Printf("No DNS records found for %s.\n", domain)
		fmt.Println()
		fmt.Printf("  Add one: regctl dns add %s -t A -n @ -c 1.2.3.4\n", domain)
		return nil
	}

	printSection(fmt.Sprintf("DNS Records — %s (%d)", domain, len(records)))
	var rows [][]string
	for _, r := range records {
		rows = append(rows, []string{
			r.ID, r.Type, r.Name, r.Content, fmt.Sprintf("%d", r.TTL),
		})
	}
	renderTable([]string{"ID", "Type", "Name", "Content", "TTL"}, rows)
	return nil
}

func newDNSAddCmd() *cobra.Command {
	var (
		recordType string
		name       string
		content    string
		ttl        int
		priority   int
	)

	cmd := &cobra.Command{
		Use:   "add <domain>",
		Short: "Add a DNS record",
		Long: `Add a new DNS record to a domain.

Common record types:
  A      IPv4 address         -c 1.2.3.4
  AAAA   IPv6 address         -c 2001:db8::1
  CNAME  Alias                -c other.example.com
  MX     Mail server          -c mail.example.com --priority 10
  TXT    Text record          -c "v=spf1 include:..."
  NS     Nameserver           -c ns1.example.com`,
		Example: `  regctl dns add example.com -t A -n @ -c 1.2.3.4
  regctl dns add example.com -t CNAME -n www -c example.com
  regctl dns add example.com -t MX -n @ -c mail.example.com --priority 10
  regctl dns add example.com -t TXT -n @ -c "v=spf1 include:_spf.google.com ~all"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]
			rec := provider.DNSRecord{
				Type:     recordType,
				Name:     name,
				Content:  content,
				TTL:      ttl,
				Priority: priority,
			}

			var err error
			if namecheapClient != nil {
				err = namecheapClient.AddRecord(domain, rec)
			} else if porkbunClient != nil {
				err = porkbunClient.AddRecord(domain, rec)
			} else if cloudflareClient != nil {
				err = cloudflareClient.AddRecord(domain, rec)
			} else if spaceshipClient != nil {
				err = spaceshipClient.AddRecord(domain, rec)
			} else if client != nil {
				vdRecord := valuedomain.DNSRecord{
					Type:     recordType,
					Name:     name,
					Content:  content,
					TTL:      ttl,
					Priority: priority,
				}
				err = client.AddDNSRecord(domain, vdRecord)
			} else {
				printErrorResult("dns add", fmt.Errorf("no DNS provider configured"), "regctl config set namecheap_api_key YOUR_KEY")
				return nil
			}

			if err != nil {
				printErrorResult("dns add", err, "Check existing records: regctl dns list "+domain)
				return nil
			}

			if isStructuredOutput() {
				printResult("dns add", map[string]interface{}{
					"domain": domain,
					"record": rec,
					"status": "created",
				},
					fmt.Sprintf("Added %s record '%s' -> '%s' to %s", recordType, name, content, domain),
					[]string{
						fmt.Sprintf("regctl dns list %s — Verify the record was added", domain),
					},
				)
				return nil
			}

			printSuccess(fmt.Sprintf("  DNS record added to %s", domain))
			fmt.Printf("  %s %s -> %s (TTL: %d)\n", recordType, name, content, ttl)
			fmt.Println()
			return nil
		},
	}

	cmd.Flags().StringVarP(&recordType, "type", "t", "A", "Record type (A, AAAA, CNAME, MX, TXT, NS)")
	cmd.Flags().StringVarP(&name, "name", "n", "@", "Record name (@ for root)")
	cmd.Flags().StringVarP(&content, "content", "c", "", "Record value (IP, hostname, text)")
	cmd.Flags().IntVar(&ttl, "ttl", 3600, "Time to live in seconds")
	cmd.Flags().IntVar(&priority, "priority", 0, "Priority (for MX records)")
	cmd.MarkFlagRequired("content")

	return cmd
}

func newDNSDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <domain> <record-id>",
		Short: "Delete a DNS record",
		Example: `  regctl dns list example.com        # Find the record ID first
  regctl dns delete example.com 42   # Delete record #42`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]
			recordID := args[1]

			var err error
			if namecheapClient != nil {
				err = namecheapClient.DeleteRecord(domain, recordID)
			} else if porkbunClient != nil {
				err = porkbunClient.DeleteRecord(domain, recordID)
			} else if cloudflareClient != nil {
				err = cloudflareClient.DeleteRecord(domain, recordID)
			} else if spaceshipClient != nil {
				err = spaceshipClient.DeleteRecord(domain, recordID)
			} else if client != nil {
				// Value Domain uses integer IDs
				var intID int
				if _, parseErr := fmt.Sscanf(recordID, "%d", &intID); parseErr != nil {
					printErrorResult("dns delete", fmt.Errorf("invalid record ID: %s", recordID),
						"Find record IDs with: regctl dns list "+domain)
					return nil
				}
				err = client.DeleteDNSRecord(domain, intID)
			} else {
				printErrorResult("dns delete", fmt.Errorf("no DNS provider configured"), "regctl config set namecheap_api_key YOUR_KEY")
				return nil
			}

			if err != nil {
				printErrorResult("dns delete", err,
					"List records first: regctl dns list "+domain)
				return nil
			}

			if isStructuredOutput() {
				printResult("dns delete", map[string]interface{}{
					"domain":    domain,
					"record_id": recordID,
					"status":    "deleted",
				},
					fmt.Sprintf("Deleted DNS record #%s from %s", recordID, domain),
					[]string{
						fmt.Sprintf("regctl dns list %s — Verify the record was removed", domain),
					},
				)
				return nil
			}

			printSuccess(fmt.Sprintf("  DNS record #%s deleted from %s", recordID, domain))
			fmt.Println()
			return nil
		},
	}
}
