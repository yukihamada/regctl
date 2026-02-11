package cli

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yukihamada/regctl/internal/billing"
	"github.com/yukihamada/regctl/internal/config"
	"github.com/yukihamada/regctl/internal/provider"
	cfprovider "github.com/yukihamada/regctl/internal/provider/cloudflare"
	"github.com/yukihamada/regctl/internal/provider/porkbun"
	ssprovider "github.com/yukihamada/regctl/internal/provider/spaceship"
)

func newDomainsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domains",
		Short: "Manage domains across registrars",
		Long: `Manage domains across multiple registrars.

  Compare prices, check availability, and register at the cheapest provider.

Examples:
  regctl domains check example.com         Compare prices across registrars
  regctl domains list                      List all domains from all providers
  regctl domains register example.com      Register at cheapest available registrar`,
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
		Short: "List all domains from all configured registrars",
		Example: `  regctl domains list
  regctl domains list --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var allDomains []provider.Domain
			var errors []string

			// Collect from all providers
			if porkbunClient != nil {
				domains, err := porkbunClient.ListDomains()
				if err != nil {
					errors = append(errors, fmt.Sprintf("Porkbun: %v", err))
				} else {
					allDomains = append(allDomains, domains...)
				}
			}
			if spaceshipClient != nil {
				domains, err := spaceshipClient.ListDomains()
				if err != nil {
					errors = append(errors, fmt.Sprintf("Spaceship: %v", err))
				} else {
					allDomains = append(allDomains, domains...)
				}
			}
			if cloudflareClient != nil {
				domains, err := cloudflareClient.ListDomains()
				if err != nil {
					errors = append(errors, fmt.Sprintf("Cloudflare: %v", err))
				} else {
					allDomains = append(allDomains, domains...)
				}
			}
			if namecheapClient != nil {
				domains, err := namecheapClient.ListDomains()
				if err != nil {
					errors = append(errors, fmt.Sprintf("Namecheap: %v", err))
				} else {
					allDomains = append(allDomains, domains...)
				}
			}
			if client != nil {
				vdDomains, err := client.ListDomains()
				if err != nil {
					errors = append(errors, fmt.Sprintf("Value Domain: %v", err))
				} else {
					for _, d := range vdDomains {
						name := d.DomainName
						if name == "" {
							name = d.Name
						}
						allDomains = append(allDomains, provider.Domain{
							Name:      name,
							Registrar: "Value Domain",
							Status:    d.Status,
							ExpiresAt: d.ExpiresAt,
							AutoRenew: d.AutoRenew,
						})
					}
				}
			}

			if isStructuredOutput() {
				data := map[string]interface{}{
					"domains": allDomains,
					"errors":  errors,
				}
				printResult("domains list", data,
					fmt.Sprintf("Found %d domain(s) across registrars", len(allDomains)),
					[]string{"regctl domains info <domain>", "regctl dns list <domain>"},
				)
				return nil
			}

			if len(allDomains) == 0 {
				fmt.Println("No domains found in any configured registrar.")
				if len(errors) > 0 {
					fmt.Println("\nErrors:")
					for _, e := range errors {
						fmt.Printf("  - %s\n", e)
					}
				}
				return nil
			}

			printSection(fmt.Sprintf("Domains (%d)", len(allDomains)))
			var rows [][]string
			for _, d := range allDomains {
				rows = append(rows, []string{d.Name, d.Registrar, d.Status, d.ExpiresAt})
			}
			renderTable([]string{"Domain", "Registrar", "Status", "Expires"}, rows)

			if len(errors) > 0 {
				fmt.Println()
				for _, e := range errors {
					color.Yellow("  ! %s", e)
				}
			}
			return nil
		},
	}
}

func newDomainsInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <domain>",
		Short: "Show detailed domain information",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if client == nil {
				printErrorResult("domains info", fmt.Errorf("Value Domain API key required"), "regctl config set api_key YOUR_KEY")
				return nil
			}
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
						fmt.Sprintf("regctl dns list %s", d.DomainName),
						fmt.Sprintf("regctl dns add %s -t A -n @ -c <ip>", d.DomainName),
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

type priceEntry struct {
	Registrar string  `json:"registrar"`
	RegPrice  float64 `json:"reg_price"`
	RenPrice  float64 `json:"renew_price"`
	Available bool    `json:"available"`
	CanRegAPI bool    `json:"can_register_via_api"`
	Cheapest  bool    `json:"cheapest"`
}

func newDomainsCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check <domain>",
		Short: "Compare domain prices across all registrars",
		Long: `Check domain availability and compare prices across Porkbun, Cloudflare, and more.

  Shows the cheapest registrar for the domain and highlights API-registrable options.`,
		Example: `  regctl domains check example.com
  regctl domains check mybrand.dev --format json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]
			parts := strings.SplitN(domain, ".", 2)
			if len(parts) < 2 {
				printErrorResult("domains check", fmt.Errorf("invalid domain: %s", domain), "Use format: name.tld (e.g. example.com)")
				return nil
			}
			tld := parts[1]

			// Load config if not already loaded (check skips PersistentPreRunE)
			if cfg == nil {
				var err error
				cfg, err = config.Load()
				if err == nil {
					initProviders(cfg)
				}
			}

			var prices []priceEntry
			var mu sync.Mutex
			var wg sync.WaitGroup

			// 1) Porkbun - live API check (needs auth) or static pricing fallback
			wg.Add(1)
			go func() {
				defer wg.Done()
				if porkbunClient != nil {
					avail, err := porkbunClient.CheckAvailability(domain)
					if err == nil {
						mu.Lock()
						prices = append(prices, priceEntry{
							Registrar: "Porkbun",
							RegPrice:  avail.RegPrice,
							RenPrice:  avail.RenPrice,
							Available: avail.Available,
							CanRegAPI: true,
						})
						mu.Unlock()
						return
					}
				}
				// Fallback: fetch public pricing (no auth needed)
				allPricing, err := porkbun.FetchPricingStatic()
				if err == nil {
					if p, ok := allPricing[tld]; ok {
						mu.Lock()
						prices = append(prices, priceEntry{
							Registrar: "Porkbun",
							RegPrice:  p.RegPrice,
							RenPrice:  p.RenPrice,
							Available: true,
							CanRegAPI: porkbunClient != nil,
						})
						mu.Unlock()
					}
				}
			}()

			// 2) Spaceship - live API check (if configured)
			if spaceshipClient != nil {
				wg.Add(1)
				go func() {
					defer wg.Done()
					avail, err := spaceshipClient.CheckAvailability(domain)
					if err == nil {
						// Spaceship availability API doesn't return price;
						// use static pricing from known data
						reg, ren := ssprovider.GetStaticPrice(tld)
						mu.Lock()
						prices = append(prices, priceEntry{
							Registrar: "Spaceship",
							RegPrice:  reg,
							RenPrice:  ren,
							Available: avail.Available,
							CanRegAPI: true,
						})
						mu.Unlock()
					}
				}()
			} else {
				// Fallback: static pricing without auth
				wg.Add(1)
				go func() {
					defer wg.Done()
					reg, ren := ssprovider.GetStaticPrice(tld)
					if reg > 0 {
						mu.Lock()
						prices = append(prices, priceEntry{
							Registrar: "Spaceship",
							RegPrice:  reg,
							RenPrice:  ren,
							Available: true,
							CanRegAPI: false,
						})
						mu.Unlock()
					}
				}()
			}

			// 3) Cloudflare - static pricing (no registration API)
			wg.Add(1)
			go func() {
				defer wg.Done()
				reg, renew, ok := cfprovider.GetStaticPrice(tld)
				if ok {
					mu.Lock()
					prices = append(prices, priceEntry{
						Registrar: "Cloudflare",
						RegPrice:  reg,
						RenPrice:  renew,
						Available: true,
						CanRegAPI: false,
					})
					mu.Unlock()
				}
			}()

			// 4) Value Domain - live API check (if configured)
			if client != nil {
				wg.Add(1)
				go func() {
					defer wg.Done()
					avail, err := client.CheckAvailability(domain)
					if err == nil {
						mu.Lock()
						prices = append(prices, priceEntry{
							Registrar: "Value Domain",
							RegPrice:  avail.Price,
							RenPrice:  avail.Price,
							Available: avail.Available,
							CanRegAPI: true,
						})
						mu.Unlock()
					}
				}()
			}

			// 5) Namecheap - live API check (if configured), with pricing
			if namecheapClient != nil {
				wg.Add(1)
				go func() {
					defer wg.Done()
					avail, err := namecheapClient.CheckAvailability(domain)
					if err != nil {
						return
					}
					// Fetch pricing to get the actual price
					pricing, pErr := namecheapClient.FetchPricing()
					var regPrice, renPrice float64
					if pErr == nil {
						if p, ok := pricing[tld]; ok {
							regPrice = p.RegPrice
							renPrice = p.RenPrice
						}
					}
					mu.Lock()
					prices = append(prices, priceEntry{
						Registrar: "Namecheap",
						RegPrice:  regPrice,
						RenPrice:  renPrice,
						Available: avail.Available,
						CanRegAPI: true,
					})
					mu.Unlock()
				}()
			}

			wg.Wait()

			if len(prices) == 0 {
				printErrorResult("domains check", fmt.Errorf("no pricing data available for %s", domain), "Check your network connection")
				return nil
			}

			// Sort by registration price
			sort.Slice(prices, func(i, j int) bool {
				return prices[i].RegPrice < prices[j].RegPrice
			})

			// Mark cheapest
			if len(prices) > 0 {
				prices[0].Cheapest = true
			}

			if isStructuredOutput() {
				data := map[string]interface{}{
					"domain": domain,
					"tld":    tld,
					"prices": prices,
				}
				cheapest := prices[0]
				summary := fmt.Sprintf("Cheapest: %s at $%.2f/yr (renew $%.2f/yr)", cheapest.Registrar, cheapest.RegPrice, cheapest.RenPrice)
				next := []string{}
				if cheapest.CanRegAPI {
					next = append(next, fmt.Sprintf("regctl domains register %s — Register via %s API", domain, cheapest.Registrar))
				} else {
					next = append(next, fmt.Sprintf("%s requires dashboard registration", cheapest.Registrar))
				}
				printResult("domains check", data, summary, next)
				return nil
			}

			// Table output
			fmt.Println()
			color.New(color.Bold).Printf("  Price comparison for %s\n", domain)
			fmt.Println()

			var rows [][]string
			for _, p := range prices {
				status := color.GreenString("Available")
				if !p.Available {
					status = color.RedString("Taken")
				}
				api := "Yes"
				if !p.CanRegAPI {
					api = color.YellowString("Dashboard")
				}
				tag := ""
				if p.Cheapest {
					tag = color.GreenString(" BEST")
				}
				rows = append(rows, []string{
					p.Registrar + tag,
					fmt.Sprintf("$%.2f", p.RegPrice),
					fmt.Sprintf("$%.2f", p.RenPrice),
					status,
					api,
				})
			}
			renderTable([]string{"Registrar", "Register", "Renew", "Status", "API"}, rows)

			fmt.Println()
			best := prices[0]
			if best.Available && best.CanRegAPI {
				color.Green("  Cheapest API-registrable: %s at $%.2f/yr", best.Registrar, best.RegPrice)
				fmt.Printf("\n  Register: regctl domains register %s\n", domain)
			} else if best.Available {
				color.Green("  Cheapest: %s at $%.2f/yr (dashboard only)", best.Registrar, best.RegPrice)
				// Find cheapest API option
				for _, p := range prices[1:] {
					if p.Available && p.CanRegAPI {
						fmt.Printf("  Cheapest via API: %s at $%.2f/yr\n", p.Registrar, p.RegPrice)
						fmt.Printf("\n  Register: regctl domains register %s\n", domain)
						break
					}
				}
			}
			fmt.Println()
			return nil
		},
	}
}

