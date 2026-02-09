package porkbun

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	baseURL   = "https://api.porkbun.com/api/json/v3"
	userAgent = "regctl/1.0"
)

// Client is the Porkbun API client.
type Client struct {
	APIKey    string
	SecretKey string
	HTTP      *http.Client
}

// NewClient creates a new Porkbun client.
func NewClient(apiKey, secretKey string) *Client {
	return &Client{
		APIKey:    apiKey,
		SecretKey: secretKey,
		HTTP:      &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Name() string { return "Porkbun" }

type authBody struct {
	APIKey    string `json:"apikey"`
	SecretKey string `json:"secretapikey"`
}

func (c *Client) auth() authBody {
	return authBody{APIKey: c.APIKey, SecretKey: c.SecretKey}
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func (c *Client) post(path string, payload interface{}, result interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequest("POST", baseURL+path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	if result != nil {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("decode: %w (body: %s)", err, string(body))
		}
	}
	return nil
}
