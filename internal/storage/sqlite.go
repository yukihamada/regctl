package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// DiscoveryEntry represents a domain that appeared in the discovery feed.
type DiscoveryEntry struct {
	Domain    string `json:"domain"`
	SearchCnt int    `json:"search_count"`
	FirstSeen string `json:"first_seen"`
}

// Store wraps a SQLite database for search logging and discovery.
type Store struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS search_log (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    domain      TEXT NOT NULL,
    searcher_id TEXT NOT NULL DEFAULT 'anon',
    available   INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_search_available ON search_log(available, created_at) WHERE available=1;
CREATE INDEX IF NOT EXISTS idx_search_domain ON search_log(domain, created_at);
CREATE INDEX IF NOT EXISTS idx_search_user ON search_log(searcher_id, created_at);

CREATE TABLE IF NOT EXISTS daily_check_count (
    searcher_id TEXT NOT NULL,
    check_date  TEXT NOT NULL,
    count       INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (searcher_id, check_date)
);

CREATE TABLE IF NOT EXISTS referral_credits (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    domain        TEXT NOT NULL UNIQUE,
    searcher_id   TEXT NOT NULL,
    registrant_id TEXT NOT NULL,
    credit_cents  INTEGER NOT NULL,
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE IF NOT EXISTS auth_codes (
    email       TEXT NOT NULL,
    code        TEXT NOT NULL,
    expires_at  TEXT NOT NULL,
    used        INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_auth_codes_email ON auth_codes(email, used);

CREATE TABLE IF NOT EXISTS sites (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    domain       TEXT NOT NULL UNIQUE,
    customer_id  TEXT NOT NULL,
    machine_id   TEXT NOT NULL DEFAULT '',
    app_name     TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'provisioning',
    tier         TEXT NOT NULL DEFAULT 'free',
    max_req_day  INTEGER NOT NULL DEFAULT 1000,
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE IF NOT EXISTS site_usage (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id         INTEGER NOT NULL,
    usage_date      TEXT NOT NULL,
    request_count   INTEGER NOT NULL DEFAULT 0,
    bandwidth_bytes INTEGER NOT NULL DEFAULT 0,
    billed_cents    INTEGER NOT NULL DEFAULT 0,
    UNIQUE(site_id, usage_date)
);

-- Domain portfolio: tracks domains registered through regctl by each user
CREATE TABLE IF NOT EXISTS domain_portfolio (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    domain          TEXT NOT NULL UNIQUE,
    customer_id     TEXT NOT NULL,
    registrar       TEXT NOT NULL DEFAULT '',
    purchase_price_cents INTEGER NOT NULL DEFAULT 0,
    purchased_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_portfolio_customer ON domain_portfolio(customer_id);

-- Market listings: domains listed for sale by users
CREATE TABLE IF NOT EXISTS market_listings (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    domain          TEXT NOT NULL UNIQUE,
    seller_id       TEXT NOT NULL,
    ask_price_cents INTEGER NOT NULL,
    purchase_price_cents INTEGER NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'active', -- active | sold | cancelled
    listed_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    sold_at         TEXT
);
CREATE INDEX IF NOT EXISTS idx_market_status ON market_listings(status, listed_at);
CREATE INDEX IF NOT EXISTS idx_market_seller ON market_listings(seller_id);

CREATE TABLE IF NOT EXISTS site_sponsors (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id             INTEGER NOT NULL,
    sponsor_email       TEXT NOT NULL,
    sponsor_customer_id TEXT NOT NULL DEFAULT '',
    amount_cents        INTEGER NOT NULL,
    site_credit_cents   INTEGER NOT NULL DEFAULT 0,
    sponsor_token_cents INTEGER NOT NULL DEFAULT 0,
    stripe_session      TEXT NOT NULL DEFAULT '',
    created_at          TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
`

// New opens (or creates) the SQLite database at dbPath and applies the schema.
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite writes are serialized
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &Store{db: db}, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// LogSearch records a domain check and returns the caller's daily count.
func (s *Store) LogSearch(domain, searcherID string, available bool) (dailyCount int, err error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	avail := 0
	if available {
		avail = 1
	}

	_, err = tx.Exec(
		`INSERT INTO search_log (domain, searcher_id, available) VALUES (?, ?, ?)`,
		domain, searcherID, avail,
	)
	if err != nil {
		return 0, fmt.Errorf("insert search_log: %w", err)
	}

	today := time.Now().UTC().Format("2006-01-02")
	_, err = tx.Exec(
		`INSERT INTO daily_check_count (searcher_id, check_date, count)
		 VALUES (?, ?, 1)
		 ON CONFLICT(searcher_id, check_date) DO UPDATE SET count = count + 1`,
		searcherID, today,
	)
	if err != nil {
		return 0, fmt.Errorf("upsert daily_check_count: %w", err)
	}

	err = tx.QueryRow(
		`SELECT count FROM daily_check_count WHERE searcher_id = ? AND check_date = ?`,
		searcherID, today,
	).Scan(&dailyCount)
	if err != nil {
		return 0, fmt.Errorf("read daily count: %w", err)
	}

	return dailyCount, tx.Commit()
}

// GetDiscoveryFeed returns available domains searched 24-48 hours ago,
// ranked by popularity (search count), with pagination.
func (s *Store) GetDiscoveryFeed(limit, offset int) ([]DiscoveryEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	now := time.Now().UTC()
	t48 := now.Add(-48 * time.Hour).Format("2006-01-02T15:04:05Z")
	t24 := now.Add(-24 * time.Hour).Format("2006-01-02T15:04:05Z")

	rows, err := s.db.Query(
		`SELECT domain, COUNT(*) as cnt, MIN(created_at) as first_seen
		 FROM search_log
		 WHERE available = 1
		   AND created_at >= ?
		   AND created_at <= ?
		 GROUP BY domain
		 HAVING COUNT(DISTINCT searcher_id) >= 2
		 ORDER BY cnt DESC
		 LIMIT ? OFFSET ?`,
		t48, t24, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("query discovery: %w", err)
	}
	defer rows.Close()

	var entries []DiscoveryEntry
	for rows.Next() {
		var e DiscoveryEntry
		if err := rows.Scan(&e.Domain, &e.SearchCnt, &e.FirstSeen); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []DiscoveryEntry{}
	}
	return entries, rows.Err()
}

// GetFirstSearcher returns the searcher_id of the earliest search for a
// domain where available=1. Returns empty string if not found.
func (s *Store) GetFirstSearcher(domain string) (string, error) {
	var id string
	err := s.db.QueryRow(
		`SELECT searcher_id FROM search_log
		 WHERE domain = ? AND available = 1 AND searcher_id != 'anon'
		 ORDER BY created_at ASC LIMIT 1`,
		domain,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return id, err
}

// RecordReferralCredit records a referral credit. Returns (true, nil) on
// success and (false, nil) if the domain already has a referral entry.
func (s *Store) RecordReferralCredit(domain, searcherID, registrantID string, creditCents int64) (bool, error) {
	res, err := s.db.Exec(
		`INSERT OR IGNORE INTO referral_credits (domain, searcher_id, registrant_id, credit_cents)
		 VALUES (?, ?, ?, ?)`,
		domain, searcherID, registrantID, creditCents,
	)
	if err != nil {
		return false, fmt.Errorf("insert referral: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// GetDailyCheckCount returns today's check count for the given searcher.
func (s *Store) GetDailyCheckCount(searcherID string) (int, error) {
	today := time.Now().UTC().Format("2006-01-02")
	var count int
	err := s.db.QueryRow(
		`SELECT count FROM daily_check_count WHERE searcher_id = ? AND check_date = ?`,
		searcherID, today,
	).Scan(&count)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return count, err
}

// CountDistinctSearchers returns the number of distinct authenticated
// searcher IDs (excluding IP-hashed anon users) who searched for a domain
// where available=1.
func (s *Store) CountDistinctSearchers(domain string) (int, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(DISTINCT searcher_id) FROM search_log
		 WHERE domain = ? AND available = 1
		   AND searcher_id NOT LIKE 'ip_%'`,
		domain,
	).Scan(&count)
	return count, err
}

// CleanupOldData deletes search logs older than 90 days and
// daily_check_count entries older than 7 days.
func (s *Store) CleanupOldData() error {
	cutoff90 := time.Now().UTC().Add(-90 * 24 * time.Hour).Format("2006-01-02T15:04:05Z")
	cutoff7 := time.Now().UTC().Add(-7 * 24 * time.Hour).Format("2006-01-02")

	if _, err := s.db.Exec(`DELETE FROM search_log WHERE created_at < ?`, cutoff90); err != nil {
		return fmt.Errorf("cleanup search_log: %w", err)
	}
	if _, err := s.db.Exec(`DELETE FROM daily_check_count WHERE check_date < ?`, cutoff7); err != nil {
		return fmt.Errorf("cleanup daily_check_count: %w", err)
	}
	return nil
}

// StoreAuthCode saves a 6-digit verification code for the given email.
func (s *Store) StoreAuthCode(email, code string, expiresAt time.Time) error {
	_, err := s.db.Exec(
		`INSERT INTO auth_codes (email, code, expires_at) VALUES (?, ?, ?)`,
		email, code, expiresAt.UTC().Format("2006-01-02T15:04:05Z"),
	)
	if err != nil {
		return fmt.Errorf("store auth code: %w", err)
	}
	return nil
}

// VerifyAuthCode checks whether a valid, unused code exists for the email.
// On success it marks the code as used and returns true.
func (s *Store) VerifyAuthCode(email, code string) (bool, error) {
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	res, err := s.db.Exec(
		`UPDATE auth_codes SET used = 1
		 WHERE email = ? AND code = ? AND used = 0 AND expires_at > ?`,
		email, code, now,
	)
	if err != nil {
		return false, fmt.Errorf("verify auth code: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// GetLatestAuthCode returns the most recent unused, non-expired code for the email.
// Used only by the admin debug endpoint.
func (s *Store) GetLatestAuthCode(email string) (string, error) {
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	var code string
	err := s.db.QueryRow(
		`SELECT code FROM auth_codes
		 WHERE email = ? AND used = 0 AND expires_at > ?
		 ORDER BY created_at DESC LIMIT 1`,
		email, now,
	).Scan(&code)
	if err != nil {
		return "", fmt.Errorf("get auth code: %w", err)
	}
	return code, nil
}

// Site represents a hosted site.
type Site struct {
	ID         int64  `json:"id"`
	Domain     string `json:"domain"`
	CustomerID string `json:"customer_id"`
	MachineID  string `json:"machine_id"`
	AppName    string `json:"app_name"`
	Status     string `json:"status"`
	Tier       string `json:"tier"`
	MaxReqDay  int    `json:"max_req_day"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// SiteUsage represents daily usage for a site.
type SiteUsage struct {
	ID             int64  `json:"id"`
	SiteID         int64  `json:"site_id"`
	UsageDate      string `json:"usage_date"`
	RequestCount   int64  `json:"request_count"`
	BandwidthBytes int64  `json:"bandwidth_bytes"`
	BilledCents    int64  `json:"billed_cents"`
}

// SiteSponsor represents a sponsor payment for a site.
type SiteSponsor struct {
	ID                int64  `json:"id"`
	SiteID            int64  `json:"site_id"`
	SponsorEmail      string `json:"sponsor_email"`
	SponsorCustomerID string `json:"sponsor_customer_id"`
	AmountCents       int64  `json:"amount_cents"`
	SiteCreditCents   int64  `json:"site_credit_cents"`
	SponsorTokenCents int64  `json:"sponsor_token_cents"`
	StripeSession     string `json:"stripe_session"`
	CreatedAt         string `json:"created_at"`
}

// CreateSite inserts a new site record.
func (s *Store) CreateSite(domain, customerID, machineID, appName string) (*Site, error) {
	res, err := s.db.Exec(
		`INSERT INTO sites (domain, customer_id, machine_id, app_name, status)
		 VALUES (?, ?, ?, ?, 'provisioning')`,
		domain, customerID, machineID, appName,
	)
	if err != nil {
		return nil, fmt.Errorf("create site: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.getSiteByID(id)
}

func (s *Store) getSiteByID(id int64) (*Site, error) {
	var site Site
	err := s.db.QueryRow(
		`SELECT id, domain, customer_id, machine_id, app_name, status, tier, max_req_day, created_at, updated_at
		 FROM sites WHERE id = ?`, id,
	).Scan(&site.ID, &site.Domain, &site.CustomerID, &site.MachineID, &site.AppName,
		&site.Status, &site.Tier, &site.MaxReqDay, &site.CreatedAt, &site.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get site by id: %w", err)
	}
	return &site, nil
}

// GetSite returns a site by domain.
func (s *Store) GetSite(domain string) (*Site, error) {
	var site Site
	err := s.db.QueryRow(
		`SELECT id, domain, customer_id, machine_id, app_name, status, tier, max_req_day, created_at, updated_at
		 FROM sites WHERE domain = ?`, domain,
	).Scan(&site.ID, &site.Domain, &site.CustomerID, &site.MachineID, &site.AppName,
		&site.Status, &site.Tier, &site.MaxReqDay, &site.CreatedAt, &site.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get site: %w", err)
	}
	return &site, nil
}

// GetSitesByCustomer returns all sites owned by a customer.
func (s *Store) GetSitesByCustomer(customerID string) ([]Site, error) {
	rows, err := s.db.Query(
		`SELECT id, domain, customer_id, machine_id, app_name, status, tier, max_req_day, created_at, updated_at
		 FROM sites WHERE customer_id = ? ORDER BY created_at DESC`, customerID,
	)
	if err != nil {
		return nil, fmt.Errorf("list sites: %w", err)
	}
	defer rows.Close()

	var sites []Site
	for rows.Next() {
		var site Site
		if err := rows.Scan(&site.ID, &site.Domain, &site.CustomerID, &site.MachineID, &site.AppName,
			&site.Status, &site.Tier, &site.MaxReqDay, &site.CreatedAt, &site.UpdatedAt); err != nil {
			return nil, err
		}
		sites = append(sites, site)
	}
	if sites == nil {
		sites = []Site{}
	}
	return sites, rows.Err()
}

// UpdateSiteStatus updates a site's status.
func (s *Store) UpdateSiteStatus(domain, status string) error {
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	_, err := s.db.Exec(
		`UPDATE sites SET status = ?, updated_at = ? WHERE domain = ?`,
		status, now, domain,
	)
	if err != nil {
		return fmt.Errorf("update site status: %w", err)
	}
	return nil
}

// UpdateSiteMachine updates a site's machine_id.
func (s *Store) UpdateSiteMachine(domain, machineID string) error {
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	_, err := s.db.Exec(
		`UPDATE sites SET machine_id = ?, updated_at = ? WHERE domain = ?`,
		machineID, now, domain,
	)
	if err != nil {
		return fmt.Errorf("update site machine: %w", err)
	}
	return nil
}

// DeleteSite removes a site record.
func (s *Store) DeleteSite(domain string) error {
	_, err := s.db.Exec(`DELETE FROM sites WHERE domain = ?`, domain)
	if err != nil {
		return fmt.Errorf("delete site: %w", err)
	}
	return nil
}

// IncrementUsage atomically increments request count for a site on a given date.
func (s *Store) IncrementUsage(siteID int64, date string, requestCount int64, bandwidthBytes int64) error {
	_, err := s.db.Exec(
		`INSERT INTO site_usage (site_id, usage_date, request_count, bandwidth_bytes)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(site_id, usage_date) DO UPDATE SET
		   request_count = request_count + excluded.request_count,
		   bandwidth_bytes = bandwidth_bytes + excluded.bandwidth_bytes`,
		siteID, date, requestCount, bandwidthBytes,
	)
	if err != nil {
		return fmt.Errorf("increment usage: %w", err)
	}
	return nil
}

// GetTodayUsage returns today's usage for a site.
func (s *Store) GetTodayUsage(siteID int64) (*SiteUsage, error) {
	today := time.Now().UTC().Format("2006-01-02")
	var u SiteUsage
	err := s.db.QueryRow(
		`SELECT id, site_id, usage_date, request_count, bandwidth_bytes, billed_cents
		 FROM site_usage WHERE site_id = ? AND usage_date = ?`, siteID, today,
	).Scan(&u.ID, &u.SiteID, &u.UsageDate, &u.RequestCount, &u.BandwidthBytes, &u.BilledCents)
	if err == sql.ErrNoRows {
		return &SiteUsage{SiteID: siteID, UsageDate: today}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get today usage: %w", err)
	}
	return &u, nil
}

// GetUsage returns usage records for a site within a date range.
func (s *Store) GetUsage(siteID int64, from, to string) ([]SiteUsage, error) {
	rows, err := s.db.Query(
		`SELECT id, site_id, usage_date, request_count, bandwidth_bytes, billed_cents
		 FROM site_usage WHERE site_id = ? AND usage_date >= ? AND usage_date <= ?
		 ORDER BY usage_date DESC`, siteID, from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("get usage: %w", err)
	}
	defer rows.Close()

	var usage []SiteUsage
	for rows.Next() {
		var u SiteUsage
		if err := rows.Scan(&u.ID, &u.SiteID, &u.UsageDate, &u.RequestCount, &u.BandwidthBytes, &u.BilledCents); err != nil {
			return nil, err
		}
		usage = append(usage, u)
	}
	if usage == nil {
		usage = []SiteUsage{}
	}
	return usage, rows.Err()
}

// UpdateUsageBilled marks usage as billed.
func (s *Store) UpdateUsageBilled(siteID int64, date string, billedCents int64) error {
	_, err := s.db.Exec(
		`UPDATE site_usage SET billed_cents = ? WHERE site_id = ? AND usage_date = ?`,
		billedCents, siteID, date,
	)
	if err != nil {
		return fmt.Errorf("update usage billed: %w", err)
	}
	return nil
}

// AddSponsor records a sponsor payment.
func (s *Store) AddSponsor(siteID int64, email, sponsorCustomerID string, amountCents, siteCreditCents, sponsorTokenCents int64, stripeSession string) error {
	_, err := s.db.Exec(
		`INSERT INTO site_sponsors (site_id, sponsor_email, sponsor_customer_id, amount_cents, site_credit_cents, sponsor_token_cents, stripe_session)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		siteID, email, sponsorCustomerID, amountCents, siteCreditCents, sponsorTokenCents, stripeSession,
	)
	if err != nil {
		return fmt.Errorf("add sponsor: %w", err)
	}
	return nil
}

// GetSponsors returns sponsor records for a site.
func (s *Store) GetSponsors(siteID int64) ([]SiteSponsor, error) {
	rows, err := s.db.Query(
		`SELECT id, site_id, sponsor_email, sponsor_customer_id, amount_cents, site_credit_cents, sponsor_token_cents, stripe_session, created_at
		 FROM site_sponsors WHERE site_id = ? ORDER BY created_at DESC`, siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("get sponsors: %w", err)
	}
	defer rows.Close()

	var sponsors []SiteSponsor
	for rows.Next() {
		var sp SiteSponsor
		if err := rows.Scan(&sp.ID, &sp.SiteID, &sp.SponsorEmail, &sp.SponsorCustomerID,
			&sp.AmountCents, &sp.SiteCreditCents, &sp.SponsorTokenCents, &sp.StripeSession, &sp.CreatedAt); err != nil {
			return nil, err
		}
		sponsors = append(sponsors, sp)
	}
	if sponsors == nil {
		sponsors = []SiteSponsor{}
	}
	return sponsors, rows.Err()
}

// GetAllActiveSites returns all sites with status 'active'.
func (s *Store) GetAllActiveSites() ([]Site, error) {
	rows, err := s.db.Query(
		`SELECT id, domain, customer_id, machine_id, app_name, status, tier, max_req_day, created_at, updated_at
		 FROM sites WHERE status = 'active' ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("get active sites: %w", err)
	}
	defer rows.Close()

	var sites []Site
	for rows.Next() {
		var site Site
		if err := rows.Scan(&site.ID, &site.Domain, &site.CustomerID, &site.MachineID, &site.AppName,
			&site.Status, &site.Tier, &site.MaxReqDay, &site.CreatedAt, &site.UpdatedAt); err != nil {
			return nil, err
		}
		sites = append(sites, site)
	}
	if sites == nil {
		sites = []Site{}
	}
	return sites, rows.Err()
}

// ── Portfolio ──────────────────────────────────────────────────────────────

// PortfolioEntry represents a domain owned by a user.
type PortfolioEntry struct {
	ID                 int64  `json:"id"`
	Domain             string `json:"domain"`
	CustomerID         string `json:"customer_id"`
	Registrar          string `json:"registrar"`
	PurchasePriceCents int64  `json:"purchase_price_cents"`
	PurchasedAt        string `json:"purchased_at"`
}

// AddToPortfolio records a domain purchase.
func (s *Store) AddToPortfolio(domain, customerID, registrar string, priceCents int64) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO domain_portfolio (domain, customer_id, registrar, purchase_price_cents)
		 VALUES (?, ?, ?, ?)`,
		domain, customerID, registrar, priceCents,
	)
	return err
}

// GetPortfolio returns all domains owned by a customer.
func (s *Store) GetPortfolio(customerID string) ([]PortfolioEntry, error) {
	rows, err := s.db.Query(
		`SELECT id, domain, customer_id, registrar, purchase_price_cents, purchased_at
		 FROM domain_portfolio WHERE customer_id = ? ORDER BY purchased_at DESC`,
		customerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []PortfolioEntry
	for rows.Next() {
		var e PortfolioEntry
		if err := rows.Scan(&e.ID, &e.Domain, &e.CustomerID, &e.Registrar, &e.PurchasePriceCents, &e.PurchasedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []PortfolioEntry{}
	}
	return entries, rows.Err()
}

// GetPortfolioEntry returns a single portfolio entry.
func (s *Store) GetPortfolioEntry(domain string) (*PortfolioEntry, error) {
	var e PortfolioEntry
	err := s.db.QueryRow(
		`SELECT id, domain, customer_id, registrar, purchase_price_cents, purchased_at
		 FROM domain_portfolio WHERE domain = ?`, domain,
	).Scan(&e.ID, &e.Domain, &e.CustomerID, &e.Registrar, &e.PurchasePriceCents, &e.PurchasedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &e, err
}

// TransferPortfolio changes the owner of a domain (market sale).
func (s *Store) TransferPortfolio(domain, newCustomerID string, priceCents int64) error {
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	_, err := s.db.Exec(
		`UPDATE domain_portfolio SET customer_id = ?, purchase_price_cents = ?, purchased_at = ?
		 WHERE domain = ?`,
		newCustomerID, priceCents, now, domain,
	)
	return err
}

// ── Market Listings ─────────────────────────────────────────────────────────

// MarketListing represents a domain listed for sale.
type MarketListing struct {
	ID                 int64   `json:"id"`
	Domain             string  `json:"domain"`
	SellerID           string  `json:"seller_id"`
	AskPriceCents      int64   `json:"ask_price_cents"`
	PurchasePriceCents int64   `json:"purchase_price_cents"`
	Status             string  `json:"status"`
	ListedAt           string  `json:"listed_at"`
	SoldAt             *string `json:"sold_at,omitempty"`
}

// ListForSale creates or updates a market listing.
func (s *Store) ListForSale(domain, sellerID string, askPriceCents, purchasePriceCents int64) error {
	_, err := s.db.Exec(
		`INSERT INTO market_listings (domain, seller_id, ask_price_cents, purchase_price_cents)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(domain) DO UPDATE SET
		   ask_price_cents = excluded.ask_price_cents,
		   status = 'active',
		   sold_at = NULL,
		   listed_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')`,
		domain, sellerID, askPriceCents, purchasePriceCents,
	)
	return err
}

// CancelListing removes a listing (seller action).
func (s *Store) CancelListing(domain, sellerID string) error {
	_, err := s.db.Exec(
		`UPDATE market_listings SET status = 'cancelled'
		 WHERE domain = ? AND seller_id = ? AND status = 'active'`,
		domain, sellerID,
	)
	return err
}

// GetListing returns a single active listing.
func (s *Store) GetListing(domain string) (*MarketListing, error) {
	var m MarketListing
	err := s.db.QueryRow(
		`SELECT id, domain, seller_id, ask_price_cents, purchase_price_cents, status, listed_at, sold_at
		 FROM market_listings WHERE domain = ? AND status = 'active'`, domain,
	).Scan(&m.ID, &m.Domain, &m.SellerID, &m.AskPriceCents, &m.PurchasePriceCents, &m.Status, &m.ListedAt, &m.SoldAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &m, err
}

// GetActiveListings returns all active market listings.
func (s *Store) GetActiveListings(limit, offset int) ([]MarketListing, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(
		`SELECT id, domain, seller_id, ask_price_cents, purchase_price_cents, status, listed_at, sold_at
		 FROM market_listings WHERE status = 'active'
		 ORDER BY listed_at DESC LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var listings []MarketListing
	for rows.Next() {
		var m MarketListing
		if err := rows.Scan(&m.ID, &m.Domain, &m.SellerID, &m.AskPriceCents, &m.PurchasePriceCents, &m.Status, &m.ListedAt, &m.SoldAt); err != nil {
			return nil, err
		}
		listings = append(listings, m)
	}
	if listings == nil {
		listings = []MarketListing{}
	}
	return listings, rows.Err()
}

// MarkSold marks a listing as sold.
func (s *Store) MarkSold(domain string) error {
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	_, err := s.db.Exec(
		`UPDATE market_listings SET status = 'sold', sold_at = ? WHERE domain = ? AND status = 'active'`,
		now, domain,
	)
	return err
}
