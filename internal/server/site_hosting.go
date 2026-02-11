package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/yukihamada/regctl/internal/provider"
)

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
