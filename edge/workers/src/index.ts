import { Hono } from 'hono'
import { cors } from 'hono/cors'
import { logger } from 'hono/logger'
import { prettyJSON } from 'hono/pretty-json'
import { secureHeaders } from 'hono/secure-headers'
import { timing } from 'hono/timing'

// Import routers
import { authRouter } from './routes/auth'
import { domainsRouter } from './routes/domains'
import { dnsRouter } from './routes/dns'
import { healthRouter } from './routes/health'
import { usersRouter } from './routes/users'
import { billingRouter } from './routes/billing'
import { apiKeysRouter } from './routes/api-keys'
import { webhooksRouter } from './routes/webhooks'
import { adminRouter } from './routes/admin'
import { testValueDomainRouter } from './routes/test-value-domain'
import { subdomainsRouter } from './routes/subdomains'

// Import middleware
import { errorHandler } from './middleware/error'
import { rateLimitMiddleware } from './middleware/rate-limit'

// Import types
import type { Env, WebhookPayload, WebhookRecord } from './types'

// Create main app
const app = new Hono<{ Bindings: Env }>()

// Global middleware
app.use('*', logger())
app.use('*', timing())
app.use('*', secureHeaders())
app.use('*', prettyJSON())

// CORS configuration
app.use('*', cors({
  origin: (origin) => {
    // Allow requests from our domains
    const allowedOrigins = [
      // Primary regctl.com domains
      'https://regctl.com',
      'https://www.regctl.com',
      'https://app.regctl.com',
      'https://docs.regctl.com',
      'https://api.regctl.com',
      // Legacy regctl.cloud domains (for transition)
      'https://regctl.cloud',
      'https://app.regctl.cloud',
      'https://staging.regctl.cloud',
      // Development
      'http://localhost:3000',
      'http://localhost:5173',
      'http://localhost:8080'
    ]
    
    // Allow all Cloudflare Pages deployments
    if (origin && (
      origin.match(/^https:\/\/[\w-]+\.regctl-app\.pages\.dev$/) ||
      origin.match(/^https:\/\/[\w-]+\.regctl-site\.pages\.dev$/) ||
      origin.match(/^https:\/\/[\w-]+\.regctl-docs\.pages\.dev$/)
    )) {
      return origin
    }
    
    return allowedOrigins.includes(origin) ? origin : 'https://regctl.com'
  },
  allowHeaders: ['Content-Type', 'Authorization', 'X-API-Key'],
  allowMethods: ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS'],
  exposeHeaders: ['X-Request-Id', 'X-RateLimit-Limit', 'X-RateLimit-Remaining'],
  credentials: true,
  maxAge: 86400
}))

// Rate limiting for API routes
app.use('/api/*', rateLimitMiddleware())

// API versioning
const v1 = new Hono<{ Bindings: Env }>()

// Mount routers
v1.route('/auth', authRouter)
v1.route('/domains', domainsRouter)
v1.route('/dns', dnsRouter)
v1.route('/health', healthRouter)
v1.route('/users', usersRouter)
v1.route('/billing', billingRouter)
v1.route('/api-keys', apiKeysRouter)
v1.route('/webhooks', webhooksRouter)
v1.route('/admin', adminRouter)
v1.route('/test/value-domain', testValueDomainRouter)
v1.route('/subdomains', subdomainsRouter)

// API base routes
app.route('/api/v1', v1)
app.route('/v1', v1) // Alternative path

// Root endpoint
app.get('/', (c) => {
  return c.json({
    name: 'regctl API',
    version: '1.0.0',
    documentation: 'https://docs.regctl.cloud',
    endpoints: {
      auth: '/api/v1/auth',
      domains: '/api/v1/domains',
      dns: '/api/v1/dns',
      health: '/api/v1/health'
    }
  })
})

// 404 handler
app.notFound((c) => {
  return c.json({
    error: 'Not Found',
    message: `The requested endpoint ${c.req.path} does not exist`,
    documentation: 'https://docs.regctl.cloud'
  }, 404)
})

// Global error handler
app.onError(errorHandler)

// Durable Objects
export class RateLimiter {
  private state: DurableObjectState
  private env: Env
  private requests: Map<string, number[]> = new Map()

  constructor(state: DurableObjectState, env: Env) {
    this.state = state
    this.env = env
  }

