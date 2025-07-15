import { Hono } from 'hono'
import type { Env } from '../types'
import { authMiddleware } from '../middleware/auth'
import { generateId } from '../utils/id'

export const webhooksRouter = new Hono<{ Bindings: Env }>()

// Middleware
webhooksRouter.use('*', authMiddleware())

// List webhooks
webhooksRouter.get('/', async (c) => {
  const userId = c.get('userId')
  
  const webhooks = await c.env.DB.prepare(
    'SELECT * FROM webhooks WHERE user_id = ? AND is_active = true'
  ).bind(userId).all()
  
  return c.json({ webhooks: webhooks.results })
})

// Create webhook
webhooksRouter.post('/', async (c) => {
  const userId = c.get('userId')
  const { url, events } = await c.req.json()
  
  const webhookId = generateId('whk')
  const secret = crypto.randomUUID()
  
  await c.env.DB.prepare(
    `INSERT INTO webhooks (id, user_id, url, events, secret, created_at, updated_at)
     VALUES (?, ?, ?, ?, ?, ?, ?)`
  ).bind(
    webhookId,
    userId,
    url,
    JSON.stringify(events || ['*']),
    secret,
    new Date().toISOString(),
    new Date().toISOString()
  ).run()
  
  return c.json({
    id: webhookId,
    url,
    events,
    secret
  }, 201)
})

// Delete webhook
webhooksRouter.delete('/:id', async (c) => {
  const userId = c.get('userId')
  const webhookId = c.param('id')
  
  await c.env.DB.prepare(
    'UPDATE webhooks SET is_active = false WHERE id = ? AND user_id = ?'
  ).bind(webhookId, userId).run()
  
  return c.json({ success: true })
})