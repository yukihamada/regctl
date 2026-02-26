package porkbun

import (
	"fmt"
	"strconv"

	"github.com/yukihamada/regctl/internal/provider"
)

type checkResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Response struct {
		Avail        string `json:"avail"`
		Price        string `json:"price"`
		RegularPrice string `json:"regularPrice"`
		Premium      string `json:"premium"`
		Additional   struct {
			Renewal struct {
				Price string `json:"price"`
			} `json:"renewal"`
		} `json:"additional"`
	} `json:"response"`
}

type createRequest struct {
	APIKey       string `json:"apikey"`
	SecretKey    string `json:"secretapikey"`
	Cost         int    `json:"cost"`
	AgreeToTerms string `json:"agreeToTerms"`
}

type createResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Domain  string `json:"domain"`
}

type listResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Domains []struct {
		Domain    string `json:"domain"`
		Status    string `json:"status"`
		TLD       string `json:"tld"`
		CreateAt  string `json:"createDate"`
		ExpireAt  string `json:"expireDate"`
		AutoRenew string `json:"autoRenew"`
	} `json:"domains"`
}

// CheckAvailability checks if a domain is available on Porkbun.
func (c *Client) CheckAvailability(domain string) (*provider.DomainAvailability, error) {
	var resp checkResponse
	if err := c.post("/domain/checkDomain/"+domain, c.auth(), &resp); err != nil {
		return nil, err
	}
	if resp.Status != "SUCCESS" {
		return nil, fmt.Errorf("porkbun: %s", resp.Message)
	}

	regPrice, _ := strconv.ParseFloat(resp.Response.Price, 64)
	renPrice, _ := strconv.ParseFloat(resp.Response.Additional.Renewal.Price, 64)

	return &provider.DomainAvailability{
		Registrar: "Porkbun",
		Domain:    domain,
		Available: resp.Response.Avail == "yes",
		Premium:   resp.Response.Premium == "yes",
		RegPrice:  regPrice,
		RenPrice:  renPrice,
		Currency:  "USD",
	}, nil
}

// RegisterDomain registers a domain on Porkbun.
func (c *Client) RegisterDomain(domain string) error {
	// First check price
	avail, err := c.CheckAvailability(domain)
	if err != nil {
		return fmt.Errorf("check price: %w", err)
	}
	if !avail.Available {
		return fmt.Errorf("%s is not available", domain)
	}

	costCents := int(avail.RegPrice * 100)

	var resp createResponse
	if err := c.post("/domain/create/"+domain, createRequest{
		APIKey:       c.APIKey,
		SecretKey:    c.SecretKey,
		Cost:         costCents,
		AgreeToTerms: "yes",
	}, &resp); err != nil {
		return err
	}
	if resp.Status != "SUCCESS" {
		return fmt.Errorf("porkbun register: %s", resp.Message)
	}
	return nil
}

// ListDomains lists all domains on Porkbun.
func (c *Client) ListDomains() ([]provider.Domain, error) {
	var resp listResponse
	if err := c.post("/domain/listAll", c.auth(), &resp); err != nil {
		return nil, err
	}
	if resp.Status != "SUCCESS" {
		return nil, fmt.Errorf("porkbun: %s", resp.Message)
	}

	var domains []provider.Domain
	for _, d := range resp.Domains {
		domains = append(domains, provider.Domain{
			Name:      d.Domain,
			Registrar: "Porkbun",
			Status:    d.Status,
			ExpiresAt: d.ExpireAt,
			AutoRenew: d.AutoRenew == "1",
		})
	}
	return domains, nil
}

type nsUpdateRequest struct {
	APIKey    string   `json:"apikey"`
	SecretKey string   `json:"secretapikey"`
	NS        []string `json:"ns"`
}

// UpdateNameservers updates the nameservers for a domain on Porkbun.
func (c *Client) UpdateNameservers(domain string, ns []string) error {
	var resp struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	if err := c.post("/domain/updateNs/"+domain, nsUpdateRequest{
		APIKey:    c.APIKey,
		SecretKey: c.SecretKey,
		NS:        ns,
	}, &resp); err != nil {
		return err
	}
	if resp.Status != "SUCCESS" {
		return fmt.Errorf("porkbun ns update: %s", resp.Message)
	}
	return nil
}

type nsGetResponse struct {
	Status  string   `json:"status"`
	Message string   `json:"message"`
	NS      []string `json:"ns"`
}

// GetNameservers returns the current nameservers for a domain.
func (c *Client) GetNameservers(domain string) ([]string, error) {
	var resp nsGetResponse
	if err := c.post("/domain/getNs/"+domain, c.auth(), &resp); err != nil {
		return nil, err
	}
	if resp.Status != "SUCCESS" {
		return nil, fmt.Errorf("porkbun get ns: %s", resp.Message)
	}
	return resp.NS, nil
}