  async fetch(request: Request): Promise<Response> {
    const url = new URL(request.url)
    const key = url.searchParams.get('key') || 'anonymous'
    const limit = parseInt(url.searchParams.get('limit') || '100')
    const window = parseInt(url.searchParams.get('window') || '3600') // 1 hour in seconds

    // Clean old entries
    await this.cleanup()

    // Get current requests for this key
    const now = Date.now()
    const windowStart = now - (window * 1000)
    const requests = this.requests.get(key) || []
    const recentRequests = requests.filter(time => time > windowStart)

    // Check if limit exceeded
    if (recentRequests.length >= limit) {
      const resetTime = Math.min(...recentRequests) + (window * 1000)
      return new Response(JSON.stringify({
        allowed: false,
        limit,
        remaining: 0,
        reset: new Date(resetTime).toISOString()
      }), {
        headers: { 'Content-Type': 'application/json' }
      })
    }

    // Add current request
    recentRequests.push(now)
    this.requests.set(key, recentRequests)

    // Store state
    await this.state.storage.put('requests', Object.fromEntries(this.requests))

    return new Response(JSON.stringify({
      allowed: true,
      limit,
      remaining: limit - recentRequests.length,
      reset: new Date(windowStart + (window * 1000)).toISOString()
    }), {
      headers: { 'Content-Type': 'application/json' }
    })
  }

  async cleanup() {
    const now = Date.now()
    const maxAge = 24 * 60 * 60 * 1000 // 24 hours

    for (const [key, requests] of this.requests.entries()) {
      const filtered = requests.filter(time => time > now - maxAge)
      if (filtered.length === 0) {
        this.requests.delete(key)
      } else {
        this.requests.set(key, filtered)
      }
    }
  }
}

// Scheduled handler for cron jobs
export async function scheduled(
  event: ScheduledEvent,
  env: Env,
  _ctx: ExecutionContext
): Promise<void> {
  switch (event.cron) {
    case '0 0 * * *': // Daily at midnight
      // Daily cleanup tasks
      await cleanupExpiredSessions(env)
      await checkExpiringDomains(env)
      break
    case '0 * * * *': // Every hour
      // Hourly tasks
      await processWebhookQueue(env)
      break
  }
}

// Helper functions for scheduled tasks
async function cleanupExpiredSessions(_env: Env) {
  // Clean up expired sessions from KV
  // Implementation here
}

async function checkExpiringDomains(env: Env) {
  // Check for domains expiring soon and send notifications
  const thirtyDaysFromNow = new Date()
  thirtyDaysFromNow.setDate(thirtyDaysFromNow.getDate() + 30)
  
  await env.DB.prepare(
    'SELECT * FROM domains WHERE expires_at <= ? AND auto_renew = false'
  ).bind(thirtyDaysFromNow.toISOString()).all()

  // Found expiring domains
  // Send notifications
}

async function processWebhookQueue(_env: Env) {
  // Process webhook queue
  // Implementation here
}

// Queue handler for webhooks
export async function queue(
  batch: MessageBatch,
  env: Env,
  _ctx: ExecutionContext
): Promise<void> {
  for (const message of batch.messages) {
    try {
      const data = message.body as WebhookPayload
      // Processing webhook: data.type
      
      // Get webhooks for this event type
      const webhooks = await env.DB.prepare(
        'SELECT * FROM webhooks WHERE active = true AND events LIKE ?'
      ).bind(`%${data.type}%`).all()

      // Send webhooks
      for (const webhook of webhooks.results) {
        await sendWebhook(webhook, data, env)
      }

      // Acknowledge message
      message.ack()
    } catch (error) {
      console.error('Failed to process webhook:', error)
      message.retry()
    }
  }
}

async function sendWebhook(webhook: WebhookRecord, data: WebhookPayload, _env: Env) {
  // Send webhook with retries
  const payload = {
    id: crypto.randomUUID(),
    type: data.type,
    data: data.data,
    created_at: new Date().toISOString()
  }

  const signature = await generateWebhookSignature(payload, webhook.secret)

  const response = await fetch(webhook.url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Webhook-Signature': signature,
      'X-Webhook-ID': payload.id
    },
    body: JSON.stringify(payload)
  })

  if (!response.ok) {
    throw new Error(`Webhook failed: ${response.status}`)
  }
}

async function generateWebhookSignature(payload: WebhookPayload, secret: string): Promise<string> {
  const encoder = new TextEncoder()
  const data = encoder.encode(JSON.stringify(payload))
  const key = encoder.encode(secret)
  
  const cryptoKey = await crypto.subtle.importKey(
    'raw',
    key,
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign']
  )
  
  const signature = await crypto.subtle.sign('HMAC', cryptoKey, data)
  return btoa(String.fromCharCode(...new Uint8Array(signature)))
}

export default app