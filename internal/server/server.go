package server

import (
	"fmt"
	"net/http"
	"os"

	"github.com/yukihamada/regctl/internal/billing"
	"github.com/yukihamada/regctl/internal/email"
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
	GitHubClientID     string
	GitHubClientSecret string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURI  string
	BaseURL            string // e.g. https://regctl-api.fly.dev
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
	githubClientID     string
	githubClientSecret string
	googleClientID     string
	googleClientSecret string
	googleRedirectURI  string
	baseURL            string
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
		githubClientID:     cfg.GitHubClientID,
		githubClientSecret: cfg.GitHubClientSecret,
		googleClientID:     cfg.GoogleClientID,
		googleClientSecret: cfg.GoogleClientSecret,
		googleRedirectURI:  cfg.GoogleRedirectURI,
		baseURL:            cfg.BaseURL,
	}
	s.routes()
	return s
}

// ListenAndServe starts the HTTP server on the given port.
func (s *Server) ListenAndServe(port int) error {
	return http.ListenAndServe(fmt.Sprintf(":%d", port), s)
}

// ServeHTTP implements the http.Handler interface.
// Adds security headers to all responses.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	s.mux.ServeHTTP(w, r)
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
	s.mux.HandleFunc("PUT /v1/domains/{domain}/nameservers", s.auth(s.handleUpdateNameservers))
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

	ghDeviceHandler := s.cors(s.handleGitHubDevice)
	s.mux.HandleFunc("POST /v1/auth/github/device", ghDeviceHandler)
	s.mux.HandleFunc("OPTIONS /v1/auth/github/device", ghDeviceHandler)

	ghPollHandler := s.cors(s.handleGitHubPoll)
	s.mux.HandleFunc("POST /v1/auth/github/poll", ghPollHandler)
	s.mux.HandleFunc("OPTIONS /v1/auth/github/poll", ghPollHandler)

	googleStartHandler := s.cors(s.handleGoogleStart)
	s.mux.HandleFunc("POST /v1/auth/google/start", googleStartHandler)
	s.mux.HandleFunc("OPTIONS /v1/auth/google/start", googleStartHandler)

	s.mux.HandleFunc("GET /v1/auth/google/callback", s.handleGoogleCallback)

	// GitHub web OAuth (redirect flow for web clients)
	s.mux.HandleFunc("GET /v1/auth/github/web", s.handleGitHubWebRedirect)
	s.mux.HandleFunc("GET /v1/auth/github/callback", s.handleGitHubCallback)

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

	// Static file serving (when staticDir is configured)
	if s.staticDir != "" {
		if _, err := os.Stat(s.staticDir); err == nil {
			s.mux.HandleFunc("GET /install.sh", s.serveStatic("install.sh", "text/x-shellscript"))
			s.mux.HandleFunc("GET /llms.txt", s.serveStatic("llms.txt", "text/plain; charset=utf-8"))
			s.mux.HandleFunc("GET /prices.json", s.serveStatic("prices.json", "application/json"))
			s.mux.HandleFunc("GET /og-image.svg", s.serveStatic("og-image.svg", "image/svg+xml"))
			s.mux.HandleFunc("GET /{$}", s.serveStatic("index.html", "text/html; charset=utf-8"))
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
