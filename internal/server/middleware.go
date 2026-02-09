package server

import (
	"net/http"
	"strings"
)

// auth is middleware that validates the Authorization: Bearer <key> header.
func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.apiKey == "" {
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

		if parts[1] != s.apiKey {
			writeError(w, http.StatusUnauthorized, "invalid API key", "")
			return
		}

		next(w, r)
	}
}
