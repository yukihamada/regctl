package netim

import (
	"fmt"
	"strings"

	"github.com/yukihamada/regctl/internal/provider"
)

// checkResult is the response from GET /3.0/domain/{domain}/check
type checkResult struct {
	Available bool    `json:"available"`
	Price     float64 `json:"price"`
	Currency  string  `json:"currency"`
}

// domainInfo is the response from GET /3.0/domain/{domain}/
type domainInfo struct {
	Fqdn      string `json:"fqdn"`
	Status    string `json:"status"`
	DateExpir string `json:"date_expir"`
	AutoRenew bool   `json:"auto_renew"`
}

// listDomainsResponse is the response from GET /3.0/domain/
type listDomainsResponse struct {
	Domains []domainInfo `json:"domains"`
}

// CheckAvailability checks if a domain is available on Netim.
func (c *Client) CheckAvailability(domain string) (*provider.DomainAvailability, error) {
	var result checkResult
	if err := c.doRequest("GET", fmt.Sprintf("/domain/%s/check", domain), nil, &result); err != nil {
		return nil, err
	}

	// Renewal price is generally same as registration for Netim flat pricing
	return &provider.DomainAvailability{
		Registrar: "Netim",
		Domain:    domain,
		Available: result.Available,
		RegPrice:  result.Price,
		RenPrice:  result.Price,
		Currency:  "EUR",
	}, nil
}

// RegisterDomain registers a domain on Netim.
// For ccTLDs like .jp that require local presence, the options.local flag is set.
func (c *Client) RegisterDomain(domain string) error {
	// Determine if this TLD requires local presence service
	tld := domain
	if i := strings.LastIndex(domain, "."); i >= 0 {
		tld = domain[i+1:]
	}
	needsLocal := requiresLocalPresence(tld)

	type options struct {
		Local bool `json:"local,omitempty"`
	}
	body := map[string]interface{}{
		"duration": 1,
		"options":  options{Local: needsLocal},
	}

	return c.doRequest("POST", fmt.Sprintf("/domain/%s/", domain), body, nil)
}

// ListDomains lists all domains registered with Netim.
func (c *Client) ListDomains() ([]provider.Domain, error) {
	var resp listDomainsResponse
	if err := c.doRequest("GET", "/domain/", nil, &resp); err != nil {
		return nil, err
	}

	var domains []provider.Domain
	for _, d := range resp.Domains {
		status := d.Status
		if status == "" {
			status = "active"
		}
		domains = append(domains, provider.Domain{
			Name:      d.Fqdn,
			Registrar: "Netim",
			Status:    status,
			ExpiresAt: d.DateExpir,
			AutoRenew: d.AutoRenew,
		})
	}
	return domains, nil
}

// GetStaticPrice returns the approximate Netim price (USD) for a TLD, or 0 if not offered.
func GetStaticPrice(tld string) float64 {
	tld = strings.ToLower(strings.TrimPrefix(tld, "."))
	prices := map[string]float64{
		"jp": 34, "fr": 17, "it": 12, "es": 12, "se": 19,
		"no": 35, "au": 24, "kr": 54, "br": 35, "cn": 15,
		"ru": 15, "at": 12, "be": 10, "pl": 10, "pt": 15,
		"hu": 14, "cz": 14, "fi": 15, "dk": 17, "sk": 14,
		"ro": 14, "bg": 17, "si": 14, "lt": 12, "lv": 17,
		"ee": 12, "gr": 17, "hr": 17, "nz": 24, "sg": 35,
		"hk": 35, "tw": 35, "mx": 27, "ar": 35, "cl": 35, "pe": 35,
	}
	return prices[tld]
}

// requiresLocalPresence returns true for ccTLDs that require a local presence trustee.
var localPresenceTLDs = map[string]bool{
	"jp": true, "de": true, "fr": true, "it": true, "es": true,
	"se": true, "no": true, "fi": true, "dk": true, "pt": true,
	"at": true, "be": true, "nl": true, "ch": true, "pl": true,
	"hu": true, "cz": true, "sk": true, "ro": true, "bg": true,
	"kr": true, "tw": true, "cn": true, "hk": true, "sg": true,
	"au": true, "nz": true, "br": true, "mx": true, "ar": true,
}

func requiresLocalPresence(tld string) bool {
	return localPresenceTLDs[strings.ToLower(tld)]
}
