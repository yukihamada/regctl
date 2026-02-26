package server

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/yukihamada/regctl/internal/billing"
	"github.com/yukihamada/regctl/internal/email"
	"github.com/yukihamada/regctl/internal/flymachines"
	"github.com/yukihamada/regctl/internal/provider"
	"github.com/yukihamada/regctl/internal/provider/valuedomain"
	"github.com/yukihamada/regctl/internal/storage"
)

// Config holds all configuration for the HTTP API server.
type Config struct {
	Client        *valuedomain.Client
	APIKey        string          // legacy shared Bearer token
	BillingClient *billing.Client // nil = billing disabled
	SigningSecret  string
	WebhookSecret  string
	StaticDir     string          // directory with static files (index.html, etc.)
	Store         *storage.Store  // nil = search logging disabled

	// Auth providers
	EmailClient        *email.Client
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURI  string
	BaseURL            string // e.g. https://regctl-api.fly.dev

	// Fly Machines hosting
	FlyAPIToken    string
	FlyAppName     string
	FlyRegion      string
	InternalSecret string

	// Multi-registrar support
	Registrars []provider.Registrar

	// Multi-DNS/NS support
	DNSProviders []provider.DNSProvider
	NSProviders  []provider.NSProvider
}

// Server is the HTTP API server.
type Server struct {
	client         *valuedomain.Client
	apiKey         string
	billingClient  *billing.Client
	signingSecret  string
	webhookSecret  string
	billingEnabled bool
	staticDir      string
	store          *storage.Store
	mux            *http.ServeMux

	// Auth providers
	emailClient        *email.Client
	googleClientID     string
	googleClientSecret string
	googleRedirectURI  string
	baseURL            string

	// Fly Machines hosting
	flyClient      *flymachines.Client
	flyAppName     string
	internalSecret string

	// Multi-registrar
	registrars []provider.Registrar

	// Multi-DNS/NS
	dnsProviders []provider.DNSProvider
	nsProviders  []provider.NSProvider

}

// New creates a new HTTP API server.
func New(cfg Config) *Server {
	s := &Server{
		client:             cfg.Client,
		apiKey:             cfg.APIKey,
		billingClient:      cfg.BillingClient,
		signingSecret:      cfg.SigningSecret,
		webhookSecret:      cfg.WebhookSecret,
		billingEnabled:     cfg.BillingClient != nil,
		staticDir:          cfg.StaticDir,
		store:              cfg.Store,
		mux:                http.NewServeMux(),
		emailClient:        cfg.EmailClient,
		googleClientID:     cfg.GoogleClientID,
		googleClientSecret: cfg.GoogleClientSecret,
		googleRedirectURI:  cfg.GoogleRedirectURI,
		baseURL:            cfg.BaseURL,
		flyAppName:         cfg.FlyAppName,
		internalSecret:     cfg.InternalSecret,
		registrars:         cfg.Registrars,
		dnsProviders:       cfg.DNSProviders,
		nsProviders:        cfg.NSProviders,
	}
	if cfg.FlyAPIToken != "" && cfg.FlyAppName != "" {
		region := cfg.FlyRegion
		if region == "" {
			region = "nrt"
		}
		s.flyClient = flymachines.NewClient(cfg.FlyAPIToken, cfg.FlyAppName, region)
	}
	s.routes()
	s.startDailyBillingWorker()
	return s
}

// ListenAndServe starts the HTTP server on the given port.
func (s *Server) ListenAndServe(port int) error {
	return http.ListenAndServe(fmt.Sprintf(":%d", port), s)
}

// ServeHTTP implements the http.Handler interface.
// Routes hosted domain requests to site handler, everything else to API mux.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")

	// Check if this is a hosted site request (custom domain, not our API domain)
	host := r.Host
	if i := strings.Index(host, ":"); i > 0 {
		host = host[:i]
	}
	if !isAPIDomain(host) && s.serveSite(w, r, host) {
		return
	}

	// SPA slug route: /name → serve index.html so JS can search for name.*
	// Matches single-segment paths that are not known API/static routes.
	if s.staticDir != "" && r.Method == "GET" && isSlugPath(r.URL.Path) {
		b, err := os.ReadFile(s.staticDir + "/index.html")
		if err == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "public, max-age=60")
			w.Write(b)
			return
		}
	}

	s.mux.ServeHTTP(w, r)
}

