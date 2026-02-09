package porkbun

import (
	"fmt"
	"strconv"

	"github.com/yukihamada/regctl/internal/provider"
)

type dnsListResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Records []struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Type     string `json:"type"`
		Content  string `json:"content"`
		TTL      string `json:"ttl"`
		Priority string `json:"prio"`
	} `json:"records"`
}

type dnsCreateRequest struct {
	APIKey    string `json:"apikey"`
	SecretKey string `json:"secretapikey"`
	Name      string `json:"name,omitempty"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	TTL       string `json:"ttl,omitempty"`
	Prio      string `json:"prio,omitempty"`
}

type dnsResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// ListRecords lists DNS records for a domain on Porkbun.
func (c *Client) ListRecords(domain string) ([]provider.DNSRecord, error) {
	var resp dnsListResponse
	if err := c.post("/dns/retrieve/"+domain, c.auth(), &resp); err != nil {
		return nil, err
	}
	if resp.Status != "SUCCESS" {
		return nil, fmt.Errorf("porkbun dns: %s", resp.Message)
	}

	var records []provider.DNSRecord
	for _, r := range resp.Records {
		ttl, _ := strconv.Atoi(r.TTL)
		prio, _ := strconv.Atoi(r.Priority)
		records = append(records, provider.DNSRecord{
			ID:       r.ID,
			Type:     r.Type,
			Name:     r.Name,
			Content:  r.Content,
			TTL:      ttl,
			Priority: prio,
		})
	}
	return records, nil
}

// AddRecord adds a DNS record on Porkbun.
func (c *Client) AddRecord(domain string, rec provider.DNSRecord) error {
	var resp dnsResponse
	if err := c.post("/dns/create/"+domain, dnsCreateRequest{
		APIKey:    c.APIKey,
		SecretKey: c.SecretKey,
		Name:      rec.Name,
		Type:      rec.Type,
		Content:   rec.Content,
		TTL:       strconv.Itoa(rec.TTL),
		Prio:      strconv.Itoa(rec.Priority),
	}, &resp); err != nil {
		return err
	}
	if resp.Status != "SUCCESS" {
		return fmt.Errorf("porkbun dns create: %s", resp.Message)
	}
	return nil
}

// DeleteRecord deletes a DNS record on Porkbun.
func (c *Client) DeleteRecord(domain, recordID string) error {
	var resp dnsResponse
	if err := c.post("/dns/delete/"+domain+"/"+recordID, c.auth(), &resp); err != nil {
		return err
	}
	if resp.Status != "SUCCESS" {
		return fmt.Errorf("porkbun dns delete: %s", resp.Message)
	}
	return nil
}
