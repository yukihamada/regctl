package server

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
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
			log.Printf("send verification email: %v", err) // no PII in logs
			writeError(w, http.StatusInternalServerError, "failed to send email", "")
			return
		}
	} else {
		// Never log the code itself — use server logs/DB for debugging
		log.Printf("email client not configured; auth code stored for %s (check DB)", req.Email)
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


// handleGoogleStart initiates Google OAuth flow.
// POST /v1/auth/google/start  {local_redirect}
// local_redirect is encoded into the OAuth state parameter so the registered
// redirect_uri stays clean (no query params that Google would reject).
func (s *Server) handleGoogleStart(w http.ResponseWriter, r *http.Request) {
	if s.googleClientID == "" {
		writeError(w, http.StatusServiceUnavailable, "Google auth not configured", "")
		return
	}

	var req struct {
		LocalRedirect string `json:"local_redirect"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	// Encode local_redirect into state: "<random_hex>|<base64(local_redirect)>"
	state := generateState()
	if req.LocalRedirect != "" {
		state = state + "|" + base64.RawURLEncoding.EncodeToString([]byte(req.LocalRedirect))
	}

	authURL := fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=email+profile&state=%s&access_type=offline",
		s.googleClientID, s.googleRedirectURI, state,
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

	// Extract local_redirect from state (format: "<random_hex>|<base64(url)>")
	var localRedirect string
	if state := r.URL.Query().Get("state"); strings.Contains(state, "|") {
		parts := strings.SplitN(state, "|", 2)
		if decoded, err := base64.RawURLEncoding.DecodeString(parts[1]); err == nil {
			localRedirect = string(decoded)
		}
	}
	// Fallback: legacy ?local_redirect= query param
	if localRedirect == "" {
		localRedirect = r.URL.Query().Get("local_redirect")
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

	// Redirect to local_redirect (web or CLI) if provided via state — whitelist only
	if localRedirect != "" && isAllowedRedirect(localRedirect) {
		http.Redirect(w, r, localRedirect+"?key="+apiKey+"&email="+email, http.StatusFound)
		return
	} else if localRedirect != "" {
		log.Printf("WARN: blocked open redirect attempt to: %s", localRedirect)
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
<pre style="background:#f5f5f5; padding:12px; border-radius:8px;">regctl config set api-key %s</pre>
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

// isAllowedRedirect validates that a localRedirect target is safe.
// Only regctl:// deep links and known origins are allowed to prevent open redirect attacks.
func isAllowedRedirect(u string) bool {
	return strings.HasPrefix(u, "regctl://") ||
		strings.HasPrefix(u, "http://localhost") ||
		strings.HasPrefix(u, "https://regctl.sh") ||
		strings.HasPrefix(u, "https://regctl-api.fly.dev")
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
