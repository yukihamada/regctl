import { Context, Next } from 'hono'
import { HTTPException } from 'hono/http-exception'
import type { Env } from '../types'

export class RateLimiter {
  state: DurableObjectState
  env: Env

  constructor(state: DurableObjectState, env: Env) {
    this.state = state
    this.env = env
  }

  async fetch(request: Request): Promise<Response> {
    const url = new URL(request.url)
    const key = url.searchParams.get('key') || 'anonymous'
    const limit = parseInt(url.searchParams.get('limit') || '100')
    const window = parseInt(url.searchParams.get('window') || '60') // seconds

    const now = Date.now()
    const windowStart = now - (window * 1000)

    // Get current count
    const requests = await this.state.storage.list<number>({
      prefix: `${key}:`,
      start: `${key}:${windowStart}`,
      end: `${key}:${now}`,
    })

    const count = requests.size

    if (count >= limit) {
      return new Response(JSON.stringify({ allowed: false, count, limit }), {
        headers: { 'Content-Type': 'application/json' },
      })
    }

    // Add new request
    await this.state.storage.put(`${key}:${now}`, 1)

    // Clean old entries
    await this.state.storage.deleteAll([...requests.keys()].filter(k => {
      const timestamp = parseInt(k.split(':')[1])
      return timestamp < windowStart
    }))

    return new Response(JSON.stringify({ allowed: true, count: count + 1, limit }), {
      headers: { 'Content-Type': 'application/json' },
    })
  }
}

export const rateLimitMiddleware = () => {
  return async (c: Context<{ Bindings: Env }>, next: Next) => {
    const clientIp = c.req.header('CF-Connecting-IP') || 'unknown'
    const auth = c.req.header('Authorization')
    const key = auth ? `auth:${auth.substring(0, 16)}` : `ip:${clientIp}`

    const id = c.env.RATE_LIMITER.idFromName(key)
    const stub = c.env.RATE_LIMITER.get(id)
    
    const response = await stub.fetch(
      `https://rate-limiter.internal/?key=${key}&limit=100&window=60`
    )
    
    const result = await response.json() as { allowed: boolean, count: number, limit: number }
    
    if (!result.allowed) {
      throw new HTTPException(429, {
        message: 'Too many requests',
      })
    }

    c.header('X-RateLimit-Limit', result.limit.toString())
    c.header('X-RateLimit-Remaining', (result.limit - result.count).toString())
    
    await next()
  }
}