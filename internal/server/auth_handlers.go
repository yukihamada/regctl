package server

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/yukihamada/regctl/internal/billing"
)

// handleEmailAuth accepts an email and sends a 6-digit verification code.
// POST /v1/auth/email  {email}
func (s *Server) handleEmailAuth(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if !strings.Contains(req.Email, "@") || !strings.Contains(req.Email, ".") {
		writeError(w, http.StatusBadRequest, "invalid email address", "")
		return
	}

	code := generateCode()
	expiresAt := time.Now().UTC().Add(10 * time.Minute)

	if s.store == nil {
		writeError(w, http.StatusInternalServerError, "storage not configured", "")
		return
	}
	if err := s.store.StoreAuthCode(req.Email, code, expiresAt); err != nil {
		log.Printf("store auth code: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to store code", "")
		return
	}

	if s.emailClient != nil {
		if err := s.emailClient.SendVerificationCode(req.Email, code); err != nil {
			log.Printf("send verification email to %s: %v", req.Email, err)
			writeError(w, http.StatusInternalServerError, "failed to send email", "")
			return
		}
	} else {
		log.Printf("email client not configured, code for %s: %s", req.Email, code)
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "verification code sent",
	})
}

// handleVerify checks the 6-digit code and returns an API key.
// POST /v1/auth/verify  {email, code}
func (s *Server) handleVerify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Code = strings.TrimSpace(req.Code)

	if s.store == nil {
		writeError(w, http.StatusInternalServerError, "storage not configured", "")
		return
	}

	ok, err := s.store.VerifyAuthCode(req.Email, req.Code)
	if err != nil {
		log.Printf("verify auth code: %v", err)
		writeError(w, http.StatusInternalServerError, "verification failed", "")
		return
	}
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid or expired code", "")
		return
	}

	// Find or create Stripe customer
	cust, err := billing.FindOrCreateCustomer(req.Email)
	if err != nil {
		log.Printf("find/create customer for %s: %v", req.Email, err)
		writeError(w, http.StatusInternalServerError, "account setup failed", "")
		return
	}

	apiKey := s.billingClient.GenerateAPIKeyForCustomer(cust.ID)

	writeJSON(w, http.StatusOK, map[string]string{
		"api_key": apiKey,
		"email":   req.Email,
	})
}

// handleGitHubDevice starts a GitHub OAuth Device Flow.
// POST /v1/auth/github/device  {}
func (s *Server) handleGitHubDevice(w http.ResponseWriter, r *http.Request) {
	if s.githubClientID == "" {
		writeError(w, http.StatusServiceUnavailable, "GitHub auth not configured", "")
		return
	}

	resp, err := http.PostForm("https://github.com/login/device/code", map[string][]string{
		"client_id": {s.githubClientID},
		"scope":     {"user:email"},
	})
	if err != nil {
		log.Printf("github device code request: %v", err)
		writeError(w, http.StatusBadGateway, "failed to start GitHub auth", "")
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	// GitHub returns form-urlencoded: device_code=...&user_code=...&verification_uri=...&interval=...
	params := parseFormBody(string(body))

	writeJSON(w, http.StatusOK, map[string]string{
		"device_code":      params["device_code"],
		"user_code":        params["user_code"],
		"verification_uri": params["verification_uri"],
		"interval":         params["interval"],
	})
}

// handleGitHubPoll polls GitHub for the device flow result.
// POST /v1/auth/github/poll  {device_code}
func (s *Server) handleGitHubPoll(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceCode string `json:"device_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "")
		return
	}

	resp, err := http.PostForm("https://github.com/login/oauth/access_token", map[string][]string{
		"client_id":   {s.githubClientID},
		"device_code": {req.DeviceCode},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
	})
	if err != nil {
		log.Printf("github poll: %v", err)
		writeError(w, http.StatusBadGateway, "failed to poll GitHub", "")
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	params := parseFormBody(string(body))

	if params["error"] != "" {
		// Still pending or other error
		writeJSON(w, http.StatusOK, map[string]string{
			"status": params["error"],
		})
		return
	}

	accessToken := params["access_token"]
	if accessToken == "" {
		writeError(w, http.StatusBadGateway, "no access token received", "")
		return
	}

	// Get user email from GitHub
	email, err := getGitHubEmail(accessToken)
	if err != nil {
		log.Printf("github user email: %v", err)
		writeError(w, http.StatusBadGateway, "failed to get GitHub user info", "")
		return
	}

	apiKey, err := s.createAPIKeyForEmail(email)
	if err != nil {
		log.Printf("create api key for %s: %v", email, err)
		writeError(w, http.StatusInternalServerError, "account setup failed", "")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"api_key": apiKey,
		"email":   email,
	})
}

// handleGoogleStart initiates Google OAuth flow.
// POST /v1/auth/google/start  {redirect_uri}
func (s *Server) handleGoogleStart(w http.ResponseWriter, r *http.Request) {
	if s.googleClientID == "" {
		writeError(w, http.StatusServiceUnavailable, "Google auth not configured", "")
		return
	}

	var req struct {
		RedirectURI string `json:"redirect_uri"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	state := generateState()
	redirectURI := s.googleRedirectURI
	if req.RedirectURI != "" {
		redirectURI = req.RedirectURI
	}

	authURL := fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=email+profile&state=%s&access_type=offline",
		s.googleClientID, redirectURI, state,
	)

	writeJSON(w, http.StatusOK, map[string]string{
		"auth_url": authURL,
		"state":    state,
	})
}

