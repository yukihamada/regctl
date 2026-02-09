package valuedomain

import (
	"fmt"
)

// GetDNSRecords returns all DNS records for a domain.
func (c *Client) GetDNSRecords(domain string) ([]DNSRecord, error) {
	var resp DNSResponse
	if err := c.doRequest("GET", fmt.Sprintf("/domains/%s/dns", domain), nil, &resp); err != nil {
		return nil, fmt.Errorf("get DNS records: %w", err)
	}
	return resp.Records, nil
}

// UpdateDNSRecords replaces all DNS records for a domain.
func (c *Client) UpdateDNSRecords(domain string, records []DNSRecord) error {
	req := DNSUpdateRequest{Records: records}
	if err := c.doRequest("PUT", fmt.Sprintf("/domains/%s/dns", domain), req, nil); err != nil {
		return fmt.Errorf("update DNS records: %w", err)
	}
	return nil
}

// AddDNSRecord adds a single DNS record by appending to existing records.
func (c *Client) AddDNSRecord(domain string, record DNSRecord) error {
	existing, err := c.GetDNSRecords(domain)
	if err != nil {
		return fmt.Errorf("get existing records: %w", err)
	}

	records := append(existing, record)
	return c.UpdateDNSRecords(domain, records)
}

// DeleteDNSRecord removes a DNS record by ID.
func (c *Client) DeleteDNSRecord(domain string, recordID int) error {
	existing, err := c.GetDNSRecords(domain)
	if err != nil {
		return fmt.Errorf("get existing records: %w", err)
	}

	var filtered []DNSRecord
	found := false
	for _, r := range existing {
		if r.ID == recordID {
			found = true
			continue
		}
		filtered = append(filtered, r)
	}

	if !found {
		return fmt.Errorf("DNS record %d not found", recordID)
	}

	return c.UpdateDNSRecords(domain, filtered)
}
