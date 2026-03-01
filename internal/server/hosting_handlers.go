package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
	"github.com/yukihamada/regctl/internal/billing"
)

// handleCreateSite creates a new hosted site.
func (s *Server) handleCreateSite(w http.ResponseWriter, r *http.Request) {
	customerID := getCustomerID(r)
	if customerID == "" {
		writeError(w, http.StatusUnauthorized, "billing account required", "Sign up first")
		return
	}

	var req struct {
		Domain string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body", "")
		return
	}
	if req.Domain == "" {
		writeError(w, http.StatusBadRequest, "domain is required", "")
		return
	}

	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "storage not configured", "")
		return
	}

	// Check if site already exists
	existing, err := s.store.GetSite(req.Domain)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "check site: "+err.Error(), "")
		return
	}
	if existing != nil {
		writeError(w, http.StatusConflict, "site already exists for this domain", "")
		return
	}

	// Charge for site creation
	if !s.chargeBilling(w, r, billing.OpSiteCreate, 0) {
		return
	}

	// Create Fly Machine
	var machineID string
	if s.flyClient != nil {
		env := map[string]string{
			"SITE_DOMAIN":    req.Domain,
			"REGCTL_API_URL": s.baseURL,
			"SITE_SUSPENDED": "false",
		}
		if s.internalSecret != "" {
			env["INTERNAL_SECRET"] = s.internalSecret
		}
		mid, err := s.flyClient.CreateMachine("site-"+sanitizeName(req.Domain), req.Domain, env)
		if err != nil {
			// Refund the creation charge
			s.refundBilling(r, billing.OpSiteCreate, 0)
			writeError(w, http.StatusInternalServerError, "create machine: "+err.Error(), "")
			return
		}
		machineID = mid
	}

	appName := ""
	if s.flyClient != nil {
		appName = s.flyAppName
	}
	site, err := s.store.CreateSite(req.Domain, customerID, machineID, appName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create site: "+err.Error(), "")
		return
	}

	// Mark as active
	if err := s.store.UpdateSiteStatus(req.Domain, "active"); err != nil {
		log.Printf("WARN: failed to mark site active: %v", err)
	}
	site.Status = "active"

	writeJSON(w, http.StatusCreated, site)
}

// handleListSites returns the caller's sites.
func (s *Server) handleListSites(w http.ResponseWriter, r *http.Request) {
	customerID := getCustomerID(r)
	if customerID == "" {
		writeError(w, http.StatusUnauthorized, "billing account required", "")
		return
	}
	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "storage not configured", "")
		return
	}

	sites, err := s.store.GetSitesByCustomer(customerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	writeJSON(w, http.StatusOK, sites)
}

// handleGetSite returns a single site's details.
func (s *Server) handleGetSite(w http.ResponseWriter, r *http.Request) {
	customerID := getCustomerID(r)
	domain := r.PathValue("domain")

	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "storage not configured", "")
		return
	}

	site, err := s.store.GetSite(domain)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	if site == nil {
		writeError(w, http.StatusNotFound, "site not found", "")
		return
	}
	if site.CustomerID != customerID {
		writeError(w, http.StatusForbidden, "not your site", "")
		return
	}

	// Include today's usage
	usage, _ := s.store.GetTodayUsage(site.ID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"site":        site,
		"today_usage": usage,
	})
}

// handleDeleteSite removes a site and its machine.
func (s *Server) handleDeleteSite(w http.ResponseWriter, r *http.Request) {
	customerID := getCustomerID(r)
	domain := r.PathValue("domain")

	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "storage not configured", "")
		return
	}

	site, err := s.store.GetSite(domain)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	if site == nil {
		writeError(w, http.StatusNotFound, "site not found", "")
		return
	}
	if site.CustomerID != customerID {
		writeError(w, http.StatusForbidden, "not your site", "")
		return
	}

	// Delete the Fly Machine
	if s.flyClient != nil && site.MachineID != "" {
		if err := s.flyClient.DeleteMachine(site.MachineID); err != nil {
			log.Printf("WARN: failed to delete machine %s: %v", site.MachineID, err)
		}
	}

	if err := s.store.DeleteSite(domain); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"deleted": domain})
}

