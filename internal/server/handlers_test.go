package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yukihamada/regctl/internal/provider/valuedomain"
)

func setupTestAPI() (*Server, *httptest.Server) {
	mockAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/domains" && r.Method == "GET":
			json.NewEncoder(w).Encode(valuedomain.DomainsResponse{
				Domains: []valuedomain.Domain{
					{ID: 1, DomainName: "example.com", Status: "active"},
				},
			})
		case strings.HasSuffix(r.URL.Path, "/dns") && r.Method == "GET":
			json.NewEncoder(w).Encode(valuedomain.DNSResponse{
				Records: []valuedomain.DNSRecord{
					{ID: 1, Type: "A", Name: "@", Content: "1.2.3.4", TTL: 3600},
				},
			})
		case strings.Contains(r.URL.Path, "domainsearch"):
			json.NewEncoder(w).Encode(valuedomain.DomainSearchResponse{
				Domains: map[string]valuedomain.DomainAvailability{
					"test.com": {Available: true, Price: 1000, Currency: "JPY"},
				},
			})
		default:
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}
	}))

	client := valuedomain.NewClient("test-key")
	client.BaseURL = mockAPI.URL

	srv := New(client, "test-api-key")
	return srv, mockAPI
}

func TestHealthEndpoint(t *testing.T) {
	srv, mockAPI := setupTestAPI()
	defer mockAPI.Close()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp apiResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.OK {
		t.Error("expected ok=true")
	}
}

func TestHealthIncludesEndpoints(t *testing.T) {
	srv, mockAPI := setupTestAPI()
	defer mockAPI.Close()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "endpoints") {
		t.Error("health response should include endpoints list for AI discovery")
	}
}

func TestAuthRequired(t *testing.T) {
	srv, mockAPI := setupTestAPI()
	defer mockAPI.Close()

	req := httptest.NewRequest("GET", "/v1/domains", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	var resp apiResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Hint == "" {
		t.Error("error response should include a hint")
	}
}

func TestAuthInvalidKey(t *testing.T) {
	srv, mockAPI := setupTestAPI()
	defer mockAPI.Close()

	req := httptest.NewRequest("GET", "/v1/domains", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListDomainsEndpoint(t *testing.T) {
	srv, mockAPI := setupTestAPI()
	defer mockAPI.Close()

	req := httptest.NewRequest("GET", "/v1/domains", nil)
	req.Header.Set("Authorization", "Bearer test-api-key")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp apiResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.OK {
		t.Error("expected ok=true")
	}
	if resp.Timestamp == "" {
		t.Error("response should include timestamp")
	}
}

func TestCheckDomainEndpoint(t *testing.T) {
	srv, mockAPI := setupTestAPI()
	defer mockAPI.Close()

	req := httptest.NewRequest("GET", "/v1/domains/check/test.com", nil)
	req.Header.Set("Authorization", "Bearer test-api-key")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestListDNSEndpoint(t *testing.T) {
	srv, mockAPI := setupTestAPI()
	defer mockAPI.Close()

	req := httptest.NewRequest("GET", "/v1/dns/example.com", nil)
	req.Header.Set("Authorization", "Bearer test-api-key")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRegisterDomainEndpoint(t *testing.T) {
	srv, mockAPI := setupTestAPI()
	defer mockAPI.Close()

	body := `{"domain":"newsite.com","years":1}`
	req := httptest.NewRequest("POST", "/v1/domains", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-api-key")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
}

func TestRegisterDomainMissingField(t *testing.T) {
	srv, mockAPI := setupTestAPI()
	defer mockAPI.Close()

	body := `{"years":1}`
	req := httptest.NewRequest("POST", "/v1/domains", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-api-key")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var resp apiResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Hint == "" {
		t.Error("error response should include a hint for AI consumers")
	}
}

func TestHealthNoAuthRequired(t *testing.T) {
	srv, mockAPI := setupTestAPI()
	defer mockAPI.Close()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 without auth, got %d", w.Code)
	}
}

func TestResponseAlwaysHasTimestamp(t *testing.T) {
	srv, mockAPI := setupTestAPI()
	defer mockAPI.Close()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var resp apiResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Timestamp == "" {
		t.Error("all responses should include a timestamp for AI logging")
	}
}
