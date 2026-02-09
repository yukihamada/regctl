package spaceship

import (
	"fmt"

	"github.com/yukihamada/regctl/internal/provider"
)

type dnsItem struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Address  string `json:"address,omitempty"`  // A, AAAA
	Cname    string `json:"cname,omitempty"`    // CNAME
	Exchange string `json:"exchange,omitempty"` // MX
	Value    string `json:"value,omitempty"`    // TXT, SRV
	TTL      int    `json:"ttl"`
}

type dnsListResponse struct {
	Items []dnsItem `json:"items"`
	Total int       `json:"total"`
}

// ListRecords lists DNS records for a domain on Spaceship.
func (c *Client) ListRecords(domain string) ([]provider.DNSRecord, error) {
	url := fmt.Sprintf("%s/dns/records/%s?take=100&skip=0", baseURL, domain)
	var resp dnsListResponse
	if err := c.doRequest("GET", url, nil, &resp); err != nil {
		return nil, err
	}

	var records []provider.DNSRecord
	for i, r := range resp.Items {
		content := r.Address
		if content == "" {
			content = r.Cname
		}
		if content == "" {
			content = r.Exchange
		}
		if content == "" {
			content = r.Value
		}
		records = append(records, provider.DNSRecord{
			ID:      fmt.Sprintf("%d", i),
			Type:    r.Type,
			Name:    r.Name,
			Content: content,
			TTL:     r.TTL,
		})
	}
	return records, nil
}

// AddRecord adds a DNS record on Spaceship.
func (c *Client) AddRecord(domain string, rec provider.DNSRecord) error {
	url := fmt.Sprintf("%s/dns/records/%s", baseURL, domain)
	item := buildDNSItem(rec)
	body := map[string]interface{}{
		"force": false,
		"items": []interface{}{item},
	}
	return c.doRequest("PUT", url, body, nil)
}

// DeleteRecord deletes a DNS record on Spaceship.
// Spaceship uses record content matching, not ID-based deletion.
func (c *Client) DeleteRecord(domain, recordID string) error {
	// First list records to find the one to delete
	records, err := c.ListRecords(domain)
	if err != nil {
		return err
	}

	for _, r := range records {
		if r.ID == recordID {
			url := fmt.Sprintf("%s/dns/records/%s", baseURL, domain)
			item := buildDNSItem(provider.DNSRecord{
				Type:    r.Type,
				Name:    r.Name,
				Content: r.Content,
			})
			// DELETE uses bare array body
			return c.doRequest("DELETE", url, []interface{}{item}, nil)
		}
	}
	return fmt.Errorf("spaceship: record %s not found", recordID)
}

func buildDNSItem(rec provider.DNSRecord) map[string]interface{} {
	item := map[string]interface{}{
		"type": rec.Type,
		"name": rec.Name,
	}
	if rec.TTL > 0 {
		item["ttl"] = rec.TTL
	}
	switch rec.Type {
	case "A", "AAAA":
		item["address"] = rec.Content
	case "CNAME":
		item["cname"] = rec.Content
	case "MX":
		item["exchange"] = rec.Content
	default:
		item["value"] = rec.Content
	}
	return item
}
