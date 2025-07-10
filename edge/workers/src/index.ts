import { Hono } from 'hono'
import { cors } from 'hono/cors'
import { logger } from 'hono/logger'
import { timing } from 'hono/timing'
import { authRouter } from './routes/auth'
import { domainsRouter } from './routes/domains'
import { dnsRouter } from './routes/dns'
import { healthRouter } from './routes/health'
import { errorHandler } from './middleware/error'
import { rateLimiter } from './middleware/rate-limit'
import type { Env } from './types'

const app = new Hono<{ Bindings: Env }>()

// Middleware
app.use('*', cors({
  origin: ['https://regctl.cloud', 'http://localhost:3000'],
  credentials: true,
}))
app.use('*', logger())
app.use('*', timing())
app.use('/api/*', rateLimiter())
app.onError(errorHandler)

// Routes
app.route('/health', healthRouter)
app.route('/api/v1/auth', authRouter)
app.route('/api/v1/domains', domainsRouter)
app.route('/api/v1/dns', dnsRouter)

// 404 handler
app.notFound((c) => {
  return c.json({ error: 'Not found' }, 404)
})

export default app