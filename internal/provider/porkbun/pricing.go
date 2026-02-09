package porkbun

import (
	"fmt"
	"strconv"

	"github.com/yukihamada/regctl/internal/provider"
)

type pricingResponse struct {
	Status  string                       `json:"status"`
	Pricing map[string]pricingTLDDetails `json:"pricing"`
}

type pricingTLDDetails struct {
	Registration string `json:"registration"`
	Renewal      string `json:"renewal"`
}

// FetchPricing gets all TLD prices from Porkbun (no auth needed).
func (c *Client) FetchPricing() (map[string]provider.DomainAvailability, error) {
	var resp pricingResponse
	if err := c.post("/pricing/get", struct{}{}, &resp); err != nil {
		return nil, err
	}
	if resp.Status != "SUCCESS" {
		return nil, fmt.Errorf("porkbun pricing failed")
	}

	result := make(map[string]provider.DomainAvailability, len(resp.Pricing))
	for tld, p := range resp.Pricing {
		reg, _ := strconv.ParseFloat(p.Registration, 64)
		ren, _ := strconv.ParseFloat(p.Renewal, 64)
		result[tld] = provider.DomainAvailability{
			Registrar: "Porkbun",
			RegPrice:  reg,
			RenPrice:  ren,
			Currency:  "USD",
		}
	}
	return result, nil
}

// FetchPricingStatic returns Porkbun pricing (public, no auth required).
func FetchPricingStatic() (map[string]provider.DomainAvailability, error) {
	c := &Client{}
	return c.FetchPricing()
}
