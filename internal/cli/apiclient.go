package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// APIClient is an HTTP client for the regctl API server.
type APIClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// NewAPIClient creates a new API client.
func NewAPIClient(baseURL, apiKey string) *APIClient {
	return &APIClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

type apiResult struct {
	OK        bool            `json:"ok"`
	Data      json.RawMessage `json:"data,omitempty"`
	Error     string          `json:"error,omitempty"`
	Hint      string          `json:"hint,omitempty"`
	Timestamp string          `json:"timestamp"`
}

func (c *APIClient) do(method, path string, body string) (*apiResult, error) {
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var result apiResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if !result.OK {
		msg := result.Error
		if result.Hint != "" {
			msg += " (" + result.Hint + ")"
		}
		return nil, fmt.Errorf("%s", msg)
	}
	return &result, nil
}

// GetBalance returns the account balance.
func (c *APIClient) GetBalance() (json.RawMessage, error) {
	r, err := c.do("GET", "/v1/billing/balance", "")
	if err != nil {
		return nil, err
	}
	return r.Data, nil
}

// CreateTopUp creates a top-up checkout session.
func (c *APIClient) CreateTopUp(amountCents int64) (json.RawMessage, error) {
	body := fmt.Sprintf(`{"amount_cents":%d}`, amountCents)
	r, err := c.do("POST", "/v1/billing/topup", body)
	if err != nil {
		return nil, err
	}
	return r.Data, nil
}

// RequestEmailAuth sends a verification code to the given email.
func (c *APIClient) RequestEmailAuth(email string) error {
	body := fmt.Sprintf(`{"email":%q}`, email)
	_, err := c.do("POST", "/v1/auth/email", body)
	return err
}

// VerifyEmailCode verifies the 6-digit code and returns (api_key, email, error).
func (c *APIClient) VerifyEmailCode(email, code string) (string, string, error) {
	body := fmt.Sprintf(`{"email":%q,"code":%q}`, email, code)
	r, err := c.do("POST", "/v1/auth/verify", body)
	if err != nil {
		return "", "", err
	}
	var data struct {
		APIKey string `json:"api_key"`
		Email  string `json:"email"`
	}
	json.Unmarshal(r.Data, &data)
	return data.APIKey, data.Email, nil
}

// GitHubDeviceResponse holds the GitHub device flow initial response.
type GitHubDeviceResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	Interval        int    `json:"interval"`
}

// StartGitHubDevice initiates the GitHub device authentication flow.
func (c *APIClient) StartGitHubDevice() (*GitHubDeviceResponse, error) {
	r, err := c.do("POST", "/v1/auth/github/device", "")
	if err != nil {
		return nil, err
	}
	var resp GitHubDeviceResponse
	json.Unmarshal(r.Data, &resp)
	return &resp, nil
}

// PollGitHubDevice polls for the GitHub device flow result.
// Returns ("", "", nil) if still pending.
func (c *APIClient) PollGitHubDevice(deviceCode string) (string, string, error) {
	body := fmt.Sprintf(`{"device_code":%q}`, deviceCode)
	r, err := c.do("POST", "/v1/auth/github/poll", body)
	if err != nil {
		return "", "", err
	}
	var data struct {
		APIKey string `json:"api_key"`
		Email  string `json:"email"`
		Status string `json:"status"`
	}
	json.Unmarshal(r.Data, &data)
	if data.APIKey == "" {
		return "", "", nil // still pending
	}
	return data.APIKey, data.Email, nil
}

// HoldBalanceResult is the result of a hold request.
type HoldBalanceResult struct {
	HoldID       string `json:"hold_id"`
	AmountCents  int64  `json:"amount_cents"`
	BalanceAfter int64  `json:"balance_after"`
}

// HoldBalance reserves amount from balance. Returns hold details.
func (c *APIClient) HoldBalance(amountCents int64, description string) (*HoldBalanceResult, error) {
	body := fmt.Sprintf(`{"amount_cents":%d,"description":%q}`, amountCents, description)
	r, err := c.do("POST", "/v1/billing/hold", body)
	if err != nil {
		return nil, err
	}
	var result HoldBalanceResult
	json.Unmarshal(r.Data, &result)
	return &result, nil
}

// ConfirmHold confirms a hold (finalizes the charge).
func (c *APIClient) ConfirmHold(holdID string) error {
	body := fmt.Sprintf(`{"hold_id":%q}`, holdID)
	_, err := c.do("POST", "/v1/billing/confirm", body)
	return err
}

// ReleaseHold releases a hold (refunds the amount).
func (c *APIClient) ReleaseHold(holdID string) error {
	body := fmt.Sprintf(`{"hold_id":%q}`, holdID)
	_, err := c.do("POST", "/v1/billing/release", body)
	return err
}

// GetBalanceCents returns the account balance in cents.
func (c *APIClient) GetBalanceCents() (int64, error) {
	r, err := c.do("GET", "/v1/billing/balance", "")
	if err != nil {
		return 0, err
	}
	var data struct {
		BalanceCents int64 `json:"balance_cents"`
	}
	json.Unmarshal(r.Data, &data)
	return data.BalanceCents, nil
}

// GoogleAuthResponse holds the Google OAuth start response.
type GoogleAuthResponse struct {
	AuthURL string `json:"auth_url"`
	State   string `json:"state"`
}

// StartGoogleAuth initiates the Google OAuth flow.
func (c *APIClient) StartGoogleAuth(redirectURI string) (*GoogleAuthResponse, error) {
	body := fmt.Sprintf(`{"redirect_uri":%q}`, redirectURI)
	r, err := c.do("POST", "/v1/auth/google/start", body)
	if err != nil {
		return nil, err
	}
	var resp GoogleAuthResponse
	json.Unmarshal(r.Data, &resp)
	return &resp, nil
}

// CreateSite creates a new hosted site.
func (c *APIClient) CreateSite(domain string) (json.RawMessage, error) {
	body := fmt.Sprintf(`{"domain":%q}`, domain)
	r, err := c.do("POST", "/v1/sites", body)
	if err != nil {
		return nil, err
	}
	return r.Data, nil
}

// ListSites returns the caller's hosted sites.
func (c *APIClient) ListSites() (json.RawMessage, error) {
	r, err := c.do("GET", "/v1/sites", "")
	if err != nil {
		return nil, err
	}
	return r.Data, nil
}

// GetSiteStatus returns a site's details and usage.
func (c *APIClient) GetSiteStatus(domain string) (json.RawMessage, error) {
	r, err := c.do("GET", "/v1/sites/"+domain, "")
	if err != nil {
		return nil, err
	}
	return r.Data, nil
}

// DeleteSite deletes a hosted site.
func (c *APIClient) DeleteSite(domain string) error {
	_, err := c.do("DELETE", "/v1/sites/"+domain, "")
	return err
}

// DeploySite uploads a tar.gz file to a site.
func (c *APIClient) DeploySite(domain string, data []byte) (json.RawMessage, error) {
	req, err := http.NewRequest("POST", c.baseURL+"/v1/sites/"+domain+"/deploy", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	req.Header.Set("Content-Type", "application/gzip")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("deploy request: %w", err)
	}
	defer resp.Body.Close()

	var result apiResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if !result.OK {
		msg := result.Error
		if result.Hint != "" {
			msg += " (" + result.Hint + ")"
		}
		return nil, fmt.Errorf("%s", msg)
	}
	return result.Data, nil
}
