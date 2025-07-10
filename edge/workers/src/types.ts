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
  owner_id: string
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