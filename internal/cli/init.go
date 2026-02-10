package cli

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yukihamada/regctl/internal/config"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Set up regctl (first-time setup wizard)",
		Long: `Interactive setup wizard for first-time users.

This will guide you through signing in or setting up your registrar API keys.

Sign-in methods:
  1) Email (verification code)
  2) GitHub (device flow)
  3) Google (browser redirect)
  4) Existing API key
  5) Manual registrar setup (Porkbun, Spaceship, Cloudflare, Value Domain)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInitWizard()
		},
	}
}

func runInitWizard() error {
	reader := bufio.NewReader(os.Stdin)

	printWelcomeBanner()

	fmt.Println("  Welcome! Let's get you set up.")
	fmt.Println()
	fmt.Println("  How would you like to sign in?")
	fmt.Println()
	fmt.Println("    1) Email (verification code)")
	fmt.Println("    2) GitHub")
	fmt.Println("    3) Google")
	fmt.Println("    4) I already have an API key")
	fmt.Println("    5) Manual registrar setup (advanced)")
	fmt.Println()

	fmt.Print("  Choose [1-5]: ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	var err error
	switch choice {
	case "1":
		err = initEmailAuth(reader)
	case "2":
		err = initGitHubAuth()
	case "3":
		err = initGoogleAuth()
	case "4":
		err = initExistingKey(reader)
	case "5":
		err = initManualRegistrar(reader)
	default:
		return fmt.Errorf("invalid choice: %s", choice)
	}

	if err != nil {
		return err
	}

	printPostAuthGuide()
	return nil
}

func initEmailAuth(reader *bufio.Reader) error {
	fmt.Println()
	fmt.Print("  Enter your email: ")
	emailAddr, _ := reader.ReadString('\n')
	emailAddr = strings.TrimSpace(emailAddr)
	if emailAddr == "" || !strings.Contains(emailAddr, "@") {
		return fmt.Errorf("invalid email address")
	}

	apiURL := getAPIURL()
	ac := NewAPIClient(apiURL, "")

	fmt.Println("  Sending verification code...")
	if err := ac.RequestEmailAuth(emailAddr); err != nil {
		return fmt.Errorf("send verification code: %w", err)
	}

	color.Green("  Code sent! Check your inbox.")
	fmt.Println()
	fmt.Print("  Enter 6-digit code: ")
	code, _ := reader.ReadString('\n')
	code = strings.TrimSpace(code)

	fmt.Println("  Verifying...")
	apiKey, email, err := ac.VerifyEmailCode(emailAddr, code)
	if err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	return saveAuthResult(apiKey, email)
}

func initGitHubAuth() error {
	apiURL := getAPIURL()
	ac := NewAPIClient(apiURL, "")

	fmt.Println()
	fmt.Println("  Starting GitHub authentication...")

	deviceResp, err := ac.StartGitHubDevice()
	if err != nil {
		return fmt.Errorf("start GitHub auth: %w", err)
	}

	fmt.Println()
	color.New(color.FgCyan, color.Bold).Printf("  Your code: %s\n", deviceResp.UserCode)
	fmt.Println()
	fmt.Printf("  Opening %s in your browser...\n", deviceResp.VerificationURI)
	fmt.Println("  Enter the code above, then authorize regctl.")
	fmt.Println()

	openBrowser(deviceResp.VerificationURI)

	// Poll for completion
	interval := 5
	if deviceResp.Interval > 0 {
		interval = deviceResp.Interval
	}

	fmt.Print("  Waiting for authorization")
	for i := 0; i < 60; i++ {
		time.Sleep(time.Duration(interval) * time.Second)
		fmt.Print(".")

		apiKey, email, err := ac.PollGitHubDevice(deviceResp.DeviceCode)
		if err != nil {
			continue
		}
		if apiKey != "" {
			fmt.Println()
			return saveAuthResult(apiKey, email)
		}
	}

	fmt.Println()
	return fmt.Errorf("GitHub authorization timed out. Please try again")
}

func initGoogleAuth() error {
	apiURL := getAPIURL()
	ac := NewAPIClient(apiURL, "")

	// Start local callback server
	listener, err := net.Listen("tcp", "127.0.0.1:18923")
	if err != nil {
		return fmt.Errorf("start local server: %w", err)
	}

	resultCh := make(chan authCallbackResult, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.URL.Query().Get("key")
		email := r.URL.Query().Get("email")
		resultCh <- authCallbackResult{apiKey: apiKey, email: email}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html><html><body style="font-family:sans-serif;text-align:center;padding:60px;">
<h2 style="color:#10b981;">Authentication successful!</h2>
<p>You can close this window and return to your terminal.</p>
</body></html>`)
	})
	srv := &http.Server{Handler: mux}
	go srv.Serve(listener)
	defer srv.Close()

	localRedirect := "http://localhost:18923/callback"

	fmt.Println()
	fmt.Println("  Starting Google authentication...")

	authResp, err := ac.StartGoogleAuth(localRedirect)
	if err != nil {
		return fmt.Errorf("start Google auth: %w", err)
	}

	fmt.Println("  Opening browser for Google sign-in...")
	openBrowser(authResp.AuthURL)

	fmt.Println("  Waiting for authentication...")
	fmt.Println()

	select {
	case result := <-resultCh:
		if result.apiKey == "" {
			return fmt.Errorf("authentication failed — no API key received")
		}
		return saveAuthResult(result.apiKey, result.email)
	case <-time.After(5 * time.Minute):
		return fmt.Errorf("Google authentication timed out. Please try again")
	}
}

