package cloudflare

import (
	"fmt"

	"github.com/yukihamada/regctl/internal/provider"
)

type cfZone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type cfDNSRecord struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority"`
}

type cfError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type cfDomain struct {
	Name            string   `json:"name"`
	LastKnownStatus string   `json:"last_known_status"`
	ExpiresAt       string   `json:"expires_at"`
	AutoRenew       bool     `json:"auto_renew"`
	NameServers     []string `json:"name_servers"`
}

// getZoneID finds the zone ID for a domain.
func (c *Client) getZoneID(domain string) (string, error) {
	url := fmt.Sprintf("%s/zones?name=%s", baseURL, domain)
	var resp struct {
		Success bool     `json:"success"`
		Result  []cfZone `json:"result"`
	}
	if err := c.doRequest("GET", url, nil, &resp); err != nil {
		return "", err
	}
	if !resp.Success || len(resp.Result) == 0 {
		return "", fmt.Errorf("zone not found for %s", domain)
	}
	return resp.Result[0].ID, nil
}

// ListRecords lists DNS records for a domain via Cloudflare.
func (c *Client) ListRecords(domain string) ([]provider.DNSRecord, error) {
	zoneID, err := c.getZoneID(domain)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/zones/%s/dns_records?per_page=100", baseURL, zoneID)
	var resp struct {
		Success bool          `json:"success"`
		Result  []cfDNSRecord `json:"result"`
	}
	if err := c.doRequest("GET", url, nil, &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("cloudflare: failed to list DNS records")
	}

	var records []provider.DNSRecord
	for _, r := range resp.Result {
		records = append(records, provider.DNSRecord{
			ID:       r.ID,
			Type:     r.Type,
			Name:     r.Name,
			Content:  r.Content,
			TTL:      r.TTL,
			Priority: r.Priority,
		})
	}
	return records, nil
}

// AddRecord adds a DNS record via Cloudflare.
func (c *Client) AddRecord(domain string, rec provider.DNSRecord) error {
	zoneID, err := c.getZoneID(domain)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/zones/%s/dns_records", baseURL, zoneID)
	body := map[string]interface{}{
		"type":    rec.Type,
		"name":    rec.Name,
		"content": rec.Content,
		"ttl":     rec.TTL,
	}
	if rec.Priority > 0 {
		body["priority"] = rec.Priority
	}

	var resp struct {
		Success bool      `json:"success"`
		Errors  []cfError `json:"errors"`
	}
	if err := c.doRequest("POST", url, body, &resp); err != nil {
		return err
	}
	if !resp.Success {
		msg := "unknown error"
		if len(resp.Errors) > 0 {
			msg = resp.Errors[0].Message
		}
		return fmt.Errorf("cloudflare: %s", msg)
	}
	return nil
}

// DeleteRecord deletes a DNS record via Cloudflare.
func (c *Client) DeleteRecord(domain, recordID string) error {
	zoneID, err := c.getZoneID(domain)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", baseURL, zoneID, recordID)
	var resp struct {
		Success bool `json:"success"`
	}
	if err := c.doRequest("DELETE", url, nil, &resp); err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("cloudflare: failed to delete record")
	}
	return nil
}

// ListDomains lists domains registered on Cloudflare Registrar.
func (c *Client) ListDomains() ([]provider.Domain, error) {
	if c.AccountID == "" {
		return nil, fmt.Errorf("cloudflare: account_id required for registrar operations")
	}

	url := fmt.Sprintf("%s/accounts/%s/registrar/domains", baseURL, c.AccountID)
	var resp struct {
		Success bool       `json:"success"`
		Result  []cfDomain `json:"result"`
	}
	if err := c.doRequest("GET", url, nil, &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("cloudflare: failed to list registrar domains")
	}

	var domains []provider.Domain
	for _, d := range resp.Result {
		domains = append(domains, provider.Domain{
			Name:      d.Name,
			Registrar: "Cloudflare",
			Status:    d.LastKnownStatus,
			ExpiresAt: d.ExpiresAt,
			AutoRenew: d.AutoRenew,
		})
	}
	return domains, nil
}

// CheckAvailability is not supported for new registrations via Cloudflare API.
func (c *Client) CheckAvailability(domain string) (*provider.DomainAvailability, error) {
	return nil, fmt.Errorf("cloudflare: API registration not supported (use dashboard)")
}

// RegisterDomain is not supported via Cloudflare API.
func (c *Client) RegisterDomain(domain string) error {
	return fmt.Errorf("cloudflare: domain registration via API is not supported")
}

// CloudflarePricing is static pricing data (at-cost, no markup).
var CloudflarePricing = map[string][2]float64{
	"sh": {45.00, 45.00}, "dev": {10.18, 10.18}, "io": {45.00, 45.00},
	"app": {12.18, 12.18}, "cc": {8.00, 8.00}, "co": {26.00, 26.00},
	"com": {10.44, 10.44}, "net": {11.84, 11.84}, "org": {7.50, 10.11},
	"xyz": {11.18, 11.18}, "me": {15.79, 15.79}, "run": {21.20, 21.20},
	"cloud": {20.18, 20.18}, "tools": {28.20, 28.20}, "ai": {70.00, 70.00},
	"codes": {56.18, 56.18}, "software": {32.18, 32.18}, "gg": {51.80, 51.80},
}

// GetStaticPrice returns Cloudflare's known price for a TLD.
func GetStaticPrice(tld string) (reg, renew float64, ok bool) {
	p, found := CloudflarePricing[tld]
	if !found {
		return 0, 0, false
	}
	return p[0], p[1], true
}
