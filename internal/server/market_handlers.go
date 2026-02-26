package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/yukihamada/regctl/internal/billing"
	"github.com/yukihamada/regctl/internal/storage"
)

// marketFeeCents calculates the platform fee for a domain sale.
// Fee = max(10% of profit, $1.00 minimum)
func marketFeeCents(askPriceCents, purchasePriceCents int64) int64 {
	profit := askPriceCents - purchasePriceCents
	fee := int64(float64(profit) * 0.10)
	if fee < 100 { // minimum $1.00
		fee = 100
	}
	return fee
}

// GET /v1/market — list all active listings (public, CORS)
func (s *Server) handleMarketList(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		writeJSON(w, http.StatusOK, []storage.MarketListing{})
		return
	}
	listings, err := s.store.GetActiveListings(100, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	// Enrich with fee info (don't expose seller_id to public)
	type PublicListing struct {
		Domain        string  `json:"domain"`
		AskPriceCents int64   `json:"ask_price_cents"`
		AskPriceUSD   float64 `json:"ask_price_usd"`
		FeeCents      int64   `json:"fee_cents"`
		ListedAt      string  `json:"listed_at"`
	}
	out := make([]PublicListing, len(listings))
	for i, m := range listings {
		out[i] = PublicListing{
			Domain:        m.Domain,
			AskPriceCents: m.AskPriceCents,
			AskPriceUSD:   float64(m.AskPriceCents) / 100,
			FeeCents:      marketFeeCents(m.AskPriceCents, m.PurchasePriceCents),
			ListedAt:      m.ListedAt,
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// GET /v1/portfolio — user's owned domains (auth required)
func (s *Server) handlePortfolio(w http.ResponseWriter, r *http.Request) {
	customerID := getCustomerID(r)
	if customerID == "" {
		writeError(w, http.StatusUnauthorized, "authentication required", "POST /v1/auth/email")
		return
	}
	if s.store == nil {
		writeJSON(w, http.StatusOK, []storage.PortfolioEntry{})
		return
	}
	entries, err := s.store.GetPortfolio(customerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	// Attach listing status for each domain
	type PortfolioWithListing struct {
		storage.PortfolioEntry
		Listed        bool    `json:"listed"`
		AskPriceCents int64   `json:"ask_price_cents,omitempty"`
		AskPriceUSD   float64 `json:"ask_price_usd,omitempty"`
	}
	out := make([]PortfolioWithListing, len(entries))
	for i, e := range entries {
		p := PortfolioWithListing{PortfolioEntry: e}
		if listing, _ := s.store.GetListing(e.Domain); listing != nil {
			p.Listed = true
			p.AskPriceCents = listing.AskPriceCents
			p.AskPriceUSD = float64(listing.AskPriceCents) / 100
		}
		out[i] = p
	}
	writeJSON(w, http.StatusOK, out)
}

// POST /v1/market/list — list a domain for sale
// Body: {"domain":"example.com","ask_price":50.00}
func (s *Server) handleMarketListDomain(w http.ResponseWriter, r *http.Request) {
	customerID := getCustomerID(r)
	if customerID == "" {
		writeError(w, http.StatusUnauthorized, "authentication required", "")
		return
	}
	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "storage not available", "")
		return
	}

	var req struct {
		Domain   string  `json:"domain"`
		AskPrice float64 `json:"ask_price"` // USD
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Domain == "" || req.AskPrice <= 0 {
		writeError(w, http.StatusBadRequest, "domain and ask_price required", `{"domain":"example.com","ask_price":50}`)
		return
	}

	// Verify ownership
	entry, err := s.store.GetPortfolioEntry(req.Domain)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	if entry == nil || entry.CustomerID != customerID {
		writeError(w, http.StatusForbidden, "you don't own this domain", "Only domains registered via regctl can be listed")
		return
	}

	askCents := int64(req.AskPrice * 100)
	if askCents < 100 {
		writeError(w, http.StatusBadRequest, "minimum ask price is $1.00", "")
		return
	}

	if err := s.store.ListForSale(req.Domain, customerID, askCents, entry.PurchasePriceCents); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}

	feeCents := marketFeeCents(askCents, entry.PurchasePriceCents)
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"domain":         req.Domain,
		"ask_price_usd":  req.AskPrice,
		"fee_usd":        float64(feeCents) / 100,
		"seller_net_usd": float64(askCents-feeCents) / 100,
		"status":         "listed",
		"message":        fmt.Sprintf("%s is now listed. You'll receive $%.2f after 10%% fee when sold.", req.Domain, float64(askCents-feeCents)/100),
	})
}

