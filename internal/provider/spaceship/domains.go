package spaceship

import (
	"fmt"
	"time"

	"github.com/yukihamada/regctl/internal/provider"
)

type availResponse struct {
	Available bool   `json:"available"`
	Domain    string `json:"domain"`
}

type domainItem struct {
	Name               string   `json:"name"`
	AutoRenew          bool     `json:"auto_renew"`
	EppStatuses        []string `json:"epp_statuses"`
	VerificationStatus string   `json:"verification_status"`
	Hosts              []string `json:"hosts"`
	ExpirationDate     string   `json:"expiration_date"`
}

type listDomainsResponse struct {
	Items []domainItem `json:"items"`
	Total int          `json:"total"`
}

type asyncStatus struct {
	Status string `json:"status"`
	Detail string `json:"detail"`
}

// CheckAvailability checks if a domain is available on Spaceship.
func (c *Client) CheckAvailability(domain string) (*provider.DomainAvailability, error) {
	url := fmt.Sprintf("%s/domains/%s/available", baseURL, domain)
	var resp availResponse
	if err := c.doRequest("GET", url, nil, &resp); err != nil {
		return nil, err
	}

	return &provider.DomainAvailability{
		Registrar: "Spaceship",
		Domain:    domain,
		Available: resp.Available,
		Currency:  "USD",
	}, nil
}

// RegisterDomain registers a domain on Spaceship (async operation).
func (c *Client) RegisterDomain(domain string) error {
	url := fmt.Sprintf("%s/domains/%s", baseURL, domain)
	body := map[string]interface{}{
		"autoRenew": false,
		"years":     1,
		"privacyProtection": map[string]interface{}{
			"level":       "high",
			"userConsent": true,
		},
	}

	headers, err := c.doRequestWithHeaders("POST", url, body, nil)
	if err != nil {
		return err
	}

	// Poll async operation
	opID := headers.Get("Spaceship-Async-Operationid")
	if opID == "" {
		return nil // No async operation, assume success
	}

	return c.waitForOperation(opID)
}

func (c *Client) waitForOperation(opID string) error {
	url := fmt.Sprintf("%s/async-operations/%s", baseURL, opID)
	for i := 0; i < 30; i++ {
		time.Sleep(2 * time.Second)
		var status asyncStatus
		if err := c.doRequest("GET", url, nil, &status); err != nil {
			return err
		}
		switch status.Status {
		case "success":
			return nil
		case "failed":
			return fmt.Errorf("spaceship: registration failed: %s", status.Detail)
		}
		// pending — continue polling
	}
	return fmt.Errorf("spaceship: registration timed out (operation: %s)", opID)
}

// ListDomains lists all domains on Spaceship.
func (c *Client) ListDomains() ([]provider.Domain, error) {
	url := fmt.Sprintf("%s/domains?take=100&skip=0", baseURL)
	var resp listDomainsResponse
	if err := c.doRequest("GET", url, nil, &resp); err != nil {
		return nil, err
	}

	var domains []provider.Domain
	for _, d := range resp.Items {
		status := "active"
		for _, s := range d.EppStatuses {
			if s == "clientHold" || s == "serverHold" {
				status = s
				break
			}
		}
		domains = append(domains, provider.Domain{
			Name:      d.Name,
			Registrar: "Spaceship",
			Status:    status,
			ExpiresAt: d.ExpirationDate,
			AutoRenew: d.AutoRenew,
		})
	}
	return domains, nil
}
