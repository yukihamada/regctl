package spaceship

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	baseURL   = "https://spaceship.dev/api/v1"
	userAgent = "regctl/1.0"
)

// Client is the Spaceship API client.
type Client struct {
	APIKey    string
	APISecret string
	HTTP      *http.Client
}

// NewClient creates a new Spaceship client.
func NewClient(apiKey, apiSecret string) *Client {
	return &Client{
		APIKey:    apiKey,
		APISecret: apiSecret,
		HTTP:      &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Name() string { return "Spaceship" }

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 30 * time.Second}
}

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

	req.Header.Set("X-Api-Key", c.APIKey)
	req.Header.Set("X-Api-Secret", c.APISecret)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr struct {
			Detail string `json:"detail"`
		}
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Detail != "" {
			return fmt.Errorf("spaceship: %s (HTTP %d)", apiErr.Detail, resp.StatusCode)
		}
		return fmt.Errorf("spaceship: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decode: %w (body: %s)", err, string(respBody))
		}
	}
	return nil
}

// doRequestWithHeaders performs a request and returns response headers.
func (c *Client) doRequestWithHeaders(method, url string, body interface{}, result interface{}) (http.Header, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("X-Api-Key", c.APIKey)
	req.Header.Set("X-Api-Secret", c.APISecret)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr struct {
			Detail string `json:"detail"`
		}
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Detail != "" {
			return resp.Header, fmt.Errorf("spaceship: %s (HTTP %d)", apiErr.Detail, resp.StatusCode)
		}
		return resp.Header, fmt.Errorf("spaceship: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return resp.Header, fmt.Errorf("decode: %w (body: %s)", err, string(respBody))
		}
	}
	return resp.Header, nil
}
