// Package notify provides notification clients for alerting on system events.
package notify

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const lineNotifyEndpoint = "https://notify-api.line.me/api/notify"

var httpClient = &http.Client{Timeout: 10 * time.Second}

// LineClient sends notifications via LINE Notify API.
type LineClient struct {
	token string
}

// NewLineClient creates a LINE Notify client. Returns nil if token is empty.
func NewLineClient(token string) *LineClient {
	if token == "" {
		return nil
	}
	return &LineClient{token: token}
}

// Send sends a message via LINE Notify. Non-blocking — logs on failure.
func (c *LineClient) Send(message string) {
	if c == nil {
		return
	}
	go func() {
		if err := c.send(message); err != nil {
			log.Printf("WARN: LINE notify failed: %v", err)
		}
	}()
}

func (c *LineClient) send(message string) error {
	body := url.Values{"message": {message}}
	req, err := http.NewRequest("POST", lineNotifyEndpoint, strings.NewReader(body.Encode()))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("LINE Notify returned status %d", resp.StatusCode)
	}
	return nil
}
