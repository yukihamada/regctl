# regctl Cloud API Documentation

## Base URL

```
https://api.regctl.cloud
```

## Authentication

All API endpoints (except auth endpoints) require authentication using a Bearer token:

```
Authorization: Bearer <token>
```

## Endpoints

### Authentication

#### Login with Email/Password
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123"
}
```

Response:
```json
{
  "token": "eyJ...",
  "user": {
    "id": "usr_123",
    "email": "user@example.com",
    "name": "John Doe",
    "role": "user"
  }
}
```

#### Device Flow - Start
```http
POST /api/v1/auth/device/code
Content-Type: application/json

{
  "client_id": "regctl-cli",
  "scope": "full"
}
```

Response:
```json
{
  "device_code": "dev_abc123",
  "user_code": "ABCD-1234",
  "verification_uri": "https://regctl.cloud/device",
  "verification_uri_complete": "https://regctl.cloud/device?code=ABCD-1234",
  "expires_in": 900,
  "interval": 5
}
```

#### Device Flow - Poll for Token
```http
POST /api/v1/auth/device/token
Content-Type: application/x-www-form-urlencoded

device_code=dev_abc123&grant_type=urn:ietf:params:oauth:grant-type:device_code
```

### Domains

#### List Domains
```http
GET /api/v1/domains?registrar=value-domain&status=active&page=1&limit=20
```

Response:
```json
{
  "domains": [
    {
      "id": "dom_123",
      "name": "example.com",
      "registrar": "value-domain",
      "status": "active",
      "expires_at": "2025-12-31T23:59:59Z",
      "auto_renew": true,
      "privacy_enabled": true,
      "nameservers": ["ns1.example.com", "ns2.example.com"]
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 42,
    "pages": 3
  }
}
```

#### Get Domain Details
```http
GET /api/v1/domains/example.com
```

#### Register Domain
```http
POST /api/v1/domains
Content-Type: application/json

{
  "name": "newdomain.com",
  "registrar": "porkbun",
  "years": 1,
  "auto_renew": true,
  "privacy_enabled": true
}
```

#### Transfer Domain
```http
POST /api/v1/domains/example.com/transfer
Content-Type: application/json

{
  "from_registrar": "value-domain",
  "to_registrar": "route53",
  "auth_code": "ABC123"
}
```

#### Update Domain
```http
PATCH /api/v1/domains/example.com
Content-Type: application/json

{
  "auto_renew": false,
  "nameservers": ["ns1.custom.com", "ns2.custom.com"]
}
```

### DNS Records

#### List DNS Records
```http
GET /api/v1/dns/example.com/records?type=A
```

Response:
```json
{
  "records": [
    {
      "id": "dns_123",
      "type": "A",
      "name": "@",
      "content": "192.0.2.1",
      "ttl": 3600,
      "proxied": true
    }
  ],
  "domain": "example.com"
}
```

#### Create DNS Record
```http
POST /api/v1/dns/example.com/records
Content-Type: application/json

{
  "type": "A",
  "name": "www",
  "content": "192.0.2.1",
  "ttl": 3600,
  "proxied": true
}
```

#### Update DNS Record
```http
PATCH /api/v1/dns/example.com/records/dns_123
Content-Type: application/json

{
  "content": "192.0.2.2",
  "ttl": 1800
}
```

#### Delete DNS Record
```http
DELETE /api/v1/dns/example.com/records/dns_123
```

#### Import Zone File
```http
POST /api/v1/dns/example.com/import
Content-Type: application/json

{
  "zone_file": "$ORIGIN example.com.\n@ IN A 192.0.2.1\nwww IN CNAME @"
}
```

## Error Responses

All errors follow this format:
```json
{
  "error": "Error message",
  "status": 400,
  "details": {
    "field": "Additional context"
  }
}
```

Common status codes:
- `400` - Bad Request
- `401` - Unauthorized
- `403` - Forbidden
- `404` - Not Found
- `429` - Too Many Requests
- `500` - Internal Server Error

## Rate Limiting

API requests are rate limited to:
- 100 requests per minute for authenticated users
- 10 requests per minute for unauthenticated users

Rate limit headers:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1640995200
```