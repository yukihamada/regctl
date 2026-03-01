package godaddy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const baseURL = "https://api.godaddy.com"

// Client is an API client for GoDaddy.
type Client struct {
	apiKey    string
	apiSecret string
	http      *http.Client
}

// NewClient creates a new GoDaddy API client.
func NewClient(apiKey, apiSecret string) *Client {
	return &Client{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		http:      &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) authHeader() string {
	return fmt.Sprintf("sso-key %s:%s", c.apiKey, c.apiSecret)
}

func (c *Client) get(path string, out interface{}) error {
	req, err := http.NewRequest("GET", baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("invalid GoDaddy API credentials")
	}
	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Message string `json:"message"`
		}
		json.Unmarshal(body, &apiErr)
		if apiErr.Message != "" {
			return fmt.Errorf("GoDaddy API: %s", apiErr.Message)
		}
		return fmt.Errorf("GoDaddy API error: %d", resp.StatusCode)
	}
	return json.Unmarshal(body, out)
}

func (c *Client) post(path, contentType string, body io.Reader, out interface{}) error {
	req, err := http.NewRequest("POST", baseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("invalid GoDaddy API credentials")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr struct {
			Message string `json:"message"`
		}
		json.Unmarshal(respBody, &apiErr)
		if apiErr.Message != "" {
			return fmt.Errorf("GoDaddy API: %s", apiErr.Message)
		}
		return fmt.Errorf("GoDaddy API error: %d", resp.StatusCode)
	}
	if out != nil && len(respBody) > 0 {
		return json.Unmarshal(respBody, out)
	}
	return nil
}

type gdDomain struct {
	Domain    string `json:"domain"`
	Status    string `json:"status"`
	Expires   string `json:"expires"`
	AutoRenew bool   `json:"renewAuto"`
}

type gdAvailability struct {
	Available bool    `json:"available"`
	Price     float64 `json:"price"` // in micro-units
	Currency  string  `json:"currency"`
}

type gdDNSRecord struct {
	Type string `json:"type"`
	Name string `json:"name"`
	Data string `json:"data"`
	TTL  int    `json:"ttl"`
}

func (c *Client) ListDomains() ([]gdDomain, error) {
	var domains []gdDomain
	if err := c.get("/v1/domains?limit=500&statuses=ACTIVE", &domains); err != nil {
		return nil, err
	}
	return domains, nil
}

func (c *Client) CheckAvailability(domain string) (*gdAvailability, error) {
	var avail gdAvailability
	if err := c.get(fmt.Sprintf("/v1/domains/available?domain=%s&checkType=FAST&forTransfer=false", domain), &avail); err != nil {
		return nil, err
	}
	return &avail, nil
}

func (c *Client) RegisterDomain(domain string) error {
	// GoDaddy requires contact info for registration; this minimal payload uses defaults
	parts := strings.SplitN(domain, ".", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid domain: %s", domain)
	}
	body := fmt.Sprintf(`{
		"domain": %q,
		"period": 1,
		"renewAuto": true,
		"privacy": false
	}`, domain)
	return c.post("/v1/domains/purchase", "application/json", strings.NewReader(body), nil)
}

func (c *Client) ListDNSRecords(domain string) ([]gdDNSRecord, error) {
	var records []gdDNSRecord
	if err := c.get(fmt.Sprintf("/v1/domains/%s/records", domain), &records); err != nil {
		return nil, err
	}
	return records, nil
}
