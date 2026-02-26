package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/yukihamada/regctl/internal/billing"
	"github.com/yukihamada/regctl/internal/provider"
	"github.com/yukihamada/regctl/internal/provider/valuedomain"
)

type apiResponse struct {
	OK        bool        `json:"ok"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp string      `json:"timestamp"`
	Hint      string      `json:"hint,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(apiResponse{
		OK:        true,
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func writeError(w http.ResponseWriter, status int, message string, hint string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(apiResponse{
		OK:        false,
		Error:     message,
		Hint:      hint,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// chargeBilling deducts the cost from the customer's balance if billing auth is active.
// Returns true if the handler should continue, false if it returned an error.
func (s *Server) chargeBilling(w http.ResponseWriter, r *http.Request, op billing.OperationType, baseCostCents int64) bool {
	customerID := getCustomerID(r)
	if customerID == "" || !s.billingEnabled {
		return true
	}
	costCents := billing.CalculateCostCents(op, baseCostCents)
	if costCents == 0 {
		return true
	}
	if err := billing.DeductBalance(customerID, costCents, fmt.Sprintf("API: %s", op)); err != nil {
		// Notify via LINE when balance is insufficient
		if strings.Contains(err.Error(), "insufficient") {
			s.lineNotify.Send(fmt.Sprintf(
				"\n[regctl] 残高不足\nCustomer: %s\n操作: %s\n必要額: $%.2f\nチャージ: https://regctl.sh/#billing",
				customerID, op, float64(costCents)/100,
			))
		}
		writeError(w, http.StatusPaymentRequired, err.Error(), "Top up: POST /v1/billing/topup")
		return false
	}
	return true
}

// refundBilling attempts to refund a charge on failure (best-effort).
func (s *Server) refundBilling(r *http.Request, op billing.OperationType, baseCostCents int64) {
	customerID := getCustomerID(r)
	if customerID == "" || !s.billingEnabled {
		return
	}
	costCents := billing.CalculateCostCents(op, baseCostCents)
	if costCents == 0 {
		return
	}
	if err := billing.AddBalance(customerID, costCents, fmt.Sprintf("Refund: %s", op)); err != nil {
		log.Printf("WARN: failed to refund %d cents for %s to %s: %v", costCents, op, customerID, err)
	}
}

// hasProvider returns true if either legacy client or multi-registrar providers are available.
func (s *Server) hasProvider() bool {
	return s.client != nil || len(s.registrars) > 0
}

// checkAvailMulti checks domain availability across all configured registrars.
// Returns the cheapest available option. Available results always beat unavailable ones.
// Results with price=0 are treated as "unsupported TLD" and deprioritized.
func (s *Server) checkAvailMulti(domain string) (*provider.DomainAvailability, error) {
	if len(s.registrars) == 0 {
		return nil, fmt.Errorf("no registrar configured")
	}
	var results []*provider.DomainAvailability
	var lastErr error
	for _, reg := range s.registrars {
		avail, err := reg.CheckAvailability(domain)
		if err != nil {
			log.Printf("DEBUG: %s check %s: %v", reg.Name(), domain, err)
			lastErr = err
			continue
		}
		log.Printf("DEBUG: %s check %s: available=%v price=%.2f", reg.Name(), domain, avail.Available, avail.RegPrice)
		results = append(results, avail)
	}
	if len(results) == 0 {
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, fmt.Errorf("no registrar could check this domain")
	}

	// Pick best result: available with price > available without > unavailable with price > unavailable without
	var best *provider.DomainAvailability
	for _, avail := range results {
		if best == nil {
			best = avail
			continue
		}
		// Available always beats unavailable
		if avail.Available && !best.Available {
			best = avail
			continue
		}
		// Don't replace available with unavailable
		if !avail.Available && best.Available {
			continue
		}
		// Among same availability status, prefer one with valid price
		if avail.RegPrice > 0 && best.RegPrice <= 0 {
			best = avail
			continue
		}
		// Among available with valid prices, prefer cheapest
		if avail.Available && best.Available && avail.RegPrice > 0 && best.RegPrice > 0 && avail.RegPrice < best.RegPrice {
			best = avail
		}
	}
	return best, nil
}

// getRegistrar returns the registrar with the given name, or nil.
func (s *Server) getRegistrar(name string) provider.Registrar {
	for _, reg := range s.registrars {
		if strings.EqualFold(reg.Name(), name) {
			return reg
		}
	}
	return nil
}

// registerDomainMulti registers a domain using the cheapest registrar or a specified one.
func (s *Server) registerDomainMulti(domain, preferredReg string) (string, error) {
	if preferredReg != "" {
		reg := s.getRegistrar(preferredReg)
		if reg == nil {
			return "", fmt.Errorf("registrar %q not configured", preferredReg)
		}
		if err := reg.RegisterDomain(domain); err != nil {
			return "", err
		}
		return reg.Name(), nil
	}
	// Try cheapest first (skip registrars with price 0 = unsupported TLD)
	var cheapest provider.Registrar
	var cheapestPrice float64
	for _, reg := range s.registrars {
		avail, err := reg.CheckAvailability(domain)
		if err != nil || !avail.Available || avail.RegPrice <= 0 {
			continue
		}
		if cheapest == nil || avail.RegPrice < cheapestPrice {
			cheapest = reg
			cheapestPrice = avail.RegPrice
		}
	}
	if cheapest == nil {
		return "", fmt.Errorf("domain not available or no registrar supports this TLD")
	}
	if err := cheapest.RegisterDomain(domain); err != nil {
		return "", err
	}
	return cheapest.Name(), nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	endpoints := []map[string]string{
		{"method": "GET", "path": "/v1/domains", "description": "List all domains"},
		{"method": "GET", "path": "/v1/domains/{domain}", "description": "Get domain details"},
		{"method": "GET", "path": "/v1/domains/check/{domain}", "description": "Check availability (no auth, free)"},
		{"method": "POST", "path": "/v1/domains/check", "description": "Bulk availability check (no auth, max 10)"},
		{"method": "POST", "path": "/v1/domains", "description": "Register a domain"},
		{"method": "POST", "path": "/v1/domains/{domain}/renew", "description": "Renew a domain"},
		{"method": "PUT", "path": "/v1/domains/{domain}/nameservers", "description": "Update nameservers"},
		{"method": "GET", "path": "/v1/dns/{domain}", "description": "List DNS records"},
		{"method": "POST", "path": "/v1/dns/{domain}", "description": "Add a DNS record"},
		{"method": "PUT", "path": "/v1/dns/{domain}/{id}", "description": "Update a DNS record"},
		{"method": "DELETE", "path": "/v1/dns/{domain}/{id}", "description": "Delete a DNS record"},
		{"method": "GET", "path": "/v1/rdap/{domain}", "description": "RDAP/WHOIS lookup (no auth)"},
		{"method": "GET", "path": "/v1/discovery", "description": "Discovery feed of available domains (no auth)"},
	}

	if s.billingEnabled {
		endpoints = append(endpoints,
			map[string]string{"method": "POST", "path": "/v1/billing/signup", "description": "Create account"},
			map[string]string{"method": "POST", "path": "/v1/billing/topup", "description": "Add credit"},
			map[string]string{"method": "GET", "path": "/v1/billing/balance", "description": "Check balance"},
			map[string]string{"method": "POST", "path": "/v1/billing/hold", "description": "Create a balance hold"},
			map[string]string{"method": "POST", "path": "/v1/billing/confirm", "description": "Confirm a hold"},
			map[string]string{"method": "POST", "path": "/v1/billing/release", "description": "Release a hold (refund)"},
		)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "ok",
		"service":   "regctl-api",
		"billing":   s.billingEnabled,
		"endpoints": endpoints,
	})
}

func (s *Server) handleListDomains(w http.ResponseWriter, r *http.Request) {
	if !s.hasProvider() {
		writeError(w, http.StatusServiceUnavailable, "domain provider not configured", "")
		return
	}
	if !s.chargeBilling(w, r, billing.OpDomainList, 0) {
		return
	}

	// Aggregate domains from all registrars
	var allDomains []provider.Domain
	if len(s.registrars) > 0 {
		for _, reg := range s.registrars {
			domains, err := reg.ListDomains()
			if err != nil {
				log.Printf("WARN: list domains from %s: %v", reg.Name(), err)
				continue
			}
			allDomains = append(allDomains, domains...)
		}
	}
	// Also include legacy client
	if s.client != nil {
		domains, err := s.client.ListDomains()
		if err != nil {
			log.Printf("WARN: list domains from ValueDomain: %v", err)
		} else {
			for _, d := range domains {
				allDomains = append(allDomains, provider.Domain{
					Name:      d.Name,
					Registrar: "ValueDomain",
					Status:    d.Status,
					ExpiresAt: d.ExpiresAt,
				})
			}
		}
	}
	if allDomains == nil {
		allDomains = []provider.Domain{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"domains": allDomains,
		"count":   len(allDomains),
	})
}

func (s *Server) handleGetDomain(w http.ResponseWriter, r *http.Request) {
	if s.client == nil {
		writeError(w, http.StatusServiceUnavailable, "domain provider not configured", "")
		return
	}
	if !s.chargeBilling(w, r, billing.OpDomainInfo, 0) {
		return
	}
	domain := r.PathValue("domain")
	info, err := s.client.GetDomainInfo(domain)
	if err != nil {
		s.refundBilling(r, billing.OpDomainInfo, 0)
		writeError(w, http.StatusInternalServerError, err.Error(),
			"Make sure the domain is in your account. GET /v1/domains to list all.")
		return
	}
	writeJSON(w, http.StatusOK, info.Domain)
}

func (s *Server) handleCheckDomain(w http.ResponseWriter, r *http.Request) {
	if !s.hasProvider() {
		writeError(w, http.StatusServiceUnavailable, "domain provider not configured", "")
		return
	}

	domain := r.PathValue("domain")

	// Identify the searcher: authenticated customer ID or hashed IP
	searcherID := getCustomerID(r)
	isAuthenticated := searcherID != ""
	if !isAuthenticated {
		searcherID = hashIP(getClientIP(r))
	}

	// Rate limits (require store)
	if s.store != nil {
		dailyCount, err := s.store.GetDailyCheckCount(searcherID)
		if err != nil {
			log.Printf("WARN: get daily check count: %v", err)
		} else if isAuthenticated && s.billingEnabled && dailyCount >= 1000 {
			// Authenticated: charge $0.01/check over 1000/day
			if !s.chargeBilling(w, r, billing.OpDomainCheckPaid, 0) {
				return
			}
		} else if !isAuthenticated && dailyCount >= 100 {
			// Anonymous: hard limit 100/day per IP
			writeError(w, http.StatusTooManyRequests,
				"rate limit exceeded (100 checks/day)",
				"Sign up for an API key for higher limits: POST /v1/billing/signup")
			return
		}
	}

	// Try multi-registrar first, fall back to legacy client
	var available bool
	var premium bool
	var price float64
	var currency string
	var registrar string

	if len(s.registrars) > 0 {
		avail, err := s.checkAvailMulti(domain)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error(), "")
			return
		}
		available = avail.Available
		premium = avail.Premium
		price = avail.RegPrice
		currency = avail.Currency
		registrar = avail.Registrar
	} else {
		avail, err := s.client.CheckAvailability(domain)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error(), "")
			return
		}
		available = avail.Available
		premium = avail.Premium
		price = avail.Price
		currency = avail.Currency
	}

	// Log the search (best-effort)
	if s.store != nil {
		if _, err := s.store.LogSearch(domain, searcherID, available); err != nil {
			log.Printf("WARN: log search: %v", err)
		}
	}

	result := map[string]interface{}{
		"domain":    domain,
		"available": available,
		"premium":   premium,
		"price":     price,
		"currency":  currency,
	}
	if registrar != "" {
		result["registrar"] = registrar
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleRegisterDomain(w http.ResponseWriter, r *http.Request) {
	if !s.hasProvider() {
		writeError(w, http.StatusServiceUnavailable, "domain provider not configured", "")
		return
	}
	var req struct {
		Domain   string `json:"domain"`
		Years    int    `json:"years"`
		Provider string `json:"provider"` // optional: "Spaceship", "Porkbun"
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 65536)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body",
			`Expected JSON: {"domain": "example.com", "years": 1}`)
		return
	}
	if req.Domain == "" {
		writeError(w, http.StatusBadRequest, "domain is required",
			`Include "domain" field in request body`)
		return
	}
	if req.Years == 0 {
		req.Years = 1
	}

	// For billing users, check price first then charge with markup
	customerID := getCustomerID(r)
	var baseCostCents int64
	if customerID != "" && s.billingEnabled {
		var price float64
		if len(s.registrars) > 0 {
			avail, err := s.checkAvailMulti(req.Domain)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error(),
					"Check availability first: GET /v1/domains/check/"+req.Domain)
				return
			}
			price = avail.RegPrice
		} else if s.client != nil {
			avail, err := s.client.CheckAvailability(req.Domain)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error(),
					"Check availability first: GET /v1/domains/check/"+req.Domain)
				return
			}
			price = avail.Price
		}
		baseCostCents = int64(price * 100) // USD to cents
		if !s.chargeBilling(w, r, billing.OpDomainRegister, baseCostCents) {
			return
		}
	}

	// Register using multi-registrar or legacy client
	var usedRegistrar string
	if len(s.registrars) > 0 {
		reg, err := s.registerDomainMulti(req.Domain, req.Provider)
		if err != nil {
			if customerID != "" && s.billingEnabled {
				s.refundBilling(r, billing.OpDomainRegister, baseCostCents)
			}
			writeError(w, http.StatusInternalServerError, err.Error(),
				"Check availability first: GET /v1/domains/check/"+req.Domain)
			return
		}
		usedRegistrar = reg
	} else {
		if err := s.client.RegisterDomain(req.Domain, req.Years); err != nil {
			if customerID != "" && s.billingEnabled {
				s.refundBilling(r, billing.OpDomainRegister, baseCostCents)
			}
			writeError(w, http.StatusInternalServerError, err.Error(),
				"Check availability first: GET /v1/domains/check/"+req.Domain)
			return
		}
		usedRegistrar = "ValueDomain"
	}

	// Referral credit: reward the first searcher if different from registrant
	if s.store != nil && s.billingEnabled && customerID != "" {
		s.tryReferralCredit(req.Domain, customerID, baseCostCents)
	}

	// Auto-setup: DNS + TLS cert + site record (non-blocking)
	go s.autoSetupSite(req.Domain, usedRegistrar, customerID)

	result := map[string]string{
		"domain":   req.Domain,
		"status":   "registered",
		"site_url": "https://" + req.Domain,
	}
	if usedRegistrar != "" {
		result["registrar"] = usedRegistrar
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) handleListDNS(w http.ResponseWriter, r *http.Request) {
	if s.client == nil {
		writeError(w, http.StatusServiceUnavailable, "domain provider not configured", "")
		return
	}
	if !s.chargeBilling(w, r, billing.OpDNSList, 0) {
		return
	}
	domain := r.PathValue("domain")
	records, err := s.client.GetDNSRecords(domain)
	if err != nil {
		s.refundBilling(r, billing.OpDNSList, 0)
		writeError(w, http.StatusInternalServerError, err.Error(),
			"Make sure you own this domain. GET /v1/domains to list all.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"domain":  domain,
		"records": records,
		"count":   len(records),
	})
}

