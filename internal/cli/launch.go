package cli

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yukihamada/regctl/internal/billing"
	"github.com/yukihamada/regctl/internal/config"
	"github.com/yukihamada/regctl/internal/hosting"
	cfprovider "github.com/yukihamada/regctl/internal/provider/cloudflare"
	"github.com/yukihamada/regctl/internal/provider/porkbun"
	ssprovider "github.com/yukihamada/regctl/internal/provider/spaceship"
	"github.com/yukihamada/regctl/internal/provider/netim"
)

func newLaunchCmd() *cobra.Command {
	var registrarFlag string
	var providerFlag string

	cmd := &cobra.Command{
		Use:   "launch <domain>",
		Short: "Register a domain and connect it to hosting in one step",
		Long: `Register a domain and set up DNS for your hosting provider — all in one command.

  Combines "domains register" and "sites create" into a single workflow.

Examples:
  regctl launch example.com
  regctl launch example.com --registrar spaceship --provider vercel
  regctl launch example.com --provider github-pages`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]

			parts := strings.SplitN(domain, ".", 2)
			if len(parts) < 2 {
				printErrorResult("launch", fmt.Errorf("invalid domain: %s", domain), "Use format: name.tld (e.g. example.com)")
				return nil
			}
			tld := parts[1]

			// Load config and providers (launch is in the skip list, so we do it ourselves)
			if cfg == nil {
				var err error
				cfg, err = config.Load()
				if err == nil {
					initProviders(cfg)
				}
			}

			reader := bufio.NewReader(os.Stdin)
			bold := color.New(color.Bold)

			fmt.Println()
			bold.Printf("  Launch %s\n", domain)

			// ── Step 1: Check availability ──
			fmt.Println()
			bold.Println("  Step 1: Check availability")
			fmt.Printf("    Checking %s...\n", domain)

			prices := checkPrices(domain, tld)
			if len(prices) == 0 {
				printErrorResult("launch", fmt.Errorf("no pricing data available for %s", domain), "Check your network connection")
				return nil
			}

			// Filter to available only
			var available []priceEntry
			for _, p := range prices {
				if p.Available {
					available = append(available, p)
				}
			}
			if len(available) == 0 {
				color.Red("    %s is not available at any registrar.", domain)
				return nil
			}

			// Pick registrar
			var chosen priceEntry
			if registrarFlag != "" {
				found := false
				for _, p := range available {
					if strings.EqualFold(p.Registrar, registrarFlag) || strings.EqualFold(strings.ReplaceAll(p.Registrar, " ", ""), registrarFlag) {
						chosen = p
						found = true
						break
					}
				}
				if !found {
					printErrorResult("launch", fmt.Errorf("%s not available or not configured for %s", registrarFlag, domain),
						"Supported: spaceship, porkbun, namecheap, valuedomain, cloudflare")
					return nil
				}
			} else {
				// Auto-select cheapest API-registrable
				for _, p := range available {
					if p.CanRegAPI {
						chosen = p
						break
					}
				}
				if chosen.Registrar == "" {
					chosen = available[0]
				}
			}

			color.Green("    Available on %s at $%.2f/yr", chosen.Registrar, chosen.RegPrice)

			if !chosen.CanRegAPI {
				color.Yellow("\n    %s does not support API registration.", chosen.Registrar)
				fmt.Println("    Register manually, then re-run with --provider to set up DNS.")
				return nil
			}

			// ── Step 2: Register domain ──
			fmt.Println()
			bold.Println("  Step 2: Register domain")

			// Billing guard: check balance and create hold
			regPriceCents := int64(chosen.RegPrice * 100)
			hold, holdErr := billingGuard(billing.OpDomainRegister, regPriceCents, domain)
			if holdErr != nil {
				return nil
			}
			if hold != nil {
				fmt.Printf("    Estimated cost: $%.2f (incl. %d%% markup)\n",
					float64(hold.CostCents)/100, billing.MarkupPercent)
				fmt.Printf("    Balance after: $%.2f\n", float64(hold.BalanceAfter)/100)
			}

			if registrarFlag == "" {
				// Interactive confirmation
				fmt.Printf("    Register %s on %s? [Y/n]: ", domain, chosen.Registrar)
				answer, _ := reader.ReadString('\n')
				answer = strings.TrimSpace(strings.ToLower(answer))
				if answer != "" && answer != "y" && answer != "yes" {
					fmt.Println("    Cancelled.")
					if hold != nil {
						hold.Release()
					}
					return nil
				}
			}

			fmt.Printf("    Registering...")
			regErr := registerDomain(domain, chosen.Registrar)
			if regErr != nil {
				color.Red(" Failed: %v", regErr)
				if hold != nil {
					hold.Release()
				}
				return nil
			}
			color.Green(" Domain registered!")

			// Confirm hold on success
			if hold != nil {
				hold.Confirm()
			}

			// ── Step 3: Connect to hosting ──
			fmt.Println()
			bold.Println("  Step 3: Connect to hosting")

			var hp *hosting.Provider
			if providerFlag != "" {
				hp = hosting.FindBySlug(providerFlag)
				if hp == nil {
					printErrorResult("launch", fmt.Errorf("unknown provider: %s", providerFlag),
						"List providers: regctl sites providers")
					return nil
				}
			} else {
				// Interactive selection with "Skip" option
				hp = promptLaunchProviderSelection(reader)
				// hp == nil means user chose to skip
			}

			var dnsAdded int
			var dnsTotal int

			if hp != nil {
				// Collect additional inputs if needed
				var answers []string
				for _, prompt := range hp.Prompts {
					fmt.Printf("    %s: ", prompt)
					answer, _ := reader.ReadString('\n')
					answers = append(answers, strings.TrimSpace(answer))
				}

				records := hp.Records(domain, answers)
				dnsTotal = len(records)

				fmt.Println()
				fmt.Printf("    Setting up DNS for %s...\n", hp.Name)

				for _, rec := range records {
					label := fmt.Sprintf("%-5s %-4s -> %s", rec.Type, rec.Name, rec.Content)
					err := addDNSRecord(domain, rec)
					if err != nil {
						color.Red("      Adding %s  x %v", label, err)
					} else {
						color.Green("      Adding %s  done", label)
						dnsAdded++
					}
				}
			}

			// ── Summary ──
			fmt.Println()
			bold.Println("  Done!")
			fmt.Println()
			bold.Println("  Summary:")
			fmt.Printf("    Domain:   %s (registered on %s)\n", domain, chosen.Registrar)
			if hp != nil {
				fmt.Printf("    Hosting:  %s (%d/%d DNS records added)\n", hp.Name, dnsAdded, dnsTotal)
			} else {
				fmt.Println("    Hosting:  skipped")
			}
			printBillingInfo(hold)

			// Next steps
			if hp != nil {
				steps := hp.NextSteps(domain)
				if len(steps) > 0 {
					fmt.Println()
					bold.Println("  Next steps:")
					for i, step := range steps {
						fmt.Printf("    %d. %s\n", i+1, step)
					}
				}
			} else {
				fmt.Println()
				bold.Println("  Next steps:")
				fmt.Printf("    1. Connect to hosting: regctl sites create %s\n", domain)
				fmt.Printf("    2. Verify DNS: regctl dns list %s\n", domain)
			}
			fmt.Println()

			return nil
		},
	}

	cmd.Flags().StringVar(&registrarFlag, "registrar", "", "Registrar to use (spaceship, porkbun, namecheap, valuedomain)")
	cmd.Flags().StringVar(&providerFlag, "provider", "", "Hosting provider (vercel, netlify, cloudflare-pages, github-pages, flyio)")

	return cmd
}

// checkPrices gathers pricing from all configured registrars (reuses domains check logic).
func checkPrices(domain, tld string) []priceEntry {
	var prices []priceEntry
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Porkbun
	wg.Add(1)
	go func() {
		defer wg.Done()
		if porkbunClient != nil {
			// API configured: use real availability check only (no fallback to static)
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
			}
			// API error → skip entirely; don't assume Available=true
			return
		}
		// No client: static pricing for comparison only (CanRegAPI=false, Available=false)
		allPricing, err := porkbun.FetchPricingStatic()
		if err == nil {
			if p, ok := allPricing[tld]; ok {
				mu.Lock()
				prices = append(prices, priceEntry{
					Registrar: "Porkbun",
					RegPrice:  p.RegPrice,
					RenPrice:  p.RenPrice,
					Available: false, // unknown — no API to verify
					CanRegAPI: false,
				})
				mu.Unlock()
			}
		}
	}()

	// Spaceship
	if spaceshipClient != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			avail, err := spaceshipClient.CheckAvailability(domain)
			if err == nil {
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
					Available: false, // unknown — no API to verify
					CanRegAPI: false,
				})
				mu.Unlock()
			}
		}()
	}

	// Cloudflare
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
				Available: false, // unknown — no API to verify
				CanRegAPI: false,
			})
			mu.Unlock()
		}
	}()

	// Value Domain
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

	// Namecheap
	if namecheapClient != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			avail, err := namecheapClient.CheckAvailability(domain)
			if err != nil {
				return
			}
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

	// Netim (ccTLD specialist: .jp, .fr, .it, .es, .au, etc.)
	if netimClient != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			avail, err := netimClient.CheckAvailability(domain)
			if err != nil {
				return
			}
			mu.Lock()
			prices = append(prices, priceEntry{
				Registrar: "Netim",
				RegPrice:  avail.RegPrice,
				RenPrice:  avail.RenPrice,
				Available: avail.Available,
				CanRegAPI: true,
			})
			mu.Unlock()
		}()
	} else {
		// Show static Netim prices for ccTLDs even without credentials
		wg.Add(1)
		go func() {
			defer wg.Done()
			if p := netim.GetStaticPrice(tld); p > 0 {
				mu.Lock()
				prices = append(prices, priceEntry{
					Registrar: "Netim",
					RegPrice:  p,
					RenPrice:  p,
					Available: false, // unknown — no API to verify
					CanRegAPI: false,
				})
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	sort.Slice(prices, func(i, j int) bool {
		return prices[i].RegPrice < prices[j].RegPrice
	})

	return prices
}

// registerDomain registers via the specified registrar.
func registerDomain(domain, registrar string) error {
	switch registrar {
	case "Spaceship":
		return spaceshipClient.RegisterDomain(domain)
	case "Porkbun":
		return porkbunClient.RegisterDomain(domain)
	case "Namecheap":
		return namecheapClient.RegisterDomain(domain)
	case "Value Domain":
		return client.RegisterDomain(domain, 1)
	case "Netim":
		return netimClient.RegisterDomain(domain)
	default:
		return fmt.Errorf("unsupported registrar: %s", registrar)
	}
}

// promptLaunchProviderSelection shows hosting providers with a "Skip" option.
func promptLaunchProviderSelection(reader *bufio.Reader) *hosting.Provider {
	fmt.Println("    Which hosting provider?")
	fmt.Println()

	for i, p := range hosting.Providers {
		fmt.Printf("      %d) %s\n", i+1, p.Name)
	}
	skipN := len(hosting.Providers) + 1
	fmt.Printf("      %d) Skip (DNS only)\n", skipN)

	fmt.Println()
	fmt.Printf("    Choose [1-%d]: ", skipN)

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	var choice int
	if _, err := fmt.Sscanf(input, "%d", &choice); err != nil || choice < 1 || choice > skipN {
		color.Red("    Invalid choice: %s", input)
		return nil
	}

	if choice == skipN {
		return nil
	}

	return &hosting.Providers[choice-1]
}

// runMenuLaunch runs the launch flow from the interactive menu.
func runMenuLaunch(reader *bufio.Reader) {
	fmt.Println()
	fmt.Print("  Enter domain to launch (e.g. example.com): ")
	domain, _ := reader.ReadString('\n')
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return
	}

	parts := strings.SplitN(domain, ".", 2)
	if len(parts) < 2 {
		color.Red("  Invalid domain: %s", domain)
		return
	}
	tld := parts[1]

	bold := color.New(color.Bold)

	fmt.Println()
	bold.Printf("  Launch %s\n", domain)

	// Step 1: Check
	fmt.Println()
	bold.Println("  Step 1: Check availability")
	fmt.Printf("    Checking %s...\n", domain)

	prices := checkPrices(domain, tld)
	if len(prices) == 0 {
		color.Red("    No pricing data available.")
		return
	}

	var available []priceEntry
	for _, p := range prices {
		if p.Available {
			available = append(available, p)
		}
	}
	if len(available) == 0 {
		color.Red("    %s is not available at any registrar.", domain)
		return
	}

	// Pick cheapest API-registrable
	var chosen priceEntry
	for _, p := range available {
		if p.CanRegAPI {
			chosen = p
			break
		}
	}
	if chosen.Registrar == "" {
		color.Yellow("    No API-registrable registrar available for %s.", domain)
		return
	}

	color.Green("    Available on %s at $%.2f/yr", chosen.Registrar, chosen.RegPrice)

	// Step 2: Register
	fmt.Println()
	bold.Println("  Step 2: Register domain")

	// Billing guard
	regPriceCents := int64(chosen.RegPrice * 100)
	hold, holdErr := billingGuard(billing.OpDomainRegister, regPriceCents, domain)
	if holdErr != nil {
		return
	}
	if hold != nil {
		fmt.Printf("    Estimated cost: $%.2f (incl. %d%% markup)\n",
			float64(hold.CostCents)/100, billing.MarkupPercent)
		fmt.Printf("    Balance after: $%.2f\n", float64(hold.BalanceAfter)/100)
	}

	fmt.Printf("    Register %s on %s? [Y/n]: ", domain, chosen.Registrar)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "" && answer != "y" && answer != "yes" {
		fmt.Println("    Cancelled.")
		if hold != nil {
			hold.Release()
		}
		return
	}

	fmt.Printf("    Registering...")
	if err := registerDomain(domain, chosen.Registrar); err != nil {
		color.Red(" Failed: %v", err)
		if hold != nil {
			hold.Release()
		}
		return
	}
	color.Green(" Domain registered!")

	if hold != nil {
		hold.Confirm()
	}

	// Step 3: Hosting
	fmt.Println()
	bold.Println("  Step 3: Connect to hosting")
	hp := promptLaunchProviderSelection(reader)

	var dnsAdded, dnsTotal int
	if hp != nil {
		var answers []string
		for _, prompt := range hp.Prompts {
			fmt.Printf("    %s: ", prompt)
			ans, _ := reader.ReadString('\n')
			answers = append(answers, strings.TrimSpace(ans))
		}

		records := hp.Records(domain, answers)
		dnsTotal = len(records)

		fmt.Println()
		fmt.Printf("    Setting up DNS for %s...\n", hp.Name)
		for _, rec := range records {
			label := fmt.Sprintf("%-5s %-4s -> %s", rec.Type, rec.Name, rec.Content)
			err := addDNSRecord(domain, rec)
			if err != nil {
				color.Red("      Adding %s  x %v", label, err)
			} else {
				color.Green("      Adding %s  done", label)
				dnsAdded++
			}
		}
	}

	// Summary
	fmt.Println()
	bold.Println("  Done!")
	fmt.Println()
	bold.Println("  Summary:")
	fmt.Printf("    Domain:   %s (registered on %s)\n", domain, chosen.Registrar)
	if hp != nil {
		fmt.Printf("    Hosting:  %s (%d/%d DNS records added)\n", hp.Name, dnsAdded, dnsTotal)
		steps := hp.NextSteps(domain)
		if len(steps) > 0 {
			fmt.Println()
			bold.Println("  Next steps:")
			for i, step := range steps {
				fmt.Printf("    %d. %s\n", i+1, step)
			}
		}
	} else {
		fmt.Println("    Hosting:  skipped")
		fmt.Println()
		bold.Println("  Next steps:")
		fmt.Printf("    1. Connect to hosting: regctl sites create %s\n", domain)
		fmt.Printf("    2. Verify DNS: regctl dns list %s\n", domain)
	}
	printBillingInfo(hold)
	fmt.Println()
}