// handleGoogleCallback handles the Google OAuth callback.
// GET /v1/auth/google/callback  ?code=&state=
func (s *Server) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "missing code parameter", "")
		return
	}

	// Exchange code for token
	resp, err := http.PostForm("https://oauth2.googleapis.com/token", map[string][]string{
		"code":          {code},
		"client_id":     {s.googleClientID},
		"client_secret": {s.googleClientSecret},
		"redirect_uri":  {s.googleRedirectURI},
		"grant_type":    {"authorization_code"},
	})
	if err != nil {
		log.Printf("google token exchange: %v", err)
		writeError(w, http.StatusBadGateway, "failed to exchange code", "")
		return
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&tokenResp)
	if tokenResp.Error != "" || tokenResp.AccessToken == "" {
		writeError(w, http.StatusBadGateway, "failed to get access token", "")
		return
	}

	// Get user info
	email, err := getGoogleEmail(tokenResp.AccessToken)
	if err != nil {
		log.Printf("google user info: %v", err)
		writeError(w, http.StatusBadGateway, "failed to get user info", "")
		return
	}

	apiKey, err := s.createAPIKeyForEmail(email)
	if err != nil {
		log.Printf("create api key for %s: %v", email, err)
		writeError(w, http.StatusInternalServerError, "account setup failed", "")
		return
	}

	// Check if this is a CLI redirect (localhost) or web redirect
	localRedirect := r.URL.Query().Get("local_redirect")
	if localRedirect != "" {
		http.Redirect(w, r, localRedirect+"?key="+apiKey+"&email="+email, http.StatusFound)
		return
	}

	// Return HTML page that shows the API key and tries to redirect to CLI
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><title>regctl - Authentication Complete</title>
<style>
body { font-family: -apple-system, sans-serif; max-width: 500px; margin: 80px auto; text-align: center; color: #333; }
.key { font-family: monospace; background: #1a1a2e; color: #10b981; padding: 16px; border-radius: 8px; word-break: break-all; font-size: 14px; cursor: pointer; }
.success { color: #10b981; font-size: 24px; }
</style></head>
<body>
<p class="success">Authentication successful!</p>
<p>Your API key:</p>
<div class="key" onclick="navigator.clipboard.writeText(this.textContent)">%s</div>
<p style="color:#666; font-size:14px;">Click to copy. Save this in your terminal:</p>
<pre style="background:#f5f5f5; padding:12px; border-radius:8px;">regctl config set regctl_billing_key %s</pre>
<p style="color:#999; font-size:12px;">You can close this window.</p>
</body></html>`, apiKey, apiKey)
}

// handleGitHubWebRedirect initiates GitHub OAuth for web (redirect flow).
// GET /v1/auth/github/web  ?redirect_uri=
func (s *Server) handleGitHubWebRedirect(w http.ResponseWriter, r *http.Request) {
	if s.githubClientID == "" {
		writeError(w, http.StatusServiceUnavailable, "GitHub auth not configured", "")
		return
	}
	redirectURI := r.URL.Query().Get("redirect_uri")
	if redirectURI == "" {
		redirectURI = "https://regctl.sh"
	}
	state := generateState()
	authURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&scope=user:email&state=%s&redirect_uri=%s",
		s.githubClientID, state, fmt.Sprintf("%s/v1/auth/github/callback?redirect_uri=%s", s.baseURL, redirectURI),
	)
	http.Redirect(w, r, authURL, http.StatusFound)
}

// handleGitHubCallback handles GitHub OAuth callback (web flow).
// GET /v1/auth/github/callback  ?code=&state=&redirect_uri=
func (s *Server) handleGitHubCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "missing code parameter", "")
		return
	}

	// Exchange code for access token
	resp, err := http.PostForm("https://github.com/login/oauth/access_token", map[string][]string{
		"client_id":     {s.githubClientID},
		"client_secret": {s.githubClientSecret},
		"code":          {code},
	})
	if err != nil {
		log.Printf("github token exchange: %v", err)
		writeError(w, http.StatusBadGateway, "failed to exchange code", "")
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	params := parseFormBody(string(body))
	accessToken := params["access_token"]
	if accessToken == "" {
		writeError(w, http.StatusBadGateway, "failed to get access token", "")
		return
	}

	email, err := getGitHubEmail(accessToken)
	if err != nil {
		log.Printf("github user email: %v", err)
		writeError(w, http.StatusBadGateway, "failed to get user info", "")
		return
	}

	apiKey, err := s.createAPIKeyForEmail(email)
	if err != nil {
		log.Printf("create api key for %s: %v", email, err)
		writeError(w, http.StatusInternalServerError, "account setup failed", "")
		return
	}

	redirectURI := r.URL.Query().Get("redirect_uri")
	if redirectURI != "" {
		http.Redirect(w, r, redirectURI+"?api_key="+apiKey+"&email="+email, http.StatusFound)
		return
	}

	// Same HTML response as Google callback
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><title>regctl - Authentication Complete</title>
<style>
body { font-family: -apple-system, sans-serif; max-width: 500px; margin: 80px auto; text-align: center; color: #333; }
.key { font-family: monospace; background: #1a1a2e; color: #10b981; padding: 16px; border-radius: 8px; word-break: break-all; font-size: 14px; cursor: pointer; }
.success { color: #10b981; font-size: 24px; }
</style></head>
<body>
<p class="success">Authentication successful!</p>
<p>Your API key:</p>
<div class="key" onclick="navigator.clipboard.writeText(this.textContent)">%s</div>
<p style="color:#666; font-size:14px;">Click to copy. Save this in your terminal:</p>
<pre style="background:#f5f5f5; padding:12px; border-radius:8px;">regctl config set regctl_billing_key %s</pre>
<p style="color:#999; font-size:12px;">You can close this window.</p>
</body></html>`, apiKey, apiKey)
}

// createAPIKeyForEmail finds or creates a Stripe customer and generates an API key.
func (s *Server) createAPIKeyForEmail(email string) (string, error) {
	cust, err := billing.FindOrCreateCustomer(email)
	if err != nil {
		return "", err
	}
	return s.billingClient.GenerateAPIKeyForCustomer(cust.ID), nil
}

// getGitHubEmail fetches the user's primary email from the GitHub API.
func getGitHubEmail(accessToken string) (string, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("github emails: %w", err)
	}
	defer resp.Body.Close()

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", fmt.Errorf("decode github emails: %w", err)
	}

	// Prefer primary+verified
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}
	for _, e := range emails {
		if e.Verified {
			return e.Email, nil
		}
	}
	if len(emails) > 0 {
		return emails[0].Email, nil
	}
	return "", fmt.Errorf("no email found on GitHub account")
}

// getGoogleEmail fetches the user's email from the Google userinfo endpoint.
func getGoogleEmail(accessToken string) (string, error) {
	req, _ := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("google userinfo: %w", err)
	}
	defer resp.Body.Close()

	var info struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", fmt.Errorf("decode google userinfo: %w", err)
	}
	if info.Email == "" {
		return "", fmt.Errorf("no email in Google account")
	}
	return info.Email, nil
}

// generateCode returns a random 6-digit code.
func generateCode() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	return fmt.Sprintf("%06d", n.Int64())
}

// generateState returns a random state string for OAuth flows.
func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// parseFormBody parses a URL-encoded form body into a map.
func parseFormBody(body string) map[string]string {
	result := make(map[string]string)
	for _, pair := range strings.Split(body, "&") {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}
