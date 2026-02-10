package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client sends emails via the Resend API.
type Client struct {
	apiKey string
	from   string
	http   *http.Client
}

// NewClient creates a new Resend email client.
func NewClient(apiKey, from string) *Client {
	return &Client{
		apiKey: apiKey,
		from:   from,
		http:   &http.Client{Timeout: 10 * time.Second},
	}
}

type sendRequest struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	HTML    string `json:"html"`
}

// Send delivers an email via Resend.
func (c *Client) Send(to, subject, html string) error {
	body, err := json.Marshal(sendRequest{
		From:    c.from,
		To:      to,
		Subject: subject,
		HTML:    html,
	})
	if err != nil {
		return fmt.Errorf("marshal email: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend API error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// SendVerificationCode sends a 6-digit verification code to the given email.
func (c *Client) SendVerificationCode(to, code string) error {
	html := fmt.Sprintf(`<div style="font-family: sans-serif; max-width: 400px; margin: 0 auto;">
  <h2 style="color: #333;">regctl verification code</h2>
  <p style="font-size: 32px; font-weight: bold; letter-spacing: 8px; color: #10b981; padding: 16px 0;">%s</p>
  <p style="color: #666;">This code expires in 10 minutes.</p>
  <p style="color: #999; font-size: 12px;">If you didn't request this, you can safely ignore this email.</p>
</div>`, code)

	return c.Send(to, "Your regctl verification code: "+code, html)
}