// handleDeploySite receives a tar.gz and updates the site's machine.
func (s *Server) handleDeploySite(w http.ResponseWriter, r *http.Request) {
	customerID := getCustomerID(r)
	domain := r.PathValue("domain")

	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "storage not configured", "")
		return
	}

	site, err := s.store.GetSite(domain)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	if site == nil {
		writeError(w, http.StatusNotFound, "site not found", "")
		return
	}
	if site.CustomerID != customerID {
		writeError(w, http.StatusForbidden, "not your site", "")
		return
	}

	// Charge for deploy
	if !s.chargeBilling(w, r, billing.OpSiteDeploy, 0) {
		return
	}

	// Read the upload body (max 500MB)
	const maxUploadSize = 500 << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		s.refundBilling(r, billing.OpSiteDeploy, 0)
		writeError(w, http.StatusBadRequest, "read upload: "+err.Error(), "Max size: 500MB")
		return
	}

	_ = bodyData // In production, this would be uploaded to the machine via Fly volumes or a sidecar

	// Update machine env to trigger redeploy
	if s.flyClient != nil && site.MachineID != "" {
		env := map[string]string{
			"SITE_DOMAIN":    domain,
			"REGCTL_API_URL": s.baseURL,
			"SITE_SUSPENDED": "false",
			"DEPLOY_VERSION": fmt.Sprintf("%d", time.Now().Unix()),
		}
		if s.internalSecret != "" {
			env["INTERNAL_SECRET"] = s.internalSecret
		}
		if err := s.flyClient.UpdateMachine(site.MachineID, env); err != nil {
			s.refundBilling(r, billing.OpSiteDeploy, 0)
			writeError(w, http.StatusInternalServerError, "update machine: "+err.Error(), "")
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"deployed": domain,
		"size":     len(bodyData),
	})
}

// handleSiteUsage returns usage statistics for a site.
func (s *Server) handleSiteUsage(w http.ResponseWriter, r *http.Request) {
	customerID := getCustomerID(r)
	domain := r.PathValue("domain")

	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "storage not configured", "")
		return
	}

	site, err := s.store.GetSite(domain)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	if site == nil {
		writeError(w, http.StatusNotFound, "site not found", "")
		return
	}
	if site.CustomerID != customerID {
		writeError(w, http.StatusForbidden, "not your site", "")
		return
	}

	// Default: last 30 days
	to := time.Now().UTC().Format("2006-01-02")
	from := time.Now().UTC().AddDate(0, 0, -30).Format("2006-01-02")
	if q := r.URL.Query().Get("from"); q != "" {
		from = q
	}
	if q := r.URL.Query().Get("to"); q != "" {
		to = q
	}

	usage, err := s.store.GetUsage(site.ID, from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"domain": domain,
		"from":   from,
		"to":     to,
		"usage":  usage,
	})
}

// handleSponsorSite creates a Stripe Checkout session for sponsoring a site.
func (s *Server) handleSponsorSite(w http.ResponseWriter, r *http.Request) {
	domain := r.PathValue("domain")

	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "storage not configured", "")
		return
	}

	site, err := s.store.GetSite(domain)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	if site == nil {
		writeError(w, http.StatusNotFound, "site not found", "")
		return
	}

	var req struct {
		Email       string `json:"email"`
		AmountCents int64  `json:"amount_cents"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body", "")
		return
	}
	if req.Email == "" || !strings.Contains(req.Email, "@") || !strings.Contains(req.Email, ".") {
		writeError(w, http.StatusBadRequest, "valid email required", "")
		return
	}
	if req.AmountCents < 100 || req.AmountCents > 100_000_00 { // max $10,000
		writeError(w, http.StatusBadRequest, "amount_cents must be between 100 and 1000000", "")
		return
	}

	if !s.billingEnabled {
		writeError(w, http.StatusServiceUnavailable, "billing not enabled", "")
		return
	}

	successURL := s.baseURL + "/?sponsored=" + domain
	cancelURL := s.baseURL + "/"

	params := &stripe.CheckoutSessionParams{
		CustomerEmail: stripe.String(req.Email),
		Mode:          stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency:   stripe.String(string(stripe.CurrencyUSD)),
					UnitAmount: stripe.Int64(req.AmountCents),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String(fmt.Sprintf("Sponsor %s", domain)),
					},
				},
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
	}
	params.AddMetadata("type", "site_sponsor")
	params.AddMetadata("domain", domain)
	params.AddMetadata("owner_customer_id", site.CustomerID)
	params.AddMetadata("sponsor_email", req.Email)
	params.AddMetadata("amount_cents", fmt.Sprintf("%d", req.AmountCents))

	sess, err := session.New(params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create checkout session: "+err.Error(), "")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"checkout_url": sess.URL,
		"session_id":   sess.ID,
	})
}

// handleSiteRequestBatch receives request count batches from site machines.
func (s *Server) handleSiteRequestBatch(w http.ResponseWriter, r *http.Request) {
	// Validate internal secret — always required for this endpoint
	if s.internalSecret == "" {
		writeError(w, http.StatusServiceUnavailable, "internal endpoint not configured", "")
		return
	}
	auth := r.Header.Get("Authorization")
	if auth != "Bearer "+s.internalSecret {
		writeError(w, http.StatusUnauthorized, "invalid internal secret", "")
		return
	}

	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "storage not configured", "")
		return
	}

	var req struct {
		SiteDomain     string `json:"site_domain"`
		RequestCount   int64  `json:"request_count"`
		BandwidthBytes int64  `json:"bandwidth_bytes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body", "")
		return
	}

	site, err := s.store.GetSite(req.SiteDomain)
	if err != nil || site == nil {
		writeError(w, http.StatusNotFound, "site not found", "")
		return
	}

	today := time.Now().UTC().Format("2006-01-02")
	if err := s.store.IncrementUsage(site.ID, today, req.RequestCount, req.BandwidthBytes); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// sanitizeName converts a domain to a valid machine name.