func newDomainsRegisterCmd() *cobra.Command {
	var registrar string
	cmd := &cobra.Command{
		Use:   "register <domain>",
		Short: "Register a domain via the cheapest available registrar",
		Example: `  regctl domains register example.com
  regctl domains register example.com --registrar porkbun`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]

			// If registrar specified, use it directly
			switch strings.ToLower(registrar) {
			case "porkbun":
				if porkbunClient == nil {
					printErrorResult("register", fmt.Errorf("Porkbun not configured"), "regctl config set porkbun_api_key YOUR_KEY")
					return nil
				}
				return registerViaPorkbun(domain)
			case "spaceship":
				if spaceshipClient == nil {
					printErrorResult("register", fmt.Errorf("Spaceship not configured"), "regctl config set spaceship_api_key YOUR_KEY")
					return nil
				}
				return registerViaSpaceship(domain)
			case "namecheap":
				if namecheapClient == nil {
					printErrorResult("register", fmt.Errorf("Namecheap not configured"), "regctl config set namecheap_api_key YOUR_KEY")
					return nil
				}
				return registerViaNamecheap(domain)
			case "valuedomain", "value-domain":
				if client == nil {
					printErrorResult("register", fmt.Errorf("Value Domain not configured"), "regctl config set api_key YOUR_KEY")
					return nil
				}
				return registerViaValueDomain(domain)
			case "":
				// Auto-select cheapest
			default:
				printErrorResult("register", fmt.Errorf("unknown registrar: %s", registrar), "Supported: porkbun, spaceship, namecheap, valuedomain")
				return nil
			}

			// Auto-select: try Spaceship first (cheapest), then Porkbun, Namecheap, Value Domain
			if spaceshipClient != nil {
				return registerViaSpaceship(domain)
			}
			if porkbunClient != nil {
				return registerViaPorkbun(domain)
			}
			if namecheapClient != nil {
				return registerViaNamecheap(domain)
			}
			if client != nil {
				return registerViaValueDomain(domain)
			}

			printErrorResult("register", fmt.Errorf("no registrar with API registration configured"),
				"Set up Spaceship: regctl config set spaceship_api_key YOUR_KEY && regctl config set spaceship_api_secret YOUR_SECRET")
			return nil
		},
	}
	cmd.Flags().StringVar(&registrar, "registrar", "", "Force a specific registrar (spaceship, porkbun, namecheap, valuedomain)")
	return cmd
}

