package server

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/yukihamada/regctl/internal/billing"
)

type contextKey string

const customerIDKey contextKey = "customerID"

// auth is middleware that validates the Authorization: Bearer <key> header.
// Supports both legacy shared key and billing API keys (rk_live_/rk_test_).
func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.apiKey == "" && !s.billingEnabled {
			next(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, "missing Authorization header",
				"Add header: Authorization: Bearer <your-api-key>")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			writeError(w, http.StatusUnauthorized, "invalid Authorization header format",
				"Use format: Authorization: Bearer <your-api-key>")
			return
		}

		token := parts[1]

		// Check if it's a billing API key
		if billing.IsBillingKey(token) {
			if !s.billingEnabled {
				writeError(w, http.StatusUnauthorized, "billing is not enabled on this server", "")
				return
			}
			customerID, err := billing.ValidateAPIKey(token, s.signingSecret)
			if err != nil {
				pfx := token
				if len(pfx) > 16 {
					pfx = pfx[:16] + "..."
				}
				log.Printf("auth: ValidateAPIKey failed (key=%q): %v", pfx, err)
				writeError(w, http.StatusUnauthorized, "invalid API key", "")
				return
			}
			ctx := context.WithValue(r.Context(), customerIDKey, customerID)
			next(w, r.WithContext(ctx))
			return
		}

		// Legacy shared key
		if s.apiKey != "" && token == s.apiKey {
			next(w, r)
			return
		}

		writeError(w, http.StatusUnauthorized, "invalid API key", "")
	}
}

// optionalAuth extracts the customer ID from the Authorization header if
// present, but does not reject unauthenticated requests. The customer ID
// (or empty string) is stored in the request context.
func (s *Server) optionalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				token := parts[1]
				if billing.IsBillingKey(token) && s.billingEnabled {
					if customerID, err := billing.ValidateAPIKey(token, s.signingSecret); err == nil {
						ctx := context.WithValue(r.Context(), customerIDKey, customerID)
						next(w, r.WithContext(ctx))
						return
					}
				}
			}
		}
		next(w, r)
	}
}

// cors wraps a handler with CORS headers for public endpoints.
func (s *Server) cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

// getCustomerID extracts the billing customer ID from the request context.
// Returns empty string if the request was authenticated with legacy key.
func getCustomerID(r *http.Request) string {
	v, _ := r.Context().Value(customerIDKey).(string)
	return v
}

// getClientIP extracts the real client IP from Fly-Client-IP, X-Forwarded-For,
// or the connection's remote address.
func getClientIP(r *http.Request) string {
	// Fly.io sets this header with the true client IP
	if ip := r.Header.Get("Fly-Client-IP"); ip != "" {
		return ip
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// First entry is the original client
		if i := strings.IndexByte(xff, ','); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}

// hashIP returns a truncated SHA-256 hash of the IP for use as a searcher ID.
// We hash to avoid storing raw IPs in the database.
func hashIP(ip string) string {
	h := sha256.Sum256([]byte(ip))
	return fmt.Sprintf("ip_%x", h[:8])
}
