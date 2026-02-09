package server

import (
	"fmt"
	"net/http"

	"github.com/yukihamada/regctl/internal/provider/valuedomain"
)

// Server is the HTTP API server.
type Server struct {
	client    *valuedomain.Client
	apiKey    string
	mux       *http.ServeMux
}

// New creates a new HTTP API server.
func New(client *valuedomain.Client, apiKey string) *Server {
	s := &Server{
		client: client,
		apiKey: apiKey,
		mux:    http.NewServeMux(),
	}
	s.routes()
	return s
}

// ListenAndServe starts the HTTP server on the given port.
func (s *Server) ListenAndServe(port int) error {
	return http.ListenAndServe(fmt.Sprintf(":%d", port), s.mux)
}

// ServeHTTP implements the http.Handler interface (for testing).
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	// Health check (no auth)
	s.mux.HandleFunc("GET /health", s.handleHealth)

	// Authenticated API routes
	s.mux.HandleFunc("GET /v1/domains", s.auth(s.handleListDomains))
	s.mux.HandleFunc("GET /v1/domains/check/{domain}", s.auth(s.handleCheckDomain))
	s.mux.HandleFunc("GET /v1/domains/{domain}", s.auth(s.handleGetDomain))
	s.mux.HandleFunc("POST /v1/domains", s.auth(s.handleRegisterDomain))
	s.mux.HandleFunc("GET /v1/dns/{domain}", s.auth(s.handleListDNS))
	s.mux.HandleFunc("POST /v1/dns/{domain}", s.auth(s.handleAddDNS))
	s.mux.HandleFunc("DELETE /v1/dns/{domain}/{id}", s.auth(s.handleDeleteDNS))
}