// isSlugPath returns true for paths like /startup or /myapp:
// single segment, no dot, not a known API/static route.
func isSlugPath(path string) bool {
	if path == "/" || path == "" {
		return false
	}
	// Must be /something with no further slashes
	trimmed := strings.TrimPrefix(path, "/")
	if strings.Contains(trimmed, "/") || strings.Contains(trimmed, ".") {
		return false
	}
	// Exclude known static files and API prefixes
	switch trimmed {
	case "health", "install", "llms", "og-image":
		return false
	}
	return !strings.HasPrefix(trimmed, "v1") && !strings.HasPrefix(trimmed, "webhooks")
}

func (s *Server) routes() {
	// Health check (no auth)
	s.mux.HandleFunc("GET /health", s.handleHealth)

	// Domain availability check (optional auth for rate-limit tracking, CORS enabled)
	checkHandler := s.cors(s.optionalAuth(s.handleCheckDomain))
	s.mux.HandleFunc("GET /v1/domains/check/{domain}", checkHandler)
	s.mux.HandleFunc("OPTIONS /v1/domains/check/{domain}", checkHandler)

	// Bulk domain check (no auth, CORS enabled)
	bulkCheckHandler := s.cors(s.handleBulkCheck)
	s.mux.HandleFunc("POST /v1/domains/check", bulkCheckHandler)
	s.mux.HandleFunc("OPTIONS /v1/domains/check", bulkCheckHandler)

	// RDAP lookup (no auth, CORS enabled)
	rdapHandler := s.cors(s.handleRDAP)
	s.mux.HandleFunc("GET /v1/rdap/{domain}", rdapHandler)
	s.mux.HandleFunc("OPTIONS /v1/rdap/{domain}", rdapHandler)

	// Discovery feed (no auth, CORS enabled)
	discoveryHandler := s.cors(s.handleDiscovery)
	s.mux.HandleFunc("GET /v1/discovery", discoveryHandler)
	s.mux.HandleFunc("OPTIONS /v1/discovery", discoveryHandler)

	// Authenticated API routes
	s.mux.HandleFunc("GET /v1/domains", s.auth(s.handleListDomains))
	s.mux.HandleFunc("GET /v1/domains/{domain}", s.auth(s.handleGetDomain))
	s.mux.HandleFunc("POST /v1/domains", s.auth(s.handleRegisterDomain))
	s.mux.HandleFunc("POST /v1/domains/{domain}/renew", s.auth(s.handleRenewDomain))
	s.mux.HandleFunc("GET /v1/ns/{domain}", s.auth(s.handleGetNameservers))
	s.mux.HandleFunc("PUT /v1/ns/{domain}", s.auth(s.handleUpdateNameservers))
	s.mux.HandleFunc("GET /v1/dns/{domain}", s.auth(s.handleListDNS))
	s.mux.HandleFunc("POST /v1/dns/{domain}", s.auth(s.handleAddDNS))
	s.mux.HandleFunc("PUT /v1/dns/{domain}/{id}", s.auth(s.handleUpdateDNSRecord))
	s.mux.HandleFunc("DELETE /v1/dns/{domain}/{id}", s.auth(s.handleDeleteDNS))

	// Auth routes (no auth required — these create auth)
	authEmailHandler := s.cors(s.handleEmailAuth)
	s.mux.HandleFunc("POST /v1/auth/email", authEmailHandler)
	s.mux.HandleFunc("OPTIONS /v1/auth/email", authEmailHandler)

	authVerifyHandler := s.cors(s.handleVerify)
	s.mux.HandleFunc("POST /v1/auth/verify", authVerifyHandler)
	s.mux.HandleFunc("OPTIONS /v1/auth/verify", authVerifyHandler)

	googleStartHandler := s.cors(s.handleGoogleStart)
	s.mux.HandleFunc("POST /v1/auth/google/start", googleStartHandler)
	s.mux.HandleFunc("OPTIONS /v1/auth/google/start", googleStartHandler)

	s.mux.HandleFunc("GET /v1/auth/google/callback", s.handleGoogleCallback)

	// Billing routes (no auth for signup/webhook, auth for topup/balance/hold)
	if s.billingEnabled {
		s.mux.HandleFunc("POST /v1/billing/signup", s.handleSignUp)
		s.mux.HandleFunc("POST /v1/billing/topup", s.auth(s.handleTopUp))
		s.mux.HandleFunc("GET /v1/billing/balance", s.auth(s.handleBalance))
		s.mux.HandleFunc("POST /v1/billing/hold", s.auth(s.handleHold))
		s.mux.HandleFunc("POST /v1/billing/confirm", s.auth(s.handleConfirm))
		s.mux.HandleFunc("POST /v1/billing/release", s.auth(s.handleRelease))
		s.mux.HandleFunc("GET /v1/billing/session/{session_id}", s.handleGetSession)
		s.mux.HandleFunc("POST /webhooks/stripe", s.handleStripeWebhook)
	}

	// Hosting routes
	s.mux.HandleFunc("POST /v1/sites", s.auth(s.handleCreateSite))
	s.mux.HandleFunc("GET /v1/sites", s.auth(s.handleListSites))
	s.mux.HandleFunc("GET /v1/sites/{domain}", s.auth(s.handleGetSite))
	s.mux.HandleFunc("DELETE /v1/sites/{domain}", s.auth(s.handleDeleteSite))
	s.mux.HandleFunc("POST /v1/sites/{domain}/deploy", s.auth(s.handleDeploySite))
	s.mux.HandleFunc("GET /v1/sites/{domain}/usage", s.auth(s.handleSiteUsage))

	sponsorHandler := s.cors(s.handleSponsorSite)
	s.mux.HandleFunc("POST /v1/sites/{domain}/sponsor", sponsorHandler)
	s.mux.HandleFunc("OPTIONS /v1/sites/{domain}/sponsor", sponsorHandler)

	// AI site generation (CORS enabled, optional auth)
	generateHandler := s.cors(s.optionalAuth(s.handleGenerateSite))
	s.mux.HandleFunc("POST /v1/sites/{domain}/generate", generateHandler)
	s.mux.HandleFunc("OPTIONS /v1/sites/{domain}/generate", generateHandler)

	s.mux.HandleFunc("POST /v1/internal/site-requests", s.handleSiteRequestBatch)

	// Market & Portfolio routes
	marketListHandler := s.cors(s.handleMarketList)
	s.mux.HandleFunc("GET /v1/market", marketListHandler)
	s.mux.HandleFunc("OPTIONS /v1/market", marketListHandler)

	s.mux.HandleFunc("GET /v1/portfolio", s.auth(s.handlePortfolio))
	s.mux.HandleFunc("POST /v1/market/list", s.auth(s.handleMarketListDomain))
	s.mux.HandleFunc("DELETE /v1/market/list/{domain}", s.auth(s.handleMarketCancelListing))

	marketBuyHandler := s.cors(s.auth(s.handleMarketBuy))
	s.mux.HandleFunc("POST /v1/market/buy/{domain}", marketBuyHandler)
	s.mux.HandleFunc("OPTIONS /v1/market/buy/{domain}", marketBuyHandler)

	balanceCheckHandler := s.cors(s.optionalAuth(s.handleBalanceCheck))
	s.mux.HandleFunc("GET /v1/billing/balance-check", balanceCheckHandler)
	s.mux.HandleFunc("OPTIONS /v1/billing/balance-check", balanceCheckHandler)

	// Curl-friendly prices endpoint (no auth, CORS enabled)
	pricesTextHandler := s.cors(s.handlePricesText)
	s.mux.HandleFunc("GET /v1/prices", pricesTextHandler)
	s.mux.HandleFunc("OPTIONS /v1/prices", pricesTextHandler)

	// Static file serving (when staticDir is configured)
	if s.staticDir != "" {
		if _, err := os.Stat(s.staticDir); err == nil {
			s.mux.HandleFunc("GET /install.sh", s.serveStatic("install.sh", "text/x-shellscript"))
			s.mux.HandleFunc("GET /llms.txt", s.serveStatic("llms.txt", "text/plain; charset=utf-8"))
			s.mux.HandleFunc("GET /prices.json", s.serveStatic("prices.json", "application/json"))
			s.mux.HandleFunc("GET /og-image.svg", s.serveStatic("og-image.svg", "image/svg+xml"))
			s.mux.HandleFunc("GET /{$}", s.serveCurlOrHTML())
		}
	}
}

func (s *Server) serveStatic(filename, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "public, max-age=300")
		http.ServeFile(w, r, s.staticDir+"/"+filename)
	}
}

// serveCurlOrHTML returns HTML for browsers, text for curl/wget/httpie.
func (s *Server) serveCurlOrHTML() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isCurl(r) {
			s.handlePricesText(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=300")
		http.ServeFile(w, r, s.staticDir+"/index.html")
	}
}
