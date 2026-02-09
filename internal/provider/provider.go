package provider

// DomainAvailability holds domain check result from a registrar.
type DomainAvailability struct {
	Registrar string  `json:"registrar"`
	Domain    string  `json:"domain"`
	Available bool    `json:"available"`
	Premium   bool    `json:"premium"`
	RegPrice  float64 `json:"reg_price"`
	RenPrice  float64 `json:"renew_price"`
	Currency  string  `json:"currency"`
}

// DNSRecord represents a DNS record across providers.
type DNSRecord struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority,omitempty"`
}

// Domain represents a registered domain across providers.
type Domain struct {
	Name      string `json:"name"`
	Registrar string `json:"registrar"`
	Status    string `json:"status"`
	ExpiresAt string `json:"expires_at"`
	AutoRenew bool   `json:"auto_renew"`
}

// Registrar is the interface for domain registration providers.
type Registrar interface {
	Name() string
	CheckAvailability(domain string) (*DomainAvailability, error)
	RegisterDomain(domain string) error
	ListDomains() ([]Domain, error)
}

// DNSProvider is the interface for DNS management providers.
type DNSProvider interface {
	Name() string
	ListRecords(domain string) ([]DNSRecord, error)
	AddRecord(domain string, rec DNSRecord) error
	DeleteRecord(domain, recordID string) error
}

// PriceFetcher can fetch bulk TLD pricing without auth.
type PriceFetcher interface {
	Name() string
	FetchPricing() (map[string]DomainAvailability, error)
}
