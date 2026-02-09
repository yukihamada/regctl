package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/yukihamada/regctl/internal/config"
	"github.com/yukihamada/regctl/internal/provider/valuedomain"
)

func newClientFromConfig(cfg *config.Config) *valuedomain.Client {
	return valuedomain.NewClient(cfg.APIKey)
}

// runInteractiveMode launches the guided menu when no arguments are given.
func runInteractiveMode() {
	reader := bufio.NewReader(os.Stdin)

	printWelcomeBanner()

	// Check if configured
	cfg, err := config.Load()
	if err != nil || cfg.APIKey == "" {
		fmt.Println("  It looks like regctl is not configured yet.")
		fmt.Println()
		fmt.Print("  Would you like to set it up now? [Y/n]: ")
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer == "" || answer == "y" || answer == "yes" {
			runInitWizard()
			return
		}
		fmt.Println()
		fmt.Println("  Run 'regctl init' when you're ready.")
		return
	}

	// Show interactive menu
	for {
		printMenu()
		fmt.Print("  Choose [1-7]: ")
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			runMenuListDomains(cfg)
		case "2":
			runMenuCheckDomain(cfg, reader)
		case "3":
			runMenuDNSList(cfg, reader)
		case "4":
			runMenuDNSAdd(cfg, reader)
		case "5":
			fmt.Println()
			printUsageGuide()
		case "6":
			fmt.Println()
			color.Cyan("  Run: regctl server --port 8080")
			fmt.Println("  Then access: http://localhost:8080/health")
			fmt.Println()
		case "7", "q", "quit", "exit":
			fmt.Println()
			fmt.Println("  Goodbye!")
			return
		default:
			color.Red("  Invalid choice. Enter 1-7.")
		}
	}
}

func printMenu() {
	fmt.Println()
	color.New(color.Bold).Println("  What would you like to do?")
	fmt.Println()
	fmt.Println("    1) List my domains")
	fmt.Println("    2) Check domain availability")
	fmt.Println("    3) View DNS records")
	fmt.Println("    4) Add a DNS record")
	fmt.Println("    5) Show help & examples")
	fmt.Println("    6) Start API server")
	fmt.Println("    7) Quit")
	fmt.Println()
}

func runMenuListDomains(cfg *config.Config) {
	c := newClientFromConfig(cfg)
	domains, err := c.ListDomains()
	if err != nil {
		color.Red("\n  Error: %s", err)
		return
	}
	fmt.Println()
	if len(domains) == 0 {
		fmt.Println("  No domains found in your account.")
		return
	}

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
}

func runMenuCheckDomain(cfg *config.Config, reader *bufio.Reader) {
	fmt.Println()
	fmt.Print("  Enter domain to check (e.g. example.com): ")
	domain, _ := reader.ReadString('\n')
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return
	}

	c := newClientFromConfig(cfg)
	avail, err := c.CheckAvailability(domain)
	if err != nil {
		color.Red("\n  Error: %s", err)
		return
	}

	fmt.Println()
	if avail.Available {
		color.Green("  %s is available!", domain)
		if avail.Price > 0 {
			fmt.Printf("  Price: %.0f %s\n", avail.Price, avail.Currency)
		}
		fmt.Println()
		fmt.Printf("  Register it with: regctl domains register %s\n", domain)
	} else {
		color.Red("  %s is not available.", domain)
	}
}

func runMenuDNSList(cfg *config.Config, reader *bufio.Reader) {
	fmt.Println()
	fmt.Print("  Enter domain (e.g. example.com): ")
	domain, _ := reader.ReadString('\n')
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return
	}

	c := newClientFromConfig(cfg)
	records, err := c.GetDNSRecords(domain)
	if err != nil {
		color.Red("\n  Error: %s", err)
		return
	}

	fmt.Println()
	if len(records) == 0 {
		fmt.Println("  No DNS records found.")
		return
	}

	var rows [][]string
	for _, r := range records {
		rows = append(rows, []string{
			fmt.Sprintf("%d", r.ID), r.Type, r.Name, r.Content, fmt.Sprintf("%d", r.TTL),
		})
	}
	renderTable([]string{"ID", "Type", "Name", "Content", "TTL"}, rows)
}

func runMenuDNSAdd(cfg *config.Config, reader *bufio.Reader) {
	fmt.Println()
	fmt.Print("  Domain (e.g. example.com): ")
	domain, _ := reader.ReadString('\n')
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return
	}

	fmt.Print("  Record type [A]: ")
	rtype, _ := reader.ReadString('\n')
	rtype = strings.TrimSpace(strings.ToUpper(rtype))
	if rtype == "" {
		rtype = "A"
	}

	fmt.Print("  Name [@]: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if name == "" {
		name = "@"
	}

	fmt.Print("  Content/Value: ")
	content, _ := reader.ReadString('\n')
	content = strings.TrimSpace(content)
	if content == "" {
		color.Red("  Content is required.")
		return
	}

	record := valuedomain.DNSRecord{
		Type:    rtype,
		Name:    name,
		Content: content,
		TTL:     3600,
	}

	c := newClientFromConfig(cfg)
	if err := c.AddDNSRecord(domain, record); err != nil {
		color.Red("\n  Error: %s", err)
		return
	}
	fmt.Println()
	color.Green("  DNS record added successfully!")
	fmt.Printf("  Equivalent command: regctl dns add %s -t %s -n %s -c %s\n", domain, rtype, name, content)
}