func registerViaPorkbun(domain string) error {
	// Billing guard: estimate cost from Porkbun pricing
	var hold *holdResult
	if cfg != nil && cfg.RegctlBillingKey != "" {
		avail, err := porkbunClient.CheckAvailability(domain)
		if err == nil {
			regPriceCents := int64(avail.RegPrice * 100)
			h, holdErr := billingGuard(billing.OpDomainRegister, regPriceCents, domain)
			if holdErr != nil {
				return nil
			}
			hold = h
		}
	}

	if err := porkbunClient.RegisterDomain(domain); err != nil {
		if hold != nil {
			hold.Release()
		}
		printErrorResult("register", err, "Check availability first: regctl domains check "+domain)
		return nil
	}

	if hold != nil {
		hold.Confirm()
	}

	if isStructuredOutput() {
		printResult("register",
			map[string]interface{}{"domain": domain, "registrar": "Porkbun", "status": "registered"},
			fmt.Sprintf("Domain %s registered on Porkbun", domain),
			[]string{
				fmt.Sprintf("regctl dns list %s", domain),
				fmt.Sprintf("regctl domains list"),
			},
		)
		return nil
	}

	printSuccess(fmt.Sprintf("  Domain %s registered on Porkbun!", domain))
	printBillingInfo(hold)
	fmt.Println()
	fmt.Printf("  Next: regctl dns list %s\n\n", domain)
	return nil
}

