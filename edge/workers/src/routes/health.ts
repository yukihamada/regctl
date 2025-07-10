import { Hono } from 'hono'
import type { Env } from '../types'

export const healthRouter = new Hono<{ Bindings: Env }>()

healthRouter.get('/', async (c) => {
  const checks = {
    status: 'healthy',
    timestamp: new Date().toISOString(),
    environment: c.env.ENVIRONMENT,
    checks: {
      database: 'unknown',
      kv: 'unknown',
      queues: 'unknown',
    },
  }

  try {
    // Check D1 Database
    const dbResult = await c.env.DB.prepare('SELECT 1').first()
    checks.checks.database = dbResult ? 'healthy' : 'unhealthy'
  } catch (error) {
    checks.checks.database = 'unhealthy'
    checks.status = 'degraded'
  }

  try {
    // Check KV
    await c.env.CACHE.put('health-check', Date.now().toString(), { expirationTtl: 10 })
    const kvResult = await c.env.CACHE.get('health-check')
    checks.checks.kv = kvResult ? 'healthy' : 'unhealthy'
  } catch (error) {
    checks.checks.kv = 'unhealthy'
    checks.status = 'degraded'
  }

  const statusCode = checks.status === 'healthy' ? 200 : 503
  return c.json(checks, statusCode)
})

healthRouter.get('/ready', (c) => {
  return c.json({
    ready: true,
    timestamp: new Date().toISOString(),
  })
})

healthRouter.get('/live', (c) => {
  return c.json({
    alive: true,
    timestamp: new Date().toISOString(),
  })
})