type authCallbackResult struct {
	apiKey string
	email  string
}

func initExistingKey(reader *bufio.Reader) error {
	fmt.Println()
	fmt.Print("  Enter your API key (rk_live_...): ")
	key, _ := reader.ReadString('\n')
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("API key is required")
	}
	return saveAuthResult(key, "")
}

func initManualRegistrar(reader *bufio.Reader) error {
	fmt.Println()
	fmt.Println("  Select registrar:")
	fmt.Println()
	fmt.Println("    1) Porkbun")
	fmt.Println("    2) Spaceship")
	fmt.Println("    3) Cloudflare")
	fmt.Println("    4) Value Domain")
	fmt.Println()
	fmt.Print("  Choose [1-4]: ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	switch choice {
	case "1":
		return setupPorkbun(reader)
	case "2":
		return setupSpaceship(reader)
	case "3":
		return setupCloudflare(reader)
	case "4":
		return setupValueDomain(reader)
	default:
		return fmt.Errorf("invalid choice: %s", choice)
	}
}

func setupPorkbun(reader *bufio.Reader) error {
	fmt.Println()
	fmt.Println("  Get your API keys at: https://porkbun.com/account/api")
	fmt.Println()
	fmt.Print("  API Key: ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)
	fmt.Print("  Secret Key: ")
	secret, _ := reader.ReadString('\n')
	secret = strings.TrimSpace(secret)

	if apiKey == "" || secret == "" {
		return fmt.Errorf("both API key and secret key are required")
	}
	if err := config.Set("porkbun_api_key", apiKey); err != nil {
		return err
	}
	if err := config.Set("porkbun_secret_key", secret); err != nil {
		return err
	}
	color.Green("  Porkbun credentials saved!")
	return nil
}

func setupSpaceship(reader *bufio.Reader) error {
	fmt.Println()
	fmt.Println("  Get your API keys from your Spaceship dashboard.")
	fmt.Println()
	fmt.Print("  API Key: ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)
	fmt.Print("  API Secret: ")
	secret, _ := reader.ReadString('\n')
	secret = strings.TrimSpace(secret)

	if apiKey == "" || secret == "" {
		return fmt.Errorf("both API key and secret are required")
	}
	if err := config.Set("spaceship_api_key", apiKey); err != nil {
		return err
	}
	if err := config.Set("spaceship_api_secret", secret); err != nil {
		return err
	}
	color.Green("  Spaceship credentials saved!")
	return nil
}

func setupCloudflare(reader *bufio.Reader) error {
	fmt.Println()
	fmt.Println("  Get your API token at: https://dash.cloudflare.com/profile/api-tokens")
	fmt.Println()
	fmt.Print("  API Token: ")
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)

	if token == "" {
		return fmt.Errorf("API token is required")
	}
	if err := config.Set("cloudflare_token", token); err != nil {
		return err
	}
	color.Green("  Cloudflare credentials saved!")
	return nil
}

func setupValueDomain(reader *bufio.Reader) error {
	fmt.Println()
	fmt.Println("  Get your API key at: https://www.value-domain.com/api/")
	fmt.Println()
	fmt.Print("  API Key: ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	if err := config.Set("api_key", apiKey); err != nil {
		return err
	}
	color.Green("  Value Domain credentials saved!")
	return nil
}

func saveAuthResult(apiKey, email string) error {
	if err := config.Set("regctl_billing_key", apiKey); err != nil {
		return fmt.Errorf("save API key: %w", err)
	}

	fmt.Println()
	if email != "" {
		color.Green("  Authenticated as %s", email)
	} else {
		color.Green("  API key saved!")
	}
	fmt.Println()

	// Mask key for display
	display := apiKey
	if len(apiKey) > 20 {
		display = apiKey[:12] + "..." + apiKey[len(apiKey)-4:]
	}
	fmt.Printf("  Your API key: %s\n", display)
	fmt.Printf("  Saved to %s\n", config.GetConfigPath())

	return nil
}

func printWelcomeBanner() {
	cyan := color.New(color.FgCyan, color.Bold)
	fmt.Println()
	cyan.Println("    ╭─────────────────────────────────────╮")
	cyan.Println("    │                                     │")
	cyan.Println("    │   ┏━┓┏━┓┏━┓┏━┓╺┳╸╻                │")
	cyan.Println("    │   ┣┳┛┣╸ ┃╺┓┃   ┃ ┃    .sh         │")
	cyan.Println("    │   ╹┗╸┗━╸┗━┛┗━╸ ╹ ┗━╸              │")
	cyan.Println("    │                                     │")
	cyan.Println("    │   Domain Management Made Easy       │")
	cyan.Println("    │                                     │")
	cyan.Println("    ╰─────────────────────────────────────╯")
	fmt.Println()
}

// printUsageGuide shows available commands (used by interactive mode).
func printUsageGuide() {
	bold := color.New(color.Bold)
	bold.Println("  Quick Start:")
	fmt.Println()
	fmt.Println("    List your domains:")
	color.Cyan("      regctl domains list")
	fmt.Println()
	fmt.Println("    Check if a domain is available:")
	color.Cyan("      regctl domains check example.com")
	fmt.Println()
	fmt.Println("    View DNS records:")
	color.Cyan("      regctl dns list example.com")
	fmt.Println()
	fmt.Println("    Add a DNS record:")
	color.Cyan("      regctl dns add example.com -t A -n @ -c 1.2.3.4")
	fmt.Println()
	fmt.Println("    See all commands:")
	color.Cyan("      regctl --help")
	fmt.Println()
}

func printPostAuthGuide() {
	fmt.Println()
	fmt.Println("  ──────────────────────────────────────")
	fmt.Println()
	bold := color.New(color.Bold)
	bold.Println("  You're all set! Try these:")
	fmt.Println()
	fmt.Println("    Check domain availability:")
	color.Cyan("      regctl domains check cool-startup.com")
	fmt.Println()
	fmt.Println("    Register a domain:")
	color.Cyan("      regctl domains register cool-startup.com")
	fmt.Println()
	fmt.Println("    List your domains:")
	color.Cyan("      regctl domains list")
	fmt.Println()
	fmt.Println("    Manage DNS:")
	color.Cyan("      regctl dns list example.com")
	fmt.Println()
}

func getAPIURL() string {
	cfg, err := config.Load()
	if err != nil || cfg.RegctlAPIURL == "" {
		return "https://regctl-api.fly.dev"
	}
	return cfg.RegctlAPIURL
}

