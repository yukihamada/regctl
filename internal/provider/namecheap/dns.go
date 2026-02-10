package namecheap

import (
	"encoding/xml"
	"fmt"
	"strconv"

	"github.com/yukihamada/regctl/internal/provider"
)

// hostRecord represents a single DNS host record from Namecheap.
type hostRecord struct {
	HostID  string `xml:"HostId,attr"`
	Name    string `xml:"Name,attr"`
	Type    string `xml:"Type,attr"`
	Address string `xml:"Address,attr"`
	MXPref  string `xml:"MXPref,attr"`
	TTL     string `xml:"TTL,attr"`
}

type getHostsResponse struct {
	XMLName xml.Name     `xml:"DomainDNSGetHostsResult"`
	Domain  string       `xml:"Domain,attr"`
	Hosts   []hostRecord `xml:"host"`
}

type setHostsResponse struct {
	XMLName   xml.Name `xml:"DomainDNSSetHostsResult"`
	Domain    string   `xml:"Domain,attr"`
	IsSuccess bool     `xml:"IsSuccess,attr"`
}

// ListRecords lists DNS records for a domain on Namecheap.
func (c *Client) ListRecords(domain string) ([]provider.DNSRecord, error) {
	sld, tld := splitDomain(domain)
	if tld == "" {
		return nil, fmt.Errorf("invalid domain: %s", domain)
	}

	inner, err := c.doRequest("namecheap.domains.dns.getHosts", map[string]string{
		"SLD": sld,
		"TLD": tld,
	})
	if err != nil {
		return nil, err
	}

	var result getHostsResponse
	if err := xml.Unmarshal(inner, &result); err != nil {
		return nil, fmt.Errorf("parse hosts: %w", err)
	}

	var records []provider.DNSRecord
	for _, h := range result.Hosts {
		ttl, _ := strconv.Atoi(h.TTL)
		prio, _ := strconv.Atoi(h.MXPref)
		records = append(records, provider.DNSRecord{
			ID:       h.HostID,
			Type:     h.Type,
			Name:     h.Name,
			Content:  h.Address,
			TTL:      ttl,
			Priority: prio,
		})
	}
	return records, nil
}

// AddRecord adds a DNS record by fetching all existing records, appending the new one,
// and setting them all back (Namecheap replaces all records on setHosts).
func (c *Client) AddRecord(domain string, rec provider.DNSRecord) error {
	sld, tld := splitDomain(domain)
	if tld == "" {
		return fmt.Errorf("invalid domain: %s", domain)
	}

	// Get existing records
	inner, err := c.doRequest("namecheap.domains.dns.getHosts", map[string]string{
		"SLD": sld,
		"TLD": tld,
	})
	if err != nil {
		return fmt.Errorf("get existing records: %w", err)
	}

	var existing getHostsResponse
	if err := xml.Unmarshal(inner, &existing); err != nil {
		return fmt.Errorf("parse existing hosts: %w", err)
	}

	// Build params with all existing records + new one
	params := map[string]string{
		"SLD": sld,
		"TLD": tld,
	}

	i := 1
	for _, h := range existing.Hosts {
		params[fmt.Sprintf("HostName%d", i)] = h.Name
		params[fmt.Sprintf("RecordType%d", i)] = h.Type
		params[fmt.Sprintf("Address%d", i)] = h.Address
		params[fmt.Sprintf("MXPref%d", i)] = h.MXPref
		params[fmt.Sprintf("TTL%d", i)] = h.TTL
		i++
	}

	// Add new record
	params[fmt.Sprintf("HostName%d", i)] = rec.Name
	params[fmt.Sprintf("RecordType%d", i)] = rec.Type
	params[fmt.Sprintf("Address%d", i)] = rec.Content
	params[fmt.Sprintf("MXPref%d", i)] = strconv.Itoa(rec.Priority)
	params[fmt.Sprintf("TTL%d", i)] = strconv.Itoa(rec.TTL)

	inner, err = c.doRequest("namecheap.domains.dns.setHosts", params)
	if err != nil {
		return err
	}

	var result setHostsResponse
	if err := xml.Unmarshal(inner, &result); err != nil {
		return fmt.Errorf("parse set hosts result: %w", err)
	}
	if !result.IsSuccess {
		return fmt.Errorf("namecheap: set hosts failed for %s", domain)
	}
	return nil
}

// DeleteRecord deletes a DNS record by fetching all records, filtering out the target,
// and setting the remaining records back.
func (c *Client) DeleteRecord(domain, recordID string) error {
	sld, tld := splitDomain(domain)
	if tld == "" {
		return fmt.Errorf("invalid domain: %s", domain)
	}

	// Get existing records
	inner, err := c.doRequest("namecheap.domains.dns.getHosts", map[string]string{
		"SLD": sld,
		"TLD": tld,
	})
	if err != nil {
		return fmt.Errorf("get existing records: %w", err)
	}

	var existing getHostsResponse
	if err := xml.Unmarshal(inner, &existing); err != nil {
		return fmt.Errorf("parse existing hosts: %w", err)
	}

	// Filter out the record to delete
	params := map[string]string{
		"SLD": sld,
		"TLD": tld,
	}

	found := false
	i := 1
	for _, h := range existing.Hosts {
		if h.HostID == recordID {
			found = true
			continue
		}
		params[fmt.Sprintf("HostName%d", i)] = h.Name
		params[fmt.Sprintf("RecordType%d", i)] = h.Type
		params[fmt.Sprintf("Address%d", i)] = h.Address
		params[fmt.Sprintf("MXPref%d", i)] = h.MXPref
		params[fmt.Sprintf("TTL%d", i)] = h.TTL
		i++
	}

	if !found {
		return fmt.Errorf("namecheap: record %s not found for %s", recordID, domain)
	}

	inner, err = c.doRequest("namecheap.domains.dns.setHosts", params)
	if err != nil {
		return err
	}

	var result setHostsResponse
	if err := xml.Unmarshal(inner, &result); err != nil {
		return fmt.Errorf("parse set hosts result: %w", err)
	}
	if !result.IsSuccess {
		return fmt.Errorf("namecheap: set hosts failed for %s", domain)
	}
	return nil
}