func sanitizeName(domain string) string {
	name := ""
	for _, c := range domain {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			name += string(c)
		} else if c == '.' {
			name += "-"
		}
	}
	if len(name) > 60 {
		name = name[:60]
	}
	return name
}

// runDailyBilling runs daily usage billing for all active sites.
func (s *Server) runDailyBilling() {
	if s.store == nil || !s.billingEnabled {
		return
	}

	sites, err := s.store.GetAllActiveSites()
	if err != nil {
		log.Printf("daily billing: list sites: %v", err)
		return
	}

	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")

	for _, site := range sites {
		usage, err := s.store.GetUsage(site.ID, yesterday, yesterday)
		if err != nil || len(usage) == 0 {
			continue
		}
		u := usage[0]
		if u.BilledCents > 0 {
			continue // already billed
		}

		freeReqs := int64(billing.FreeTierMaxReqDay)
		if site.Tier == "paid" {
			freeReqs = int64(billing.PaidTierFreeReqDay)
		}

		billableReqs := u.RequestCount - freeReqs
		if billableReqs <= 0 {
			continue
		}

		// $0.001 per request = 0.1 cents
		costCents := billableReqs * int64(billing.PaidReqCostMills) / 10
		if costCents <= 0 {
			costCents = 1 // minimum 1 cent
		}

		desc := fmt.Sprintf("Site %s usage %s: %d requests", site.Domain, yesterday, u.RequestCount)
		if err := billing.DeductBalance(site.CustomerID, costCents, desc); err != nil {
			log.Printf("daily billing: deduct %s: %v", site.Domain, err)
			// Suspend if balance insufficient
			if err.Error() == "insufficient balance" {
				s.suspendSiteByDomain(site.Domain)
			}
			continue
		}

		if err := s.store.UpdateUsageBilled(site.ID, yesterday, costCents); err != nil {
			log.Printf("daily billing: update billed %s: %v", site.Domain, err)
		}
	}
}

// suspendSiteByDomain suspends a site by domain name.
func (s *Server) suspendSiteByDomain(domain string) {
	if err := s.store.UpdateSiteStatus(domain, "suspended"); err != nil {
		log.Printf("suspend site %s: %v", domain, err)
		return
	}
	site, err := s.store.GetSite(domain)
	if err != nil || site == nil {
		return
	}
	if s.flyClient != nil && site.MachineID != "" {
		env := map[string]string{
			"SITE_DOMAIN":    domain,
			"SITE_SUSPENDED": "true",
			"REGCTL_API_URL": s.baseURL,
		}
		if err := s.flyClient.UpdateMachine(site.MachineID, env); err != nil {
			log.Printf("suspend machine %s: %v", site.MachineID, err)
		}
	}
}

// startDailyBillingWorker starts a background goroutine for daily billing.
func (s *Server) startDailyBillingWorker() {
	go func() {
		for {
			// Run at 00:05 UTC each day
			now := time.Now().UTC()
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 5, 0, 0, time.UTC)
			time.Sleep(time.Until(next))
			log.Println("running daily billing...")
			s.runDailyBilling()
		}
	}()
}
