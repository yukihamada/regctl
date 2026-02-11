package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yukihamada/regctl/internal/provider"
)

const nanobotAPI = "https://chatweb.ai/api/v1/chat"

// Fly app IPs for regctl-api (from `fly ips list --app regctl-api`)
const (
	flyIPv4 = "66.241.124.38"
	flyIPv6 = "2a09:8280:1::d1:da84:0"
)

const defaultSiteHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>%[1]s</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:system-ui,-apple-system,sans-serif;background:#0a0a0a;color:#fff;min-height:100vh;display:flex;align-items:center;justify-content:center;text-align:center}
.c{padding:2rem;max-width:600px}
h1{font-size:clamp(2rem,8vw,4rem);margin-bottom:.5rem;background:linear-gradient(135deg,#22c55e,#10b981);-webkit-background-clip:text;-webkit-text-fill-color:transparent;background-clip:text}
.sub{color:#888;font-size:1.1rem;margin-bottom:2rem}
.card{background:#111;border:1px solid #222;border-radius:12px;padding:1.5rem;margin-top:2rem;text-align:left}
.card h3{color:#22c55e;font-size:.9rem;margin-bottom:.5rem}
.card code{display:block;background:#0a0a0a;padding:10px 14px;border-radius:8px;font-size:.85rem;color:#ccc;margin-top:.5rem;word-break:break-all}
.footer{margin-top:3rem;color:#333;font-size:.75rem}
.footer a{color:#555;text-decoration:none}
</style>
</head>
<body>
<div class="c">
<h1>%[1]s</h1>
<p class="sub">This domain is live!</p>
<div class="card">
<h3>Deploy your site</h3>
<code>curl -fsSL https://regctl.sh/install.sh | sh<br>regctl hosting deploy %[1]s ./your-site/</code>
</div>
<p class="footer">Powered by <a href="https://regctl.sh">regctl.sh</a></p>
</div>
</body>
</html>`

// autoSetupSite runs after domain registration to set up DNS, TLS cert, and site record.
// Runs in a goroutine — logs errors instead of returning them.
func (s *Server) autoSetupSite(domain, registrarName, customerID string) {
	log.Printf("auto-setup: %s via %s for %s", domain, registrarName, customerID)

	// 1. Set DNS A/AAAA records pointing to this Fly app
	s.autoSetupDNS(domain, registrarName)

	// 2. Add Fly TLS certificate
	if s.flyClient != nil {
		if err := s.flyClient.AddCertificate(s.flyAppName, domain); err != nil {
			log.Printf("auto-cert: %s: %v", domain, err)
		} else {
			log.Printf("auto-cert: %s added", domain)
		}
	}

	// 3. Create site record (shared hosting — no dedicated machine)
	if s.store != nil {
		existing, _ := s.store.GetSite(domain)
		if existing == nil {
			if _, err := s.store.CreateSite(domain, customerID, "", ""); err != nil {
				log.Printf("auto-site: create %s: %v", domain, err)
			} else {
				s.store.UpdateSiteStatus(domain, "active")
				log.Printf("auto-site: %s created", domain)
			}
		}
	}
}

func (s *Server) autoSetupDNS(domain, registrarName string) {
	reg := s.getRegistrar(registrarName)
	if reg == nil {
		log.Printf("auto-dns: registrar %q not found", registrarName)
		return
	}

	dns, ok := reg.(provider.DNSProvider)
	if !ok {
		log.Printf("auto-dns: %s doesn't implement DNSProvider", registrarName)
		return
	}

	// A record → Fly shared IPv4
	if err := dns.AddRecord(domain, provider.DNSRecord{
		Type:    "A",
		Name:    "",
		Content: flyIPv4,
		TTL:     300,
	}); err != nil {
		log.Printf("auto-dns: %s A record: %v", domain, err)
	} else {
		log.Printf("auto-dns: %s A → %s", domain, flyIPv4)
	}

	// AAAA record → Fly dedicated IPv6
	if err := dns.AddRecord(domain, provider.DNSRecord{
		Type:    "AAAA",
		Name:    "",
		Content: flyIPv6,
		TTL:     300,
	}); err != nil {
		log.Printf("auto-dns: %s AAAA record: %v", domain, err)
	}
}

// serveSite handles HTTP requests for hosted domains.
// Returns true if the domain matched a hosted site.
func (s *Server) serveSite(w http.ResponseWriter, r *http.Request, host string) bool {
	if s.store == nil {
		return false
	}

	site, err := s.store.GetSite(host)
	if err != nil || site == nil {
		return false
	}

	// Check for custom content on disk
	siteDir := filepath.Join("/data/sites", host)
	reqPath := r.URL.Path
	if reqPath == "/" || reqPath == "" {
		reqPath = "/index.html"
	}
	filePath := filepath.Join(siteDir, filepath.Clean(reqPath))

	// Security: prevent path traversal
	if !strings.HasPrefix(filePath, siteDir) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return true
	}

	if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
		http.ServeFile(w, r, filePath)
		return true
	}

	// Serve default landing page
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=60")
	fmt.Fprintf(w, defaultSiteHTML, host)
	return true
}

// isAPIDomain returns true if the host is our API/admin domain (not a hosted site).
func isAPIDomain(host string) bool {
	return host == "regctl-api.fly.dev" ||
		host == "regctl.sh" ||
		host == "localhost" ||
		strings.HasSuffix(host, ".fly.dev") ||
		strings.HasPrefix(host, "127.") ||
		strings.HasPrefix(host, "192.168.") ||
		strings.HasPrefix(host, "[::") ||
		host == "::1" ||
		host == ""
}

const siteGenPrompt = `You are a professional web designer. Generate a complete, self-contained HTML page.

Rules:
- Output ONLY valid HTML (<!DOCTYPE html> to </html>). No markdown, no explanation.
- All CSS must be inline in a <style> tag. No external stylesheets or CDNs.
- All JS must be inline in a <script> tag. No external scripts.
- Make it responsive, modern, and visually polished.
- Use a dark theme (#0a0a0a background) unless the user requests otherwise.
- Include proper meta tags (charset, viewport).
- The site is for the domain: %s

User's request: %s`

// handleGenerateSite generates a website using AI from a text description.
func (s *Server) handleGenerateSite(w http.ResponseWriter, r *http.Request) {
	domain := r.PathValue("domain")

	var req struct {
		Prompt string `json:"prompt"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 65536)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body", "")
		return
	}
	if req.Prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt is required", "Describe what kind of site you want")
		return
	}

	// Call nanobot API
	prompt := fmt.Sprintf(siteGenPrompt, domain, req.Prompt)
	chatReq := map[string]interface{}{
		"message":    prompt,
		"session_id": "regctl-site-" + domain,
		"model":      "claude-sonnet-4-5-20250929",
		"device":     "pc",
	}
	body, err := json.Marshal(chatReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "marshal: "+err.Error(), "")
		return
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Post(nanobotAPI, "application/json", bytes.NewReader(body))
	if err != nil {
		writeError(w, http.StatusBadGateway, "AI service error: "+err.Error(), "")
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		writeError(w, http.StatusBadGateway, "read AI response: "+err.Error(), "")
		return
	}

	var chatResp struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		writeError(w, http.StatusBadGateway, "parse AI response: "+err.Error(), "")
		return
	}

	html := chatResp.Response
	if html == "" {
		writeError(w, http.StatusBadGateway, "AI returned empty response", "Try a different description")
		return
	}

	// Extract HTML from response (AI might wrap in markdown code blocks)
	html = extractHTML(html)

	// Save to disk
	siteDir := filepath.Join("/data/sites", domain)
	if err := os.MkdirAll(siteDir, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, "create dir: "+err.Error(), "")
		return
	}
	indexPath := filepath.Join(siteDir, "index.html")
	if err := os.WriteFile(indexPath, []byte(html), 0644); err != nil {
		writeError(w, http.StatusInternalServerError, "write file: "+err.Error(), "")
		return
	}

	// Ensure site record exists
	if s.store != nil {
		existing, _ := s.store.GetSite(domain)
		if existing == nil {
			customerID := getCustomerID(r)
			s.store.CreateSite(domain, customerID, "", "")
			s.store.UpdateSiteStatus(domain, "active")
		}
	}

	log.Printf("ai-generate: %s → %d bytes", domain, len(html))

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"domain":   domain,
		"site_url": "https://" + domain,
		"size":     len(html),
		"status":   "deployed",
	})
}

// extractHTML strips markdown code fences if the AI wrapped HTML in them.
func extractHTML(s string) string {
	s = strings.TrimSpace(s)
	// Remove ```html ... ``` wrapper
	if strings.HasPrefix(s, "```") {
		lines := strings.SplitN(s, "\n", 2)
		if len(lines) == 2 {
			s = lines[1]
		}
		if idx := strings.LastIndex(s, "```"); idx > 0 {
			s = s[:idx]
		}
	}
	s = strings.TrimSpace(s)
	return s
}
