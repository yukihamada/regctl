package valuedomain

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupTestServer(handler http.HandlerFunc) (*httptest.Server, *Client) {
	ts := httptest.NewServer(handler)
	client := NewClient("test-api-key")
	client.BaseURL = ts.URL
	return ts, client
}

func TestListDomains(t *testing.T) {
	ts, client := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		resp := DomainsResponse{
			Domains: []Domain{
				{ID: 1, DomainName: "example.com", Status: "active", ExpiresAt: "2025-12-31"},
				{ID: 2, DomainName: "test.jp", Status: "active", ExpiresAt: "2025-06-30"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer ts.Close()

	domains, err := client.ListDomains()
	if err != nil {
		t.Fatalf("ListDomains() error: %v", err)
	}

	if len(domains) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(domains))
	}
	if domains[0].DomainName != "example.com" {
		t.Errorf("expected example.com, got %s", domains[0].DomainName)
	}
}

func TestCheckAvailability(t *testing.T) {
	ts, client := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		resp := DomainSearchResponse{
			Domains: map[string]DomainAvailability{
				"newdomain.com": {Available: true, Price: 1280, Currency: "JPY"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer ts.Close()

	avail, err := client.CheckAvailability("newdomain.com")
	if err != nil {
		t.Fatalf("CheckAvailability() error: %v", err)
	}
	if !avail.Available {
		t.Error("expected domain to be available")
	}
	if avail.Price != 1280 {
		t.Errorf("expected price 1280, got %f", avail.Price)
	}
}

func TestCheckAvailabilityUnavailable(t *testing.T) {
	ts, client := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		resp := DomainSearchResponse{
			Domains: map[string]DomainAvailability{
				"taken.com": {Available: false},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer ts.Close()

	avail, err := client.CheckAvailability("taken.com")
	if err != nil {
		t.Fatalf("CheckAvailability() error: %v", err)
	}
	if avail.Available {
		t.Error("expected domain to be unavailable")
	}
}

func TestGetDNSRecords(t *testing.T) {
	ts, client := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		resp := DNSResponse{
			Records: []DNSRecord{
				{ID: 1, Type: "A", Name: "@", Content: "1.2.3.4", TTL: 3600},
				{ID: 2, Type: "MX", Name: "@", Content: "mail.example.com", TTL: 3600, Priority: 10},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer ts.Close()

	records, err := client.GetDNSRecords("example.com")
	if err != nil {
		t.Fatalf("GetDNSRecords() error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].Type != "A" {
		t.Errorf("expected A record, got %s", records[0].Type)
	}
}

func TestRegisterDomain(t *testing.T) {
	ts, client := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var req RegisterRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.SLD != "newsite" || req.TLD != "com" {
			t.Errorf("unexpected SLD/TLD: %s.%s", req.SLD, req.TLD)
		}
		if req.Years != 1 {
			t.Errorf("expected 1 year, got %d", req.Years)
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	defer ts.Close()

	err := client.RegisterDomain("newsite.com", 1)
	if err != nil {
		t.Fatalf("RegisterDomain() error: %v", err)
	}
}

func TestAPIError(t *testing.T) {
	ts, client := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "unauthorized"}`))
	})
	defer ts.Close()

	_, err := client.ListDomains()
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestDeleteDNSRecord(t *testing.T) {
	callCount := 0
	ts, client := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Method == "GET" {
			// Return existing records
			resp := DNSResponse{
				Records: []DNSRecord{
					{ID: 1, Type: "A", Name: "@", Content: "1.2.3.4", TTL: 3600},
					{ID: 2, Type: "MX", Name: "@", Content: "mail.example.com", TTL: 3600},
				},
			}
			json.NewEncoder(w).Encode(resp)
		} else if r.Method == "PUT" {
			// Verify only record 2 remains
			var req DNSUpdateRequest
			json.NewDecoder(r.Body).Decode(&req)
			if len(req.Records) != 1 {
				t.Errorf("expected 1 record after delete, got %d", len(req.Records))
			}
			if req.Records[0].ID != 2 {
				t.Errorf("expected record ID 2, got %d", req.Records[0].ID)
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}
	})
	defer ts.Close()

	err := client.DeleteDNSRecord("example.com", 1)
	if err != nil {
		t.Fatalf("DeleteDNSRecord() error: %v", err)
	}
}