func registerViaValueDomain(domain string) error {
	// Billing guard: estimate cost from Value Domain pricing
	var hold *holdResult
	if cfg != nil && cfg.RegctlBillingKey != "" {
		avail, err := client.CheckAvailability(domain)
		if err == nil {
			regPriceCents := int64(avail.Price * 100)
			h, holdErr := billingGuard(billing.OpDomainRegister, regPriceCents, domain)
			if holdErr != nil {
				return nil
			}
			hold = h
		}
	}

	if err := client.RegisterDomain(domain, 1); err != nil {
		if hold != nil {
			hold.Release()
		}
		printErrorResult("register", err, "Check availability first: regctl domains check "+domain)
		return nil
	}

	if hold != nil {
		hold.Confirm()
	}

	if isStructuredOutput() {
		printResult("register",
			map[string]interface{}{"domain": domain, "registrar": "Value Domain", "status": "registered"},
			fmt.Sprintf("Domain %s registered on Value Domain", domain),
			[]string{fmt.Sprintf("regctl dns list %s", domain)},
		)
		return nil
	}

	printSuccess(fmt.Sprintf("  Domain %s registered on Value Domain!", domain))
	printBillingInfo(hold)
	fmt.Println()
	fmt.Printf("  Next: regctl dns list %s\n\n", domain)
	return nil
}

