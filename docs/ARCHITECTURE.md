# regctl Cloud Architecture

## Overview

regctl Cloud is a CLI-first multi-registrar API proxy SaaS built entirely on Cloudflare's edge infrastructure. It provides a unified interface for managing domains across VALUE-DOMAIN, Route 53, and Porkbun.

## Architecture Diagram

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   regctl    │     │   Web UI    │     │   Mobile    │
│    CLI      │     │  (Future)   │     │   (Future)  │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       └───────────────────┴───────────────────┘
                           │
                    ┌──────▼──────┐
                    │ Cloudflare  │
                    │   Workers   │
                    │ (API Gateway)│
                    └──────┬──────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
┌───────▼───────┐ ┌────────▼────────┐ ┌──────▼──────┐
│  Cloudflare   │ │   Cloudflare    │ │ Cloudflare  │
│      D1       │ │       KV        │ │   Durable   │
│  (Database)   │ │    (Cache)      │ │   Objects   │
└───────────────┘ └─────────────────┘ └─────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
┌───────▼───────┐ ┌────────▼────────┐ ┌──────▼──────┐
│ VALUE-DOMAIN  │ │    Route 53     │ │   Porkbun   │
│     API       │ │      API        │ │     API     │
└───────────────┘ └─────────────────┘ └─────────────┘
```

## Components

### 1. CLI (regctl)
- Written in Go with Cobra framework
- Device flow authentication
- Secure credential storage using OS keychain
- Supports all domain management operations

### 2. API Gateway (Cloudflare Workers)
- Hono.js framework for routing
- JWT-based authentication
- Rate limiting with Durable Objects
- Request/response validation with Zod

### 3. Data Storage
- **D1**: Relational data (users, domains, DNS records)
- **KV**: Session cache, temporary data
- **Durable Objects**: Rate limiting, real-time state

### 4. Provider Integration
- Unified interface for multiple registrars
- Async job processing with Queues
- Webhook notifications

## Security

1. **Authentication**
   - JWT tokens with 7-day expiry (30-day for CLI)
   - Device flow for CLI authentication
   - Password hashing with PBKDF2

2. **Authorization**
   - Role-based access control (admin, user, viewer)
   - Resource-level permissions
   - API key scoping

3. **Data Protection**
   - TLS everywhere
   - Encrypted credentials in KV
   - WHOIS privacy by default

## Scalability

- Global edge deployment on Cloudflare's network
- Auto-scaling Workers
- Geo-distributed D1 read replicas
- CDN-cached DNS responses

## Development Workflow

1. Local development with Wrangler
2. Feature branches with preview deployments
3. Automated testing in CI/CD
4. Blue-green deployments to production

## Monitoring

- Cloudflare Analytics for traffic insights
- Custom metrics in Workers
- Error tracking and alerting
- Audit logs in D1