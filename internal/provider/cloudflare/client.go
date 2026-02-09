package cloudflare

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	baseURL   = "https://api.cloudflare.com/client/v4"
	userAgent = "regctl/1.0"
)

// Client is the Cloudflare API client.
// Supports both API Token (Bearer) and Global API Key (X-Auth-Key + X-Auth-Email).
type Client struct {
	APIToken   string // Bearer token
	GlobalKey  string // Global API key
	Email      string // Required for Global API key auth
	AccountID  string
	HTTP       *http.Client
}

// NewClient creates a Cloudflare client with API token auth.
func NewClient(apiToken string) *Client {
	return &Client{
		APIToken: apiToken,
		HTTP:     &http.Client{Timeout: 30 * time.Second},
	}
}

// NewClientGlobal creates a Cloudflare client with Global API Key auth.
func NewClientGlobal(globalKey, email, accountID string) *Client {
	return &Client{
		GlobalKey: globalKey,
		Email:     email,
		AccountID: accountID,
		HTTP:      &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Name() string { return "Cloudflare" }

func (c *Client) doRequest(method, url string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if c.GlobalKey != "" && c.Email != "" {
		req.Header.Set("X-Auth-Key", c.GlobalKey)
		req.Header.Set("X-Auth-Email", c.Email)
	} else if c.APIToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIToken)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decode: %w (body: %s)", err, string(respBody))
		}
	}
	return nil
}