func (s *Server) handleAddDNS(w http.ResponseWriter, r *http.Request) {
	if s.client == nil {
		writeError(w, http.StatusServiceUnavailable, "domain provider not configured", "")
		return
	}
	if !s.chargeBilling(w, r, billing.OpDNSAdd, 0) {
		return
	}
	domain := r.PathValue("domain")
	var record valuedomain.DNSRecord
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 65536)).Decode(&record); err != nil {
		s.refundBilling(r, billing.OpDNSAdd, 0)
		writeError(w, http.StatusBadRequest, "invalid request body",
			`Expected JSON: {"type": "A", "name": "@", "content": "1.2.3.4", "ttl": 3600}`)
		return
	}
	if record.Type == "" || record.Content == "" {
		s.refundBilling(r, billing.OpDNSAdd, 0)
		writeError(w, http.StatusBadRequest, "type and content are required",
			`Include "type" and "content" fields. Example types: A, AAAA, CNAME, MX, TXT`)
		return
	}

	if err := s.client.AddDNSRecord(domain, record); err != nil {
		s.refundBilling(r, billing.OpDNSAdd, 0)
		writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "created"})
}

func (s *Server) handleDeleteDNS(w http.ResponseWriter, r *http.Request) {
	if s.client == nil {
		writeError(w, http.StatusServiceUnavailable, "domain provider not configured", "")
		return
	}
	domain := r.PathValue("domain")
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid record ID: %s", idStr),
			"Get record IDs from: GET /v1/dns/"+domain)
		return
	}

	if !s.chargeBilling(w, r, billing.OpDNSDelete, 0) {
		return
	}

	if err := s.client.DeleteDNSRecord(domain, id); err != nil {
		s.refundBilling(r, billing.OpDNSDelete, 0)
		writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleRenewDomain(w http.ResponseWriter, r *http.Request) {
	if s.client == nil {
		writeError(w, http.StatusServiceUnavailable, "domain provider not configured", "")
		return
	}
	domain := r.PathValue("domain")
	var req struct {
		Years int `json:"years"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 65536)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body",
			`Expected JSON: {"years": 1}`)
		return
	}
	if req.Years <= 0 {
		req.Years = 1
	}
	if !s.chargeBilling(w, r, billing.OpDomainRenew, 0) {
		return
	}
	if err := s.client.RenewDomain(domain, req.Years); err != nil {
		s.refundBilling(r, billing.OpDomainRenew, 0)
		writeError(w, http.StatusInternalServerError, err.Error(),
			"Make sure the domain is in your account. GET /v1/domains to list all.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"domain": domain,
		"status": "renewed",
	})
}

func (s *Server) handleUpdateNameservers(w http.ResponseWriter, r *http.Request) {
	if s.client == nil {
		writeError(w, http.StatusServiceUnavailable, "domain provider not configured", "")
		return
	}
	domain := r.PathValue("domain")
	var req struct {
		Nameservers []string `json:"nameservers"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 65536)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body",
			`Expected JSON: {"nameservers": ["ns1.example.com", "ns2.example.com"]}`)
		return
	}
	if len(req.Nameservers) < 1 || len(req.Nameservers) > 6 {
		writeError(w, http.StatusBadRequest, "1-6 nameservers required",
			"Provide between 1 and 6 nameserver hostnames")
		return
	}
	if !s.chargeBilling(w, r, billing.OpNSUpdate, 0) {
		return
	}
	if err := s.client.UpdateNameservers(domain, req.Nameservers); err != nil {
		s.refundBilling(r, billing.OpNSUpdate, 0)
		writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"domain": domain,
		"status": "nameservers_updated",
	})
}