// DELETE /v1/market/list/{domain} — cancel a listing
func (s *Server) handleMarketCancelListing(w http.ResponseWriter, r *http.Request) {
	customerID := getCustomerID(r)
	if customerID == "" {
		writeError(w, http.StatusUnauthorized, "authentication required", "")
		return
	}
	domain := r.PathValue("domain")
	if s.store == nil || domain == "" {
		writeError(w, http.StatusBadRequest, "domain required", "")
		return
	}
	if err := s.store.CancelListing(domain, customerID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"domain": domain, "status": "cancelled"})
}

// POST /v1/market/buy/{domain} — buy a listed domain
// Instantly deducts from buyer's balance; pays seller minus fee
func (s *Server) handleMarketBuy(w http.ResponseWriter, r *http.Request) {
	if !s.billingEnabled {
		writeError(w, http.StatusServiceUnavailable, "billing not enabled", "")
		return
	}
	customerID := getCustomerID(r)
	if customerID == "" {
		writeError(w, http.StatusUnauthorized, "authentication required", "POST /v1/auth/email")
		return
	}
	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "storage not available", "")
		return
	}

	domain := r.PathValue("domain")
	listing, err := s.store.GetListing(domain)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	if listing == nil {
		writeError(w, http.StatusNotFound, "listing not found or no longer active", "GET /v1/market")
		return
	}
	if listing.SellerID == customerID {
		writeError(w, http.StatusBadRequest, "you cannot buy your own listing", "")
		return
	}

	// Check buyer's balance
	buyerBalance, err := billing.GetBalance(customerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get balance", "")
		return
	}
	askCents := listing.AskPriceCents
	if buyerBalance < askCents {
		needed := askCents - buyerBalance
		writeError(w, http.StatusPaymentRequired,
			fmt.Sprintf("insufficient balance: need $%.2f more", float64(needed)/100),
			fmt.Sprintf("Top up: POST /v1/billing/topup — amount_cents: %d", needed),
		)
		return
	}

	// Deduct from buyer
	if err := billing.DeductBalance(customerID, askCents, fmt.Sprintf("Market buy: %s", domain)); err != nil {
		writeError(w, http.StatusPaymentRequired, err.Error(), "")
		return
	}

	// Calculate and pay seller (ask - fee)
	feeCents := marketFeeCents(askCents, listing.PurchasePriceCents)
	sellerNetCents := askCents - feeCents
	if err := billing.AddBalance(listing.SellerID, sellerNetCents,
		fmt.Sprintf("Market sale: %s (fee: $%.2f)", domain, float64(feeCents)/100)); err != nil {
		// Refund buyer if seller credit fails
		_ = billing.AddBalance(customerID, askCents, fmt.Sprintf("Refund: market buy %s failed", domain))
		writeError(w, http.StatusInternalServerError, "failed to credit seller", "")
		return
	}

	// Transfer portfolio ownership
	if err := s.store.TransferPortfolio(domain, customerID, askCents); err != nil {
		// Non-fatal — ownership transfer can be retried
		fmt.Printf("WARN: failed to transfer portfolio for %s: %v\n", domain, err)
	}

	// Mark listing as sold
	_ = s.store.MarkSold(domain)

	// Notify seller via LINE if configured
	if s.lineNotify != nil {
		s.lineNotify.Send(fmt.Sprintf(
			"\n[regctl] ドメイン売却完了\nドメイン: %s\n売却額: $%.2f\n手数料: $%.2f\n受取額: $%.2f",
			domain, float64(askCents)/100, float64(feeCents)/100, float64(sellerNetCents)/100,
		))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"domain":          domain,
		"paid_usd":        float64(askCents) / 100,
		"fee_usd":         float64(feeCents) / 100,
		"seller_net_usd":  float64(sellerNetCents) / 100,
		"status":          "purchased",
		"message":         fmt.Sprintf("You now own %s! Transfer instructions have been sent.", domain),
		"transfer_note":   "The seller will initiate domain transfer via their registrar within 48h. Contact support if not received.",
	})
}

// GET /v1/billing/balance-check?amount_cents=NNN — quick balance check for buy flow
func (s *Server) handleBalanceCheck(w http.ResponseWriter, r *http.Request) {
	customerID := getCustomerID(r)
	if customerID == "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"balance_cents": 0,
			"sufficient":    false,
		})
		return
	}
	balance, err := billing.GetBalance(customerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get balance", "")
		return
	}
	needed := int64(0)
	if amountStr := r.URL.Query().Get("amount_cents"); amountStr != "" {
		var amount int64
		if _, err := fmt.Sscanf(amountStr, "%d", &amount); err == nil {
			needed = amount
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"balance_cents": balance,
		"balance_usd":   float64(balance) / 100,
		"sufficient":    balance >= needed,
		"shortfall_cents": func() int64 {
			if balance >= needed {
				return 0
			}
			return needed - balance
		}(),
	})
}
