package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/stripe/stripe-go/v81/checkout/session"

	"github.com/yukihamada/regctl/internal/billing"
)

// holdEntry represents a pending balance hold.
type holdEntry struct {
	CustomerID  string
	AmountCents int64
	Description string
	CreatedAt   time.Time
}

// holdStore manages in-memory holds with TTL-based expiry.
var (
	holds   = make(map[string]*holdEntry)
	holdsMu sync.Mutex
)

func init() {
	// Background goroutine to expire holds after 10 minutes
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		for range ticker.C {
			holdsMu.Lock()
			now := time.Now()
			for id, h := range holds {
				if now.Sub(h.CreatedAt) > 10*time.Minute {
					// Auto-release: refund the held amount
					if err := billing.AddBalance(h.CustomerID, h.AmountCents, fmt.Sprintf("Auto-release expired hold: %s", h.Description)); err != nil {
						log.Printf("WARN: auto-release hold %s: %v", id, err)
					} else {
						log.Printf("INFO: auto-released expired hold %s ($%.2f for %s)", id, float64(h.AmountCents)/100, h.Description)
					}
					delete(holds, id)
				}
			}
			holdsMu.Unlock()
		}
	}()
}

func generateHoldID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return "hold_" + hex.EncodeToString(b)
}

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
	const minTopUp = int64(50) // $0.50 Stripe minimum
	if req.AmountCents < minTopUp {
		writeError(w, http.StatusBadRequest,
			fmt.Sprintf("minimum top-up is $%.2f", float64(minTopUp)/100), "")
		return
	}

	sess, err := s.billingClient.CreateTopUpSession(customerID, req.AmountCents)
	if err != nil {
		log.Printf("WARN: create topup session for %s: %v", customerID, err)
		writeError(w, http.StatusInternalServerError, "failed to create checkout session",
			"Please try again or contact support")
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

	// Handle site_sponsor: record in DB and reactivate site if suspended
	if result.Type == "site_sponsor" && s.store != nil {
		site, err := s.store.GetSite(result.Domain)
		if err == nil && site != nil {
			if err := s.store.AddSponsor(site.ID, result.SponsorEmail, result.SponsorCustomerID,
				result.AmountCents, result.SiteCreditCents, result.SponsorTokenCents, ""); err != nil {
				log.Printf("webhook: record sponsor: %v", err)
			}
			// Reactivate suspended site
			if site.Status == "suspended" {
				if err := s.store.UpdateSiteStatus(result.Domain, "active"); err != nil {
					log.Printf("webhook: reactivate site: %v", err)
				}
				if s.flyClient != nil && site.MachineID != "" {
					env := map[string]string{
						"SITE_DOMAIN":    result.Domain,
						"SITE_SUSPENDED": "false",
						"REGCTL_API_URL": s.baseURL,
					}
					if err := s.flyClient.UpdateMachine(site.MachineID, env); err != nil {
						log.Printf("webhook: restart machine: %v", err)
					}
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "processed",
		"type":   result.Type,
	})
}

// handleHold creates a balance hold (deducts immediately, stores hold for potential release).
func (s *Server) handleHold(w http.ResponseWriter, r *http.Request) {
	customerID := getCustomerID(r)
	if customerID == "" {
		writeError(w, http.StatusForbidden, "hold requires a billing API key", "")
		return
	}

	var req struct {
		AmountCents int64  `json:"amount_cents"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body",
			`Expected JSON: {"amount_cents": 1090, "description": "domain_register: example.com"}`)
		return
	}
	if req.AmountCents <= 0 {
		writeError(w, http.StatusBadRequest, "amount_cents must be positive", "")
		return
	}

	// Deduct balance immediately
	if err := billing.DeductBalance(customerID, req.AmountCents, fmt.Sprintf("Hold: %s", req.Description)); err != nil {
		writeError(w, http.StatusPaymentRequired, err.Error(), "Top up: regctl billing topup 10")
		return
	}

	// Get balance after deduction
	balanceAfter, err := billing.GetBalance(customerID)
	if err != nil {
		log.Printf("WARN: get balance after hold: %v", err)
		balanceAfter = 0
	}

	// Store hold
	holdID := generateHoldID()
	holdsMu.Lock()
	holds[holdID] = &holdEntry{
		CustomerID:  customerID,
		AmountCents: req.AmountCents,
		Description: req.Description,
		CreatedAt:   time.Now(),
	}
	holdsMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"hold_id":       holdID,
		"amount_cents":  req.AmountCents,
		"balance_after": balanceAfter,
	})
}

// handleConfirm confirms a hold (no-op since hold already deducted).
func (s *Server) handleConfirm(w http.ResponseWriter, r *http.Request) {
	customerID := getCustomerID(r)
	if customerID == "" {
		writeError(w, http.StatusForbidden, "confirm requires a billing API key", "")
		return
	}

	var req struct {
		HoldID string `json:"hold_id"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body",
			`Expected JSON: {"hold_id": "hold_..."}`)
		return
	}

	holdsMu.Lock()
	h, ok := holds[req.HoldID]
	if !ok {
		holdsMu.Unlock()
		writeError(w, http.StatusNotFound, "hold not found or already processed", "")
		return
	}
	if h.CustomerID != customerID {
		holdsMu.Unlock()
		writeError(w, http.StatusForbidden, "hold belongs to a different customer", "")
		return
	}
	delete(holds, req.HoldID)
	holdsMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]string{"status": "confirmed"})
}

// handleRelease releases a hold (refunds the held amount).
func (s *Server) handleRelease(w http.ResponseWriter, r *http.Request) {
	customerID := getCustomerID(r)
	if customerID == "" {
		writeError(w, http.StatusForbidden, "release requires a billing API key", "")
		return
	}

	var req struct {
		HoldID string `json:"hold_id"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body",
			`Expected JSON: {"hold_id": "hold_..."}`)
		return
	}

	holdsMu.Lock()
	h, ok := holds[req.HoldID]
	if !ok {
		holdsMu.Unlock()
		writeError(w, http.StatusNotFound, "hold not found or already processed", "")
		return
	}
	if h.CustomerID != customerID {
		holdsMu.Unlock()
		writeError(w, http.StatusForbidden, "hold belongs to a different customer", "")
		return
	}
	delete(holds, req.HoldID)
	holdsMu.Unlock()

	// Refund the held amount
	if err := billing.AddBalance(customerID, h.AmountCents, fmt.Sprintf("Release hold: %s", h.Description)); err != nil {
		log.Printf("WARN: release hold %s: %v", req.HoldID, err)
		writeError(w, http.StatusInternalServerError, "failed to release hold", "")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "released"})
}