func (s *Server) handleUpdateDNSRecord(w http.ResponseWriter, r *http.Request) {
	if s.client == nil {
		writeError(w, http.StatusServiceUnavailable, "domain provider not configured", "")
		return
	}
	domain := r.PathValue("domain")
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid record ID: %s", idStr),
			"Get record IDs from: GET /v1/dns/"+domain)
		return
	}

	var record valuedomain.DNSRecord
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 65536)).Decode(&record); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body",
			`Expected JSON: {"type":"A","name":"@","content":"1.2.3.4","ttl":3600}`)
		return
	}
	if record.Type == "" || record.Content == "" {
		writeError(w, http.StatusBadRequest, "type and content are required",
			`Include "type" and "content" fields. Example types: A, AAAA, CNAME, MX, TXT`)
		return
	}

	if !s.chargeBilling(w, r, billing.OpDNSUpdate, 0) {
		return
	}

	// GET-modify-PUT: fetch all records, replace matching ID, update all
	records, err := s.client.GetDNSRecords(domain)
	if err != nil {
		s.refundBilling(r, billing.OpDNSUpdate, 0)
		writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}

	found := false
	for i, rec := range records {
		if rec.ID == id {
			record.ID = id
			records[i] = record
			found = true
			break
		}
	}
	if !found {
		s.refundBilling(r, billing.OpDNSUpdate, 0)
		writeError(w, http.StatusNotFound, fmt.Sprintf("record ID %d not found", id),
			"Get record IDs from: GET /v1/dns/"+domain)
		return
	}

	if err := s.client.UpdateDNSRecords(domain, records); err != nil {
		s.refundBilling(r, billing.OpDNSUpdate, 0)
		writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleBulkCheck(w http.ResponseWriter, r *http.Request) {
	if !s.hasProvider() {
		writeError(w, http.StatusServiceUnavailable, "domain provider not configured", "")
		return
	}
	var req struct {
		Domains []string `json:"domains"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 65536)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body",
			`Expected JSON: {"domains": ["example.com", "example.io"]}`)
		return
	}
	if len(req.Domains) == 0 || len(req.Domains) > 10 {
		writeError(w, http.StatusBadRequest, "1-10 domains required",
			"Provide between 1 and 10 domain names")
		return
	}

	type result struct {
		Domain    string  `json:"domain"`
		Available bool    `json:"available"`
		Premium   bool    `json:"premium,omitempty"`
		Price     float64 `json:"price,omitempty"`
		Currency  string  `json:"currency,omitempty"`
		Registrar string  `json:"registrar,omitempty"`
		Error     string  `json:"error,omitempty"`
	}

	results := make([]result, 0, len(req.Domains))
	for _, domain := range req.Domains {
		if len(s.registrars) > 0 {
			avail, err := s.checkAvailMulti(domain)
			if err != nil {
				results = append(results, result{Domain: domain, Error: err.Error()})
				continue
			}
			results = append(results, result{
				Domain:    domain,
				Available: avail.Available,
				Premium:   avail.Premium,
				Price:     avail.RegPrice,
				Currency:  avail.Currency,
				Registrar: avail.Registrar,
			})
		} else {
			avail, err := s.client.CheckAvailability(domain)
			if err != nil {
				results = append(results, result{Domain: domain, Error: err.Error()})
				continue
			}
			results = append(results, result{
				Domain:    domain,
				Available: avail.Available,
				Premium:   avail.Premium,
				Price:     avail.Price,
				Currency:  avail.Currency,
			})
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"results": results,
		"count":   len(results),
	})
}

func (s *Server) handleDiscovery(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"domains": []interface{}{},
			"count":   0,
		})
		return
	}

	limit := 50
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	entries, err := s.store.GetDiscoveryFeed(limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch discovery feed", "")
		log.Printf("ERROR: discovery feed: %v", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"domains": entries,
		"count":   len(entries),
	})
}

// tryReferralCredit rewards the first searcher of a domain if they are
// different from the registrant. The credit is 10% of the registration cost.
// Requires at least 2 distinct authenticated searchers to prevent self-dealing
// (e.g. searching with account A and registering with account B).
func (s *Server) tryReferralCredit(domain, registrantID string, baseCostCents int64) {
	// Anti-abuse: require at least 2 distinct authenticated searchers
	distinctCount, err := s.store.CountDistinctSearchers(domain)
	if err != nil {
		log.Printf("WARN: count distinct searchers for %s: %v", domain, err)
		return
	}
	if distinctCount < 2 {
		return
	}

	firstSearcher, err := s.store.GetFirstSearcher(domain)
	if err != nil {
		log.Printf("WARN: get first searcher for %s: %v", domain, err)
		return
	}
	if firstSearcher == "" || firstSearcher == registrantID {
		return
	}

	totalCost := billing.CalculateCostCents(billing.OpDomainRegister, baseCostCents)
	creditCents := totalCost / 10 // 10% of registration cost
	if creditCents <= 0 {
		return
	}

	inserted, err := s.store.RecordReferralCredit(domain, firstSearcher, registrantID, creditCents)
	if err != nil {
		log.Printf("WARN: record referral credit for %s: %v", domain, err)
		return
	}
	if !inserted {
		return // already credited
	}

	desc := fmt.Sprintf("Referral: %s registered by %s", domain, registrantID)
	if err := billing.AddBalance(firstSearcher, creditCents, desc); err != nil {
		log.Printf("WARN: add referral balance for %s: %v", firstSearcher, err)
	}
}

func (s *Server) handleRDAP(w http.ResponseWriter, r *http.Request) {
	domain := r.PathValue("domain")

	resp, err := http.Get("https://rdap.org/domain/" + domain)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to query RDAP", err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to read RDAP response", "")
		return
	}

	if resp.StatusCode != http.StatusOK {
		writeError(w, resp.StatusCode, "RDAP lookup failed", string(body))
		return
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		writeError(w, http.StatusBadGateway, "invalid RDAP response", "")
		return
	}

	// Extract key fields
	out := map[string]interface{}{
		"domain": domain,
	}

	if v, ok := raw["status"]; ok {
		out["status"] = v
	}
	if v, ok := raw["ldhName"]; ok {
		out["name"] = v
	}

	// Extract registrar from entities
	if entities, ok := raw["entities"].([]interface{}); ok {
		for _, ent := range entities {
			e, ok := ent.(map[string]interface{})
			if !ok {
				continue
			}
			if roles, ok := e["roles"].([]interface{}); ok {
				for _, role := range roles {
					if role == "registrar" {
						if vcards, ok := e["vcardArray"].([]interface{}); ok && len(vcards) > 1 {
							if arr, ok := vcards[1].([]interface{}); ok {
								for _, item := range arr {
									if a, ok := item.([]interface{}); ok && len(a) >= 4 && a[0] == "fn" {
										out["registrar"] = a[3]
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Extract dates from events
	if events, ok := raw["events"].([]interface{}); ok {
		for _, ev := range events {
			e, ok := ev.(map[string]interface{})
			if !ok {
				continue
			}
			action, _ := e["eventAction"].(string)
			date, _ := e["eventDate"].(string)
			switch action {
			case "registration":
				out["created"] = date
			case "expiration":
				out["expires"] = date
			}
		}
	}

	// Extract nameservers
	if ns, ok := raw["nameservers"].([]interface{}); ok {
		var names []string
		for _, n := range ns {
			if nsObj, ok := n.(map[string]interface{}); ok {
				if name, ok := nsObj["ldhName"].(string); ok {
					names = append(names, name)
				}
			}
		}
		if len(names) > 0 {
			out["nameservers"] = names
		}
	}

	writeJSON(w, http.StatusOK, out)
}

// handlePricesText returns a curl-friendly plain text price table.
// GET /v1/prices?q=dojo  → show TLD options for "dojo"
// GET /v1/prices?tld=com → show registrar prices for .com
// GET /v1/prices         → show top 20 cheapest TLDs
func (s *Server) handlePricesText(w http.ResponseWriter, r *http.Request) {
	// Load prices.json
	data, err := os.ReadFile(s.staticDir + "/prices.json")
	if err != nil {
		http.Error(w, "prices data not available", http.StatusInternalServerError)
		return
	}

	var pf struct {
		Updated   string `json:"updated"`
		TotalTLDs int    `json:"total_tlds"`
		TLDs      []struct {
			TLD    string             `json:"tld"`
			Prices map[string]float64 `json:"prices"`
			Best   string             `json:"best_registrar"`
			BestP  float64            `json:"best_price"`
		} `json:"tlds"`
	}
	if err := json.Unmarshal(data, &pf); err != nil {
		http.Error(w, "failed to parse prices", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	query := r.URL.Query().Get("q")
	tldFilter := r.URL.Query().Get("tld")
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 897 {
		limit = n
	}

	if tldFilter != "" {
		// Show registrar comparison for a specific TLD
		tldFilter = strings.TrimPrefix(strings.ToLower(tldFilter), ".")
		for _, t := range pf.TLDs {
			tld := strings.TrimPrefix(t.TLD, ".")
			if tld == tldFilter {
				fmt.Fprintf(w, "regctl.sh — Renewal prices for .%s\n", tld)
				fmt.Fprintf(w, "Updated: %s\n\n", pf.Updated)
				fmt.Fprintf(w, "  %-14s %10s\n", "Registrar", "Price/yr")
				fmt.Fprintf(w, "  %-14s %10s\n", "──────────────", "─────────")
				type rp struct {
					name  string
					price float64
				}
				var rows []rp
				for reg, price := range t.Prices {
					rows = append(rows, rp{reg, price})
				}
				sort.Slice(rows, func(i, j int) bool { return rows[i].price < rows[j].price })
				for i, row := range rows {
					marker := "  "
					if i == 0 {
						marker = "→ "
					}
					fmt.Fprintf(w, "%s%-14s %9s\n", marker, row.name, fmt.Sprintf("$%.2f", row.price))
				}
				fmt.Fprintf(w, "\nBest: %s at $%.2f/yr\n", t.Best, t.BestP)
				return
			}
		}
		fmt.Fprintf(w, "No pricing data for .%s\n", tldFilter)
		return
	}

	if query != "" {
		// Show TLD options for a keyword
		query = strings.ToLower(strings.TrimSpace(query))
		sorted := make([]struct {
			tld   string
			best  string
			price float64
		}, 0, len(pf.TLDs))
		for _, t := range pf.TLDs {
			if t.BestP > 0 {
				tld := strings.TrimPrefix(t.TLD, ".")
				sorted = append(sorted, struct {
					tld   string
					best  string
					price float64
				}{tld, t.Best, t.BestP})
			}
		}
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].price < sorted[j].price })

		fmt.Fprintf(w, "regctl.sh — TLD options for \"%s\"\n", query)
		fmt.Fprintf(w, "Updated: %s | %d TLDs available\n\n", pf.Updated, len(sorted))
		fmt.Fprintf(w, "  %-28s %10s  %-12s\n", "Domain", "Price/yr", "Registrar")
		fmt.Fprintf(w, "  %-28s %10s  %-12s\n", "────────────────────────────", "─────────", "────────────")
		n := limit
		if n > len(sorted) {
			n = len(sorted)
		}
		for i := 0; i < n; i++ {
			s := sorted[i]
			marker := "  "
			if i == 0 {
				marker = "→ "
			}
			domain := query + "." + s.tld
			fmt.Fprintf(w, "%s%-28s %9s  %s\n", marker, domain, fmt.Sprintf("$%.2f", s.price), s.best)
		}
		if len(sorted) > n {
			fmt.Fprintf(w, "\n  ... and %d more TLDs. Use ?limit=%d to see all.\n", len(sorted)-n, len(sorted))
		}
		fmt.Fprintf(w, "\nCheck availability: curl %s/v1/domains/check/%s.com\n", s.baseURL, query)
		fmt.Fprintf(w, "Full data:          curl %s/prices.json\n", s.baseURL)
		return
	}

	// Default: show top N cheapest TLDs
	type entry struct {
		tld   string
		best  string
		price float64
	}
	sorted := make([]entry, 0, len(pf.TLDs))
	for _, t := range pf.TLDs {
		if t.BestP > 0 {
			sorted = append(sorted, entry{strings.TrimPrefix(t.TLD, "."), t.Best, t.BestP})
		}
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].price < sorted[j].price })

	fmt.Fprintf(w, "regctl.sh — Domain Price Comparison (%d TLDs)\n", len(sorted))
	fmt.Fprintf(w, "Updated: %s\n\n", pf.Updated)
	fmt.Fprintf(w, "  %-10s %10s  %-12s\n", "TLD", "Price/yr", "Best Registrar")
	fmt.Fprintf(w, "  %-10s %10s  %-12s\n", "──────────", "─────────", "──────────────")
	n := limit
	if n > len(sorted) {
		n = len(sorted)
	}
	for _, e := range sorted[:n] {
		fmt.Fprintf(w, "  .%-9s %9s  %s\n", e.tld, fmt.Sprintf("$%.2f", e.price), e.best)
	}
	fmt.Fprintf(w, "\nSearch:     curl %s/v1/prices?q=dojo\n", s.baseURL)
	fmt.Fprintf(w, "TLD detail: curl %s/v1/prices?tld=com\n", s.baseURL)
	fmt.Fprintf(w, "Full data:  curl %s/prices.json\n", s.baseURL)
}

// isCurl returns true if the request appears to be from curl or a CLI tool.
func isCurl(r *http.Request) bool {
	ua := r.Header.Get("User-Agent")
	return strings.HasPrefix(ua, "curl/") || strings.HasPrefix(ua, "Wget/") || strings.HasPrefix(ua, "HTTPie/")
}
