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
