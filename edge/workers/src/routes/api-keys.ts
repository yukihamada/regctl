import { Hono } from 'hono'
import type { Env } from '../types'
import { authMiddleware } from '../middleware/auth'
import { generateId } from '../utils/id'

export const apiKeysRouter = new Hono<{ Bindings: Env }>()

// Middleware
apiKeysRouter.use('*', authMiddleware())

// List API keys
apiKeysRouter.get('/', async (c) => {
  const userId = c.get('userId')
  
  const keys = await c.env.DB.prepare(
    'SELECT id, name, key_preview, scopes, created_at, last_used_at FROM api_keys WHERE user_id = ? AND is_active = true'
  ).bind(userId).all()
  
  return c.json({ keys: keys.results })
})

// Create API key
apiKeysRouter.post('/', async (c) => {
  const userId = c.get('userId')
  const { name, scopes } = await c.req.json()
  
  const keyId = generateId('key')
  const apiKey = generateId('sk_live')
  const keyHash = await hashApiKey(apiKey)
  const keyPreview = `${apiKey.slice(0, 8)}...${apiKey.slice(-4)}`
  
  await c.env.DB.prepare(
    `INSERT INTO api_keys (id, user_id, name, key_hash, key_preview, scopes, created_at)
     VALUES (?, ?, ?, ?, ?, ?, ?)`
  ).bind(
    keyId,
    userId,
    name,
    keyHash,
    keyPreview,
    JSON.stringify(scopes || ['read']),
    new Date().toISOString()
  ).run()
  
  return c.json({
    id: keyId,
    key: apiKey, // Only shown once
    name,
    scopes
  }, 201)
})

// Delete API key
apiKeysRouter.delete('/:id', async (c) => {
  const userId = c.get('userId')
  const keyId = c.param('id')
  
  await c.env.DB.prepare(
    'UPDATE api_keys SET is_active = false WHERE id = ? AND user_id = ?'
  ).bind(keyId, userId).run()
  
  return c.json({ success: true })
})

async function hashApiKey(key: string): Promise<string> {
  const encoder = new TextEncoder()
  const data = encoder.encode(key)
  const hash = await crypto.subtle.digest('SHA-256', data)
  return btoa(String.fromCharCode(...new Uint8Array(hash)))
}