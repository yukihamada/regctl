package namecheap

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/yukihamada/regctl/internal/provider"
)

// checkResult represents the DomainCheckResult XML element.
type checkResult struct {
	Domain    string `xml:"Domain,attr"`
	Available bool   `xml:"Available,attr"`
	Premium   bool   `xml:"IsPremiumName,attr"`
}

type domainCheckResponse struct {
	XMLName            xml.Name      `xml:"DomainCheckResult"`
	DomainCheckResults []checkResult `xml:"DomainCheckResult"`
}

// listResult represents a single domain in the getList response.
type listResult struct {
	Name      string `xml:"Name,attr"`
	Expires   string `xml:"Expires,attr"`
	IsExpired bool   `xml:"IsExpired,attr"`
	IsLocked  bool   `xml:"IsLocked,attr"`
	AutoRenew bool   `xml:"AutoRenew,attr"`
}

type domainListResponse struct {
	XMLName xml.Name     `xml:"DomainGetListResult"`
	Domains []listResult `xml:"Domain"`
}

type domainCreateResponse struct {
	XMLName    xml.Name `xml:"DomainCreateResult"`
	Domain     string   `xml:"Domain,attr"`
	Registered bool     `xml:"Registered,attr"`
}

// splitDomain splits a domain into SLD and TLD parts.
// e.g., "example.com" → ("example", "com"), "example.co.uk" → ("example", "co.uk")
func splitDomain(domain string) (sld, tld string) {
	parts := strings.SplitN(domain, ".", 2)
	if len(parts) < 2 {
		return domain, ""
	}
	return parts[0], parts[1]
}

// CheckAvailability checks if a domain is available on Namecheap.
func (c *Client) CheckAvailability(domain string) (*provider.DomainAvailability, error) {
	inner, err := c.doRequest("namecheap.domains.check", map[string]string{
		"DomainList": domain,
	})
	if err != nil {
		return nil, err
	}

	// Parse the check result directly (it's the root element in inner XML)
	var result checkResult
	if err := xml.Unmarshal(inner, &result); err != nil {
		return nil, fmt.Errorf("parse check result: %w", err)
	}

	return &provider.DomainAvailability{
		Registrar: "Namecheap",
		Domain:    domain,
		Available: result.Available,
		Premium:   result.Premium,
		Currency:  "USD",
	}, nil
}

// RegisterDomain registers a domain on Namecheap for 1 year.
func (c *Client) RegisterDomain(domain string) error {
	sld, tld := splitDomain(domain)
	if tld == "" {
		return fmt.Errorf("invalid domain: %s", domain)
	}

	params := map[string]string{
		"DomainName":                      domain,
		"Years":                           "1",
		"AuxBillingFirstName":             "Registration",
		"AuxBillingLastName":              "Admin",
		"AuxBillingAddress1":              "N/A",
		"AuxBillingCity":                  "N/A",
		"AuxBillingStateProvince":         "N/A",
		"AuxBillingPostalCode":            "00000",
		"AuxBillingCountry":               "US",
		"AuxBillingPhone":                 "+1.0000000000",
		"AuxBillingEmailAddress":          c.APIUser + "@namecheap.com",
		"TechFirstName":                   "Registration",
		"TechLastName":                    "Admin",
		"TechAddress1":                    "N/A",
		"TechCity":                        "N/A",
		"TechStateProvince":               "N/A",
		"TechPostalCode":                  "00000",
		"TechCountry":                     "US",
		"TechPhone":                       "+1.0000000000",
		"TechEmailAddress":                c.APIUser + "@namecheap.com",
		"AdminFirstName":                  "Registration",
		"AdminLastName":                   "Admin",
		"AdminAddress1":                   "N/A",
		"AdminCity":                       "N/A",
		"AdminStateProvince":              "N/A",
		"AdminPostalCode":                 "00000",
		"AdminCountry":                    "US",
		"AdminPhone":                      "+1.0000000000",
		"AdminEmailAddress":               c.APIUser + "@namecheap.com",
		"RegistrantFirstName":             "Registration",
		"RegistrantLastName":              "Admin",
		"RegistrantAddress1":              "N/A",
		"RegistrantCity":                  "N/A",
		"RegistrantStateProvince":         "N/A",
		"RegistrantPostalCode":            "00000",
		"RegistrantCountry":               "US",
		"RegistrantPhone":                 "+1.0000000000",
		"RegistrantEmailAddress":          c.APIUser + "@namecheap.com",
		"AddFreeWhoisguard":               "yes",
		"WGEnabled":                       "yes",
	}

	inner, err := c.doRequest("namecheap.domains.create", params)
	if err != nil {
		return err
	}

	var result domainCreateResponse
	if err := xml.Unmarshal(inner, &result); err != nil {
		return fmt.Errorf("parse create result: %w", err)
	}

	if !result.Registered {
		return fmt.Errorf("namecheap: domain %s registration failed (SLD=%s, TLD=%s)", domain, sld, tld)
	}
	return nil
}

// ListDomains lists all domains on Namecheap.
func (c *Client) ListDomains() ([]provider.Domain, error) {
	var allDomains []provider.Domain
	page := 1
	pageSize := 100

	for {
		inner, err := c.doRequest("namecheap.domains.getList", map[string]string{
			"Page":     fmt.Sprintf("%d", page),
			"PageSize": fmt.Sprintf("%d", pageSize),
		})
		if err != nil {
			return nil, err
		}

		var result domainListResponse
		if err := xml.Unmarshal(inner, &result); err != nil {
			return nil, fmt.Errorf("parse domain list: %w", err)
		}

		for _, d := range result.Domains {
			status := "active"
			if d.IsExpired {
				status = "expired"
			}
			allDomains = append(allDomains, provider.Domain{
				Name:      d.Name,
				Registrar: "Namecheap",
				Status:    status,
				ExpiresAt: d.Expires,
				AutoRenew: d.AutoRenew,
			})
		}

		if len(result.Domains) < pageSize {
			break
		}
		page++
	}

	return allDomains, nil
}
