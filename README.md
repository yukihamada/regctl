# regctl

Domain registration CLI that compares prices across registrars and auto-registers at the cheapest one.

regctl compares renewal prices for 897 TLDs across Spaceship, Porkbun, Cloudflare, and Value Domain, then registers domains at the lowest-cost registrar.

## Quick Start

```bash
curl -fsSL https://regctl.sh/install.sh | sh
regctl init
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `regctl domains check <domain>` | Compare prices across registrars |
| `regctl domains register <domain>` | Register at cheapest registrar |
| `regctl domains list` | List all domains |
| `regctl dns list <domain>` | List DNS records |
| `regctl dns add <domain> <type> <name> <content>` | Add DNS record |
| `regctl dns delete <domain> <id>` | Delete DNS record |
| `regctl billing signup` | Create billing account |
| `regctl billing topup` | Add credit |
| `regctl billing balance` | Check balance |
| `regctl config set <key> <value>` | Set API keys |
| `regctl init` | Interactive setup |

## REST API

Start the server:

```bash
regctl server --port 8080
```

### Public Endpoints (no auth)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check + endpoint list |
| GET | `/v1/domains/check/{domain}` | Check availability (free, 1000/day then $0.01) |
| POST | `/v1/domains/check` | Bulk check (max 10) |
| GET | `/v1/rdap/{domain}` | WHOIS/RDAP lookup |
| GET | `/v1/discovery` | Discovery feed of available domains |

### Authenticated Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/domains` | List domains |
| GET | `/v1/domains/{domain}` | Domain details |
| POST | `/v1/domains` | Register domain |
| POST | `/v1/domains/{domain}/renew` | Renew domain |
| PUT | `/v1/domains/{domain}/nameservers` | Update nameservers |
| GET | `/v1/dns/{domain}` | List DNS records |
| POST | `/v1/dns/{domain}` | Add DNS record |
| PUT | `/v1/dns/{domain}/{id}` | Update DNS record |
| DELETE | `/v1/dns/{domain}/{id}` | Delete DNS record |

### Billing Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/billing/signup` | Create account (no auth) |
| POST | `/v1/billing/topup` | Add credit |
| GET | `/v1/billing/balance` | Check balance |

## Discovery Feed

Domains searched via `/v1/domains/check` are logged. Available domains appear in the **discovery feed** (`GET /v1/discovery`) 24-48 hours after being searched, ranked by search popularity.

```bash
curl https://api.regctl.sh/v1/discovery?limit=50&offset=0
```

## Referral Rewards

When someone registers a domain that was first discovered by another user, the original searcher receives **10% of the registration cost** as account credit. Requirements:

- Both searcher and registrant must be authenticated (billing API key)
- The searcher must be the first authenticated user to check that domain
- Credit is applied once per domain (no duplicates)

## Rate Limits

Domain availability checks are free up to **1,000 per day** per authenticated user. Beyond that, each check costs **$0.01**. Anonymous checks are always free (no rate limit enforced via billing).

## Supported Registrars

| Registrar | Register | DNS | List | Pricing |
|-----------|----------|-----|------|---------|
| Spaceship | Yes | Yes | Yes | Yes |
| Porkbun | Yes | Yes | Yes | Yes |
| Cloudflare | No | Yes | Yes | Static |
| Value Domain | Yes | Yes | Yes | Yes |

## Price Data

All 897 TLD prices as JSON:

```bash
curl https://regctl.sh/prices.json
```

## Deployment (Fly.io)

```bash
# Create persistent volume for SQLite
fly volumes create regctl_data --region nrt --size 1

# Deploy
fly deploy
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `STRIPE_SECRET_KEY` | Stripe secret key (enables billing) |
| `REGCTL_SIGNING_SECRET` | HMAC signing secret for API keys |
| `STRIPE_WEBHOOK_SECRET` | Stripe webhook verification |
| `REGCTL_DB_PATH` | SQLite database path (default: `/data/regctl.db`) |
| `REGCTL_SUCCESS_URL` | Stripe checkout success URL |
| `REGCTL_CANCEL_URL` | Stripe checkout cancel URL |

## Links

- Website: https://regctl.sh
- GitHub: https://github.com/yukihamada/regctl
- Price Data: https://regctl.sh/prices.json
- License: MIT
