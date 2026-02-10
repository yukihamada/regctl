package hosting

import "github.com/yukihamada/regctl/internal/provider"

// Provider defines a hosting service and the DNS records needed to connect a domain to it.
type Provider struct {
	Name        string
	Slug        string
	Description string
	Prompts     []string                                          // questions for user input (empty = no input needed)
	Records     func(domain string, answers []string) []provider.DNSRecord
	NextSteps   func(domain string) []string
}

// Providers lists all supported hosting providers.
var Providers = []Provider{
	{
		Name:        "Vercel",
		Slug:        "vercel",
		Description: "Frontend cloud platform (Next.js, React, etc.)",
		Records: func(domain string, answers []string) []provider.DNSRecord {
			return []provider.DNSRecord{
				{Type: "A", Name: "@", Content: "76.76.21.21", TTL: 3600},
				{Type: "CNAME", Name: "www", Content: "cname.vercel-dns.com", TTL: 3600},
			}
		},
		NextSteps: func(domain string) []string {
			return []string{
				"In Vercel dashboard, add \"" + domain + "\" as a custom domain",
				"Vercel will auto-provision SSL",
				"Verify: regctl dns list " + domain,
			}
		},
	},
	{
		Name:        "Netlify",
		Slug:        "netlify",
		Description: "Web platform for modern web projects",
		Prompts:     []string{"Netlify site name (e.g. my-site for my-site.netlify.app)"},
		Records: func(domain string, answers []string) []provider.DNSRecord {
			siteName := domain
			if len(answers) > 0 && answers[0] != "" {
				siteName = answers[0]
			}
			return []provider.DNSRecord{
				{Type: "A", Name: "@", Content: "75.2.60.5", TTL: 3600},
				{Type: "CNAME", Name: "www", Content: siteName + ".netlify.app", TTL: 3600},
			}
		},
		NextSteps: func(domain string) []string {
			return []string{
				"In Netlify dashboard → Domain settings → Add custom domain \"" + domain + "\"",
				"Netlify will auto-provision SSL",
				"Verify: regctl dns list " + domain,
			}
		},
	},
	{
		Name:        "Cloudflare Pages",
		Slug:        "cloudflare-pages",
		Description: "Full-stack application platform by Cloudflare",
		Prompts:     []string{"Cloudflare Pages project name"},
		Records: func(domain string, answers []string) []provider.DNSRecord {
			project := domain
			if len(answers) > 0 && answers[0] != "" {
				project = answers[0]
			}
			return []provider.DNSRecord{
				{Type: "CNAME", Name: "@", Content: project + ".pages.dev", TTL: 3600},
				{Type: "CNAME", Name: "www", Content: project + ".pages.dev", TTL: 3600},
			}
		},
		NextSteps: func(domain string) []string {
			return []string{
				"In Cloudflare Pages → Custom domains → Add \"" + domain + "\"",
				"SSL is provisioned automatically",
				"Verify: regctl dns list " + domain,
			}
		},
	},
	{
		Name:        "GitHub Pages",
		Slug:        "github-pages",
		Description: "Static site hosting from GitHub repositories",
		Prompts:     []string{"GitHub username (e.g. octocat for octocat.github.io)"},
		Records: func(domain string, answers []string) []provider.DNSRecord {
			username := ""
			if len(answers) > 0 {
				username = answers[0]
			}
			records := []provider.DNSRecord{
				{Type: "A", Name: "@", Content: "185.199.108.153", TTL: 3600},
				{Type: "A", Name: "@", Content: "185.199.109.153", TTL: 3600},
				{Type: "A", Name: "@", Content: "185.199.110.153", TTL: 3600},
				{Type: "A", Name: "@", Content: "185.199.111.153", TTL: 3600},
			}
			if username != "" {
				records = append(records, provider.DNSRecord{
					Type: "CNAME", Name: "www", Content: username + ".github.io", TTL: 3600,
				})
			}
			return records
		},
		NextSteps: func(domain string) []string {
			return []string{
				"In repo Settings → Pages → Custom domain → Enter \"" + domain + "\"",
				"Check \"Enforce HTTPS\" after DNS propagates",
				"Verify: regctl dns list " + domain,
			}
		},
	},
	{
		Name:        "Fly.io",
		Slug:        "flyio",
		Description: "Full-stack app hosting close to users",
		Prompts:     []string{"IPv4 address (from `fly ips list`)", "IPv6 address (from `fly ips list`, leave empty to skip)"},
		Records: func(domain string, answers []string) []provider.DNSRecord {
			var records []provider.DNSRecord
			if len(answers) > 0 && answers[0] != "" {
				records = append(records, provider.DNSRecord{
					Type: "A", Name: "@", Content: answers[0], TTL: 3600,
				})
			}
			if len(answers) > 1 && answers[1] != "" {
				records = append(records, provider.DNSRecord{
					Type: "AAAA", Name: "@", Content: answers[1], TTL: 3600,
				})
			}
			return records
		},
		NextSteps: func(domain string) []string {
			return []string{
				"Run: fly certs create " + domain,
				"Fly.io will auto-provision SSL",
				"Verify: regctl dns list " + domain,
			}
		},
	},
}

// FindBySlug returns the provider matching the given slug, or nil.
func FindBySlug(slug string) *Provider {
	for i := range Providers {
		if Providers[i].Slug == slug {
			return &Providers[i]
		}
	}
	return nil
}
