package namecheap

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	productionURL = "https://api.namecheap.com/xml.response"
	sandboxURL    = "https://api.sandbox.namecheap.com/xml.response"
	userAgent     = "regctl/1.0"
	ipifyURL      = "https://api.ipify.org"
)

// Client is the Namecheap API client.
type Client struct {
	APIUser  string
	APIKey   string
	UserName string
	ClientIP string
	HTTP     *http.Client
	Sandbox  bool
}

// NewClient creates a new Namecheap client.
// If clientIP is empty, it will be auto-detected on first request.
func NewClient(apiUser, apiKey, userName, clientIP string) *Client {
	return &Client{
		APIUser:  apiUser,
		APIKey:   apiKey,
		UserName: userName,
		ClientIP: clientIP,
		HTTP:     &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the provider name.
func (c *Client) Name() string { return "Namecheap" }

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func (c *Client) baseURL() string {
	if c.Sandbox {
		return sandboxURL
	}
	return productionURL
}

// apiResponse is the top-level XML envelope for all Namecheap API responses.
type apiResponse struct {
	XMLName xml.Name `xml:"ApiResponse"`
	Status  string   `xml:"Status,attr"`
	Errors  struct {
		Error []struct {
			Number  string `xml:"Number,attr"`
			Message string `xml:",chardata"`
		} `xml:"Error"`
	} `xml:"Errors"`
	CommandResponse struct {
		Inner []byte `xml:",innerxml"`
	} `xml:"CommandResponse"`
}

// doRequest executes a Namecheap API call and returns the CommandResponse inner XML.
func (c *Client) doRequest(command string, params map[string]string) ([]byte, error) {
	if c.ClientIP == "" {
		ip, err := detectIP(c.httpClient())
		if err != nil {
			return nil, fmt.Errorf("detect client IP: %w", err)
		}
		c.ClientIP = ip
	}

	q := url.Values{}
	q.Set("ApiUser", c.APIUser)
	q.Set("ApiKey", c.APIKey)
	q.Set("UserName", c.UserName)
	q.Set("ClientIp", c.ClientIP)
	q.Set("Command", command)
	for k, v := range params {
		q.Set(k, v)
	}

	reqURL := c.baseURL() + "?" + q.Encode()
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var apiResp apiResponse
	if err := xml.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("parse XML: %w", err)
	}

	if apiResp.Status != "OK" {
		if len(apiResp.Errors.Error) > 0 {
			return nil, fmt.Errorf("namecheap: %s", apiResp.Errors.Error[0].Message)
		}
		return nil, fmt.Errorf("namecheap: API returned status %s", apiResp.Status)
	}

	return apiResp.CommandResponse.Inner, nil
}

// detectIP fetches the public IPv4 address via ipify.
func detectIP(client *http.Client) (string, error) {
	resp, err := client.Get(ipifyURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
