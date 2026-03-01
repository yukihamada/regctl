package netim

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	baseURL   = "https://rest.netim.com/3.0"
	userAgent = "regctl/1.0"
)

// Client is the Netim REST API v3 client.
type Client struct {
	Login    string
	Password string
	http     *http.Client

	mu         sync.Mutex
	session    string
	sessionExp time.Time
}

// NewClient creates a new Netim client.
func NewClient(login, password string) *Client {
	return &Client{
		Login:    login,
		Password: password,
		http:     &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Name() string { return "Netim" }

// getSession returns a valid session token, re-authenticating if needed.
func (c *Client) getSession() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.session != "" && time.Now().Before(c.sessionExp) {
		return c.session, nil
	}

	// POST /session to authenticate
	body, _ := json.Marshal(map[string]string{
		"login":    c.Login,
		"password": c.Password,
	})
	req, err := http.NewRequest("POST", baseURL+"/session/", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("netim: login: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("netim: login HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Session string `json:"session"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil || result.Session == "" {
		return "", fmt.Errorf("netim: login: no session in response: %s", string(respBody))
	}

	c.session = result.Session
	c.sessionExp = time.Now().Add(25 * time.Minute) // Netim sessions last 30 min
	return c.session, nil
}

func (c *Client) doRequest(method, path string, body interface{}, result interface{}) error {
	session, err := c.getSession()
	if err != nil {
		return err
	}

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", session)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("netim: request %s: %w", path, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 401 {
		// Session expired — clear and retry once
		c.mu.Lock()
		c.session = ""
		c.mu.Unlock()
		return fmt.Errorf("netim: session expired, retry")
	}

	if resp.StatusCode >= 400 {
		var apiErr struct {
			Message string `json:"message"`
			Error   string `json:"error"`
		}
		if json.Unmarshal(respBody, &apiErr) == nil {
			if apiErr.Message != "" {
				return fmt.Errorf("netim: %s (HTTP %d)", apiErr.Message, resp.StatusCode)
			}
			if apiErr.Error != "" {
				return fmt.Errorf("netim: %s (HTTP %d)", apiErr.Error, resp.StatusCode)
			}
		}
		return fmt.Errorf("netim: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("netim: decode %s: %w (body: %s)", path, err, string(respBody))
		}
	}
	return nil
}
