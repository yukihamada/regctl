package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/yukihamada/regctl/internal/billing"
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
	if s.client == nil {
		writeError(w, http.StatusServiceUnavailable, "domain provider not configured", "")
		return
	}
	if !s.chargeBilling(w, r, billing.OpDomainList, 0) {
		return
	}
	domains, err := s.client.ListDomains()
	if err != nil {
		s.refundBilling(r, billing.OpDomainList, 0)
		writeError(w, http.StatusInternalServerError, err.Error(), "Check your VALUEDOMAIN_API_KEY")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"domains": domains,
		"count":   len(domains),
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
	if s.client == nil {
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

	avail, err := s.client.CheckAvailability(domain)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}

	// Log the search (best-effort)
	if s.store != nil {
		if _, err := s.store.LogSearch(domain, searcherID, avail.Available); err != nil {
			log.Printf("WARN: log search: %v", err)
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"domain":    domain,
		"available": avail.Available,
		"premium":   avail.Premium,
		"price":     avail.Price,
		"currency":  avail.Currency,
	})
}

func (s *Server) handleRegisterDomain(w http.ResponseWriter, r *http.Request) {
	if s.client == nil {
		writeError(w, http.StatusServiceUnavailable, "domain provider not configured", "")
		return
	}
	var req struct {
		Domain string `json:"domain"`
		Years  int    `json:"years"`
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
		avail, err := s.client.CheckAvailability(req.Domain)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error(),
				"Check availability first: GET /v1/domains/check/"+req.Domain)
			return
		}
		// Convert price to cents (price is in JPY or the registrar's currency)
		baseCostCents = int64(avail.Price)
		if !s.chargeBilling(w, r, billing.OpDomainRegister, baseCostCents) {
			return
		}
	}

	if err := s.client.RegisterDomain(req.Domain, req.Years); err != nil {
		if customerID != "" && s.billingEnabled {
			s.refundBilling(r, billing.OpDomainRegister, baseCostCents)
		}
		writeError(w, http.StatusInternalServerError, err.Error(),
			"Check availability first: GET /v1/domains/check/"+req.Domain)
		return
	}

	// Referral credit: reward the first searcher if different from registrant
	if s.store != nil && s.billingEnabled && customerID != "" {
		s.tryReferralCredit(req.Domain, customerID, baseCostCents)
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"domain": req.Domain,
		"status": "registered",
	})
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
	if s.client == nil {
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
		Error     string  `json:"error,omitempty"`
	}

	results := make([]result, 0, len(req.Domains))
	for _, domain := range req.Domains {
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
