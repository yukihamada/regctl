package server

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/stripe/stripe-go/v81/checkout/session"

	"github.com/yukihamada/regctl/internal/billing"
)

func (s *Server) handleSignUp(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email       string `json:"email"`
		AmountCents int64  `json:"amount_cents"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body",
			`Expected JSON: {"email": "user@example.com", "amount_cents": 1000}`)
		return
	}
	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" || !strings.Contains(req.Email, "@") || !strings.Contains(req.Email, ".") {
		writeError(w, http.StatusBadRequest, "valid email is required", "")
		return
	}
	if req.AmountCents == 0 {
		req.AmountCents = int64(billing.MinTopUpCents)
	}
	if req.AmountCents < 0 {
		writeError(w, http.StatusBadRequest, "amount must be positive", "")
		return
	}

	sess, err := s.billingClient.CreateSignUpSession(req.Email, req.AmountCents)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"checkout_url": sess.URL,
		"session_id":   sess.ID,
	})
}

func (s *Server) handleTopUp(w http.ResponseWriter, r *http.Request) {
	customerID := getCustomerID(r)
	if customerID == "" {
		writeError(w, http.StatusForbidden, "top-up requires a billing API key",
			"Sign up at https://regctl.sh/#signup to get a billing key")
		return
	}

	var req struct {
		AmountCents int64 `json:"amount_cents"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body",
			`Expected JSON: {"amount_cents": 1000}`)
		return
	}
	if req.AmountCents <= 0 {
		writeError(w, http.StatusBadRequest, "amount must be positive", "")
		return
	}

	sess, err := s.billingClient.CreateTopUpSession(customerID, req.AmountCents)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"checkout_url": sess.URL,
		"session_id":   sess.ID,
	})
}

func (s *Server) handleBalance(w http.ResponseWriter, r *http.Request) {
	customerID := getCustomerID(r)
	if customerID == "" {
		writeError(w, http.StatusForbidden, "balance requires a billing API key",
			"Sign up at https://regctl.sh/#signup to get a billing key")
		return
	}

	balance, err := billing.GetBalance(customerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to retrieve balance", "")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"balance_cents": balance,
		"balance":       float64(balance) / 100,
		"currency":      "USD",
	})
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("session_id")

	// Validate session ID format (Stripe session IDs start with cs_)
	if !strings.HasPrefix(sessionID, "cs_") || len(sessionID) < 10 {
		writeError(w, http.StatusBadRequest, "invalid session ID", "")
		return
	}

	sess, err := session.Get(sessionID, nil)
	if err != nil {
		writeError(w, http.StatusNotFound, "session not found", "")
		return
	}

	if sess.PaymentStatus != "paid" {
		writeJSON(w, http.StatusOK, map[string]string{
			"status": string(sess.PaymentStatus),
		})
		return
	}

	result := map[string]string{
		"status": "paid",
	}

	// For signup sessions, generate the API key.
	// Only return api_key if this is a signup session with metadata type=signup.
	if sess.Metadata["type"] == "signup" && sess.Customer != nil {
		result["api_key"] = s.billingClient.GenerateAPIKeyForCustomer(sess.Customer.ID)
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	if s.webhookSecret == "" {
		writeError(w, http.StatusServiceUnavailable, "webhook not configured", "")
		return
	}

	payload, err := io.ReadAll(io.LimitReader(r.Body, 65536))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body", "")
		return
	}

	sigHeader := r.Header.Get("Stripe-Signature")
	if sigHeader == "" {
		writeError(w, http.StatusBadRequest, "missing Stripe-Signature header", "")
		return
	}

	result, err := billing.HandleWebhook(payload, sigHeader, s.webhookSecret, s.signingSecret, s.billingClient != nil && s.billingClient.IsLive())
	if err != nil {
		log.Printf("webhook error: %v", err)
		writeError(w, http.StatusBadRequest, "webhook processing failed", "")
		return
	}

	if result == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "processed",
		"type":   result.Type,
	})
}
