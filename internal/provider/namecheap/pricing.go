package namecheap

import (
	"encoding/xml"
	"fmt"
	"strconv"

	"github.com/yukihamada/regctl/internal/provider"
)

// productPrice represents a single product price from the pricing API.
type productPrice struct {
	Duration          string `xml:"Duration,attr"`
	DurationType      string `xml:"DurationType,attr"`
	Price             string `xml:"Price,attr"`
	RegularPrice      string `xml:"RegularPrice,attr"`
	YourPrice         string `xml:"YourPrice,attr"`
	CouponPrice       string `xml:"CouponPrice,attr"`
	Currency          string `xml:"Currency,attr"`
}

type productCategory struct {
	Name     string `xml:"Name,attr"`
	Products []struct {
		Name   string         `xml:"Name,attr"`
		Prices []productPrice `xml:"Price"`
	} `xml:"Product"`
}

type pricingResponse struct {
	XMLName    xml.Name          `xml:"UserGetPricingResult"`
	Categories []productCategory `xml:"ProductType>ProductCategory"`
}

// FetchPricing fetches TLD pricing from Namecheap users.getPricing API.
func (c *Client) FetchPricing() (map[string]provider.DomainAvailability, error) {
	inner, err := c.doRequest("namecheap.users.getPricing", map[string]string{
		"ProductType": "DOMAIN",
		"ActionName":  "REGISTER",
	})
	if err != nil {
		return nil, err
	}

	var result pricingResponse
	if err := xml.Unmarshal(inner, &result); err != nil {
		return nil, fmt.Errorf("parse pricing: %w", err)
	}

	prices := make(map[string]provider.DomainAvailability)
	for _, cat := range result.Categories {
		for _, prod := range cat.Products {
			tld := prod.Name
			if len(prod.Prices) == 0 {
				continue
			}
			// Use the first (1-year) price
			p := prod.Prices[0]
			price, _ := strconv.ParseFloat(p.YourPrice, 64)
			if price == 0 {
				price, _ = strconv.ParseFloat(p.Price, 64)
			}
			prices[tld] = provider.DomainAvailability{
				Registrar: "Namecheap",
				RegPrice:  price,
				RenPrice:  price,
				Currency:  p.Currency,
			}
		}
	}
	return prices, nil
}
