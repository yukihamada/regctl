package valuedomain

import (
	"fmt"
	"net/url"
	"strings"
)

// ListDomains returns all domains in the account.
func (c *Client) ListDomains() ([]Domain, error) {
	var resp DomainsResponse
	if err := c.doRequest("GET", "/domains", nil, &resp); err != nil {
		return nil, fmt.Errorf("list domains: %w", err)
	}
	return resp.Domains, nil
}

// GetDomainInfo returns detailed information for a domain.
// It first lists all domains to find the domain ID, then fetches details.
func (c *Client) GetDomainInfo(domain string) (*DomainDetail, error) {
	domains, err := c.ListDomains()
	if err != nil {
		return nil, err
	}

	var domainID int
	found := false
	for _, d := range domains {
		name := d.DomainName
		if name == "" {
			name = d.Name
		}
		if name == domain {
			domainID = d.ID
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("domain %s not found in account", domain)
	}

	var detail DomainDetail
	if err := c.doRequest("GET", fmt.Sprintf("/domains/%d", domainID), nil, &detail); err != nil {
		return nil, fmt.Errorf("get domain info: %w", err)
	}
	return &detail, nil
}

// CheckAvailability checks if a domain is available for registration.
func (c *Client) CheckAvailability(domain string) (*DomainAvailability, error) {
	var resp DomainSearchResponse
	path := "/domainsearch?domainnames=" + url.QueryEscape(domain)
	if err := c.doRequest("GET", path, nil, &resp); err != nil {
		return nil, fmt.Errorf("check availability: %w", err)
	}

	if avail, ok := resp.Domains[domain]; ok {
		return &avail, nil
	}

	// Not in response = TLD not supported or API returned no data
	return nil, fmt.Errorf("valuedomain: no availability data for %s", domain)
}

// RegisterDomain registers a new domain.
func (c *Client) RegisterDomain(domain string, years int) error {
	parts := strings.SplitN(domain, ".", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid domain format: %s", domain)
	}

	req := RegisterRequest{
		Registrar:  "GMO",
		SLD:        parts[0],
		TLD:        parts[1],
		Years:      years,
		WhoisProxy: 1,
		NS:         []string{"ns1.value-domain.com", "ns2.value-domain.com"},
	}

	if err := c.doRequest("POST", "/domains", req, nil); err != nil {
		return fmt.Errorf("register domain: %w", err)
	}
	return nil
}

// UpdateNameservers updates nameservers for a domain.
func (c *Client) UpdateNameservers(domain string, nameservers []string) error {
	req := NameserverUpdateRequest{NS: nameservers}
	if err := c.doRequest("PUT", fmt.Sprintf("/domains/%s/nameserver", domain), req, nil); err != nil {
		return fmt.Errorf("update nameservers: %w", err)
	}
	return nil
}

// RenewDomain renews a domain for the specified number of years.
func (c *Client) RenewDomain(domain string, years int) error {
	req := RenewRequest{Period: years}
	if err := c.doRequest("POST", fmt.Sprintf("/domains/%s/renew", domain), req, nil); err != nil {
		return fmt.Errorf("renew domain: %w", err)
	}
	return nil
}