func registerViaNamecheap(domain string) error {
	// Billing guard: estimate cost from Namecheap pricing
	var hold *holdResult
	if cfg != nil && cfg.RegctlBillingKey != "" {
		parts := strings.SplitN(domain, ".", 2)
		if len(parts) == 2 {
			pricing, pErr := namecheapClient.FetchPricing()
			if pErr == nil {
				if p, ok := pricing[parts[1]]; ok {
					regPriceCents := int64(p.RegPrice * 100)
					h, holdErr := billingGuard(billing.OpDomainRegister, regPriceCents, domain)
					if holdErr != nil {
						return nil
					}
					hold = h
				}
			}
		}
	}

	if err := namecheapClient.RegisterDomain(domain); err != nil {
		if hold != nil {
			hold.Release()
		}
		printErrorResult("register", err, "Check availability first: regctl domains check "+domain)
		return nil
	}

	if hold != nil {
		hold.Confirm()
	}

	if isStructuredOutput() {
		printResult("register",
			map[string]interface{}{"domain": domain, "registrar": "Namecheap", "status": "registered"},
			fmt.Sprintf("Domain %s registered on Namecheap", domain),
			[]string{
				fmt.Sprintf("regctl dns list %s", domain),
				fmt.Sprintf("regctl domains list"),
			},
		)
		return nil
	}

	printSuccess(fmt.Sprintf("  Domain %s registered on Namecheap!", domain))
	printBillingInfo(hold)
	fmt.Println()
	fmt.Printf("  Next: regctl dns list %s\n\n", domain)
	return nil
}

func registerViaSpaceship(domain string) error {
	// Billing guard: estimate cost from Spaceship static pricing
	var hold *holdResult
	if cfg != nil && cfg.RegctlBillingKey != "" {
		parts := strings.SplitN(domain, ".", 2)
		if len(parts) == 2 {
			reg, _ := ssprovider.GetStaticPrice(parts[1])
			if reg > 0 {
				regPriceCents := int64(reg * 100)
				h, holdErr := billingGuard(billing.OpDomainRegister, regPriceCents, domain)
				if holdErr != nil {
					return nil
				}
				hold = h
			}
		}
	}

	if err := spaceshipClient.RegisterDomain(domain); err != nil {
		if hold != nil {
			hold.Release()
		}
		printErrorResult("register", err, "Check availability first: regctl domains check "+domain)
		return nil
	}

	if hold != nil {
		hold.Confirm()
	}

	if isStructuredOutput() {
		printResult("register",
			map[string]interface{}{"domain": domain, "registrar": "Spaceship", "status": "registered"},
			fmt.Sprintf("Domain %s registered on Spaceship", domain),
			[]string{
				fmt.Sprintf("regctl dns list %s", domain),
				fmt.Sprintf("regctl domains list"),
			},
		)
		return nil
	}

	printSuccess(fmt.Sprintf("  Domain %s registered on Spaceship!", domain))
	printBillingInfo(hold)
	fmt.Println()
	fmt.Printf("  Next: regctl dns list %s\n\n", domain)
	return nil
}
