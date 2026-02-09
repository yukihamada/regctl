package valuedomain

// APIResponse is the generic wrapper for Value Domain API responses.
type APIResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

// Domain represents a domain in the Value Domain account.
type Domain struct {
	ID         int    `json:"id"`
	DomainName string `json:"domainname"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	ExpiresAt  string `json:"expirationdate"`
	AutoRenew  bool   `json:"auto_renew"`
	Locked     bool   `json:"locked"`
}

// DomainsResponse is the response from GET /domains.
type DomainsResponse struct {
	Results struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
	} `json:"results"`
	Paging  Paging   `json:"paging"`
	Domains []Domain `json:"domains"`
}

// Paging holds pagination info.
type Paging struct {
	TotalCount int `json:"total_count"`
	Limit      int `json:"limit"`
	Offset     int `json:"offset"`
}

// DomainDetail is the response from GET /domains/{id}.
type DomainDetail struct {
	Results struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
	} `json:"results"`
	Domain struct {
		ID             int      `json:"id"`
		DomainName     string   `json:"domainname"`
		Status         string   `json:"status"`
		ExpirationDate string   `json:"expirationdate"`
		AutoRenew      bool     `json:"auto_renew"`
		Locked         bool     `json:"locked"`
		Privacy        bool     `json:"privacy_enabled"`
		Nameservers    []string `json:"nameservers"`
		Registrant     *Contact `json:"registrant"`
	} `json:"domain"`
}

// Contact represents registrant contact info.
type Contact struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	Organization string `json:"organization"`
}

// DomainSearchResponse is the response from GET /domainsearch.
type DomainSearchResponse struct {
	Results struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
	} `json:"results"`
	Domains map[string]DomainAvailability `json:"domains"`
}

// DomainAvailability holds availability check results.
type DomainAvailability struct {
	Available bool    `json:"available"`
	Premium   bool    `json:"premium"`
	Price     float64 `json:"price"`
	Currency  string  `json:"currency"`
}

// DNSResponse is the response from GET /domains/{domain}/dns.
type DNSResponse struct {
	Results struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
	} `json:"results"`
	Records []DNSRecord `json:"records"`
}

// DNSRecord represents a single DNS record.
type DNSRecord struct {
	ID       int    `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority,omitempty"`
}

// RegisterRequest is the body for POST /domains.
type RegisterRequest struct {
	Registrar string   `json:"registrar"`
	SLD       string   `json:"sld"`
	TLD       string   `json:"tld"`
	Years     int      `json:"years"`
	WhoisProxy int    `json:"whois_proxy"`
	NS        []string `json:"ns"`
}

// NameserverUpdateRequest is the body for PUT /domains/{domain}/nameserver.
type NameserverUpdateRequest struct {
	NS []string `json:"ns"`
}

// DNSUpdateRequest is the body for PUT /domains/{domain}/dns.
type DNSUpdateRequest struct {
	Records []DNSRecord `json:"records"`
}

// RenewRequest is the body for POST /domains/{domain}/renew.
type RenewRequest struct {
	Period int `json:"period"`
}
