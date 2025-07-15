import { Hono } from 'hono'
import type { Env } from '../types'
import { authMiddleware } from '../middleware/auth'

export const adminRouter = new Hono<{ Bindings: Env }>()

// Admin only middleware
adminRouter.use('*', authMiddleware(), async (c, next) => {
  const user = c.get('user')
  if (user?.role !== 'admin') {
    return c.json({ error: 'Forbidden' }, 403)
  }
  await next()
})

// Get system stats
adminRouter.get('/stats', async (c) => {
  const [users, domains, apiCalls] = await Promise.all([
    c.env.DB.prepare('SELECT COUNT(*) as count FROM users').first(),
    c.env.DB.prepare('SELECT COUNT(*) as count FROM domains').first(),
    c.env.DB.prepare('SELECT SUM(api_usage_count) as total FROM users').first(),
  ])
  
  return c.json({
    users: users?.count || 0,
    domains: domains?.count || 0,
    apiCalls: apiCalls?.total || 0,
  })
})

// List all users
adminRouter.get('/users', async (c) => {
  const users = await c.env.DB.prepare(
    'SELECT id, email, name, subscription_plan, created_at FROM users ORDER BY created_at DESC LIMIT 100'
  ).all()
  
  return c.json({ users: users.results })
})