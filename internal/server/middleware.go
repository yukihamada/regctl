package server

import (
	"context"
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

// cors wraps a handler with CORS headers for public endpoints.
func (s *Server) cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
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
