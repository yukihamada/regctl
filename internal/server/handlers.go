package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

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

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"service": "regctl-api",
		"endpoints": []map[string]string{
			{"method": "GET", "path": "/v1/domains", "description": "List all domains"},
			{"method": "GET", "path": "/v1/domains/{domain}", "description": "Get domain details"},
			{"method": "GET", "path": "/v1/domains/check/{domain}", "description": "Check availability"},
			{"method": "POST", "path": "/v1/domains", "description": "Register a domain"},
			{"method": "GET", "path": "/v1/dns/{domain}", "description": "List DNS records"},
			{"method": "POST", "path": "/v1/dns/{domain}", "description": "Add a DNS record"},
			{"method": "DELETE", "path": "/v1/dns/{domain}/{id}", "description": "Delete a DNS record"},
		},
	})
}

func (s *Server) handleListDomains(w http.ResponseWriter, r *http.Request) {
	domains, err := s.client.ListDomains()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "Check your VALUEDOMAIN_API_KEY")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"domains": domains,
		"count":   len(domains),
	})
}

func (s *Server) handleGetDomain(w http.ResponseWriter, r *http.Request) {
	domain := r.PathValue("domain")
	info, err := s.client.GetDomainInfo(domain)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(),
			"Make sure the domain is in your account. GET /v1/domains to list all.")
		return
	}
	writeJSON(w, http.StatusOK, info.Domain)
}

func (s *Server) handleCheckDomain(w http.ResponseWriter, r *http.Request) {
	domain := r.PathValue("domain")
	avail, err := s.client.CheckAvailability(domain)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
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
	var req struct {
		Domain string `json:"domain"`
		Years  int    `json:"years"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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

	if err := s.client.RegisterDomain(req.Domain, req.Years); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(),
			"Check availability first: GET /v1/domains/check/"+req.Domain)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{
		"domain": req.Domain,
		"status": "registered",
	})
}

func (s *Server) handleListDNS(w http.ResponseWriter, r *http.Request) {
	domain := r.PathValue("domain")
	records, err := s.client.GetDNSRecords(domain)
	if err != nil {
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
	domain := r.PathValue("domain")
	var record valuedomain.DNSRecord
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body",
			`Expected JSON: {"type": "A", "name": "@", "content": "1.2.3.4", "ttl": 3600}`)
		return
	}
	if record.Type == "" || record.Content == "" {
		writeError(w, http.StatusBadRequest, "type and content are required",
			`Include "type" and "content" fields. Example types: A, AAAA, CNAME, MX, TXT`)
		return
	}

	if err := s.client.AddDNSRecord(domain, record); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "created"})
}

func (s *Server) handleDeleteDNS(w http.ResponseWriter, r *http.Request) {
	domain := r.PathValue("domain")
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid record ID: %s", idStr),
			"Get record IDs from: GET /v1/dns/"+domain)
		return
	}

	if err := s.client.DeleteDNSRecord(domain, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
