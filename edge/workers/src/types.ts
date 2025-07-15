export interface Env {
  // KV Namespaces
  SESSIONS: KVNamespace
  CACHE: KVNamespace
  
  // D1 Database
  DB: D1Database
  
  // Durable Objects
  RATE_LIMITER: DurableObjectNamespace
  
  // Queues
  WEBHOOKS: Queue
  
  // Environment Variables
  ENVIRONMENT: string
  JWT_SECRET: string
  STRIPE_SECRET_KEY: string
  STRIPE_WEBHOOK_SECRET: string
  VALUE_DOMAIN_API_KEY: string
  AWS_ACCESS_KEY_ID: string
  AWS_SECRET_ACCESS_KEY: string
  PORKBUN_API_KEY: string
  PORKBUN_API_SECRET: string
}

export interface User {
  id: string
  email: string
  name: string
  organization_id?: string
  role: 'admin' | 'user' | 'viewer'
  created_at: string
  updated_at: string
}

export interface Domain {
  id: string
  name: string
  registrar: 'value-domain' | 'route53' | 'porkbun'
  status: 'active' | 'pending' | 'expired' | 'transferring'
  user_id: string
  expires_at: string
  auto_renew: boolean
  locked: boolean
  privacy_enabled: boolean
  nameservers: string[]
  created_at: string
  updated_at: string
}

export interface DNSRecord {
  id: string
  domain_id: string
  type: 'A' | 'AAAA' | 'CNAME' | 'MX' | 'TXT' | 'NS' | 'SOA' | 'SRV' | 'CAA'
  name: string
  content: string
  ttl: number
  priority?: number
  proxied?: boolean
  created_at: string
  updated_at: string
}

export interface Session {
  id: string
  user_id: string
  token: string
  expires_at: string
  created_at: string
}

// Webhook types
export interface WebhookPayload {
  id: string
  type: string
  data: Record<string, unknown>
  created_at: string
}

export interface WebhookRecord {
  id: string
  url: string
  secret: string
  events: string
  active: boolean
}

// Stripe types
export interface StripeCheckoutSession {
  id: string
  url: string
  customer: string
  mode: string
  payment_method_types: string[]
  line_items: Array<{
    price: string
    quantity: number
  }>
  success_url: string
  cancel_url: string
  metadata: Record<string, string>
}

export interface StripeSubscription {
  id: string
  customer: string
  status: string
  current_period_end: number
  cancel_at_period_end: boolean
  data: Array<{
    id: string
  }>
}

export interface StripeInvoice {
  id: string
  number: string
  amount_paid: number
  currency: string
  status: string
  invoice_pdf: string
  created: number
  period_start: number
  period_end: number
}

// Provider response types
export interface ProviderDomainResponse {
  domains: Domain[]
  nextPage?: string
}

export interface ProviderDNSResponse {
  records: DNSRecord[]
  nextPage?: string
}

// Zone file types
export interface ParsedZoneRecord {
  name: string
  ttl: number
  type: string
  content: string
  priority?: number
}