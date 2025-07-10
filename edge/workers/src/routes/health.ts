import { Hono } from 'hono'
import type { Env } from '../types'

export const healthRouter = new Hono<{ Bindings: Env }>()

healthRouter.get('/', async (c) => {
  // シンプルなヘルスチェック（データベースアクセスなし）
  const checks = {
    status: 'healthy',
    timestamp: new Date().toISOString(),
    environment: c.env?.ENVIRONMENT || 'development',
    version: '0.1.0',
  }

  return c.json(checks, 200)
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