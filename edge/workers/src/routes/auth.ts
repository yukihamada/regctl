import { Hono } from 'hono'
import { zValidator } from '@hono/zod-validator'
import { z } from 'zod'
import { sign } from 'hono/jwt'
import { HTTPException } from 'hono/http-exception'
import type { Env, User } from '../types'
import { hashPassword, verifyPassword } from '../utils/crypto'
import { generateId } from '../utils/id'

export const authRouter = new Hono<{ Bindings: Env }>()

// Schemas
const loginSchema = z.object({
  email: z.string().email(),
  password: z.string().min(8),
})

const registerSchema = z.object({
  email: z.string().email(),
  password: z.string().min(8),
  name: z.string().min(1),
  organization_name: z.string().optional(),
})

const deviceCodeSchema = z.object({
  client_id: z.string(),
  scope: z.string().optional(),
})

// Login
authRouter.post('/login', zValidator('json', loginSchema), async (c) => {
  const { email, password } = c.req.valid('json')

  // Get user
  const user = await c.env.DB.prepare(
    'SELECT * FROM users WHERE email = ?'
  ).bind(email).first<User & { password_hash: string }>()

  if (!user || !(await verifyPassword(password, user.password_hash))) {
    throw new HTTPException(401, { message: 'Invalid credentials' })
  }

  // Generate JWT
  const token = await sign({
    sub: user.id,
    email: user.email,
    role: user.role,
    exp: Math.floor(Date.now() / 1000) + (7 * 24 * 60 * 60), // 7 days
  }, c.env.JWT_SECRET)

  // Store session
  await c.env.SESSIONS.put(
    `user:${user.id}:session`,
    JSON.stringify({
      token,
      user_id: user.id,
      created_at: new Date().toISOString(),
    }),
    { expirationTtl: 7 * 24 * 60 * 60 }
  )

  return c.json({
    token,
    user: {
      id: user.id,
      email: user.email,
      name: user.name,
      role: user.role,
    },
  })
})

// Register
authRouter.post('/register', zValidator('json', registerSchema), async (c) => {
  const data = c.req.valid('json')

  // Check if user exists
  const existing = await c.env.DB.prepare(
    'SELECT id FROM users WHERE email = ?'
  ).bind(data.email).first()

  if (existing) {
    throw new HTTPException(409, { message: 'User already exists' })
  }

  const userId = generateId('usr')
  const passwordHash = await hashPassword(data.password)
  const now = new Date().toISOString()

  // Create organization if provided
  let organizationId = null
  if (data.organization_name) {
    organizationId = generateId('org')
    await c.env.DB.prepare(
      'INSERT INTO organizations (id, name, user_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)'
    ).bind(organizationId, data.organization_name, userId, now, now).run()
  }

  // Create user
  await c.env.DB.prepare(
    'INSERT INTO users (id, email, name, password_hash, organization_id, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)'
  ).bind(userId, data.email, data.name, passwordHash, organizationId, 'user', now, now).run()

  // Generate JWT
  const token = await sign({
    sub: userId,
    email: data.email,
    role: 'user',
    exp: Math.floor(Date.now() / 1000) + (7 * 24 * 60 * 60),
  }, c.env.JWT_SECRET)

  return c.json({
    token,
    user: {
      id: userId,
      email: data.email,
      name: data.name,
      role: 'user',
    },
  }, 201)
})

// Device code flow for CLI
authRouter.post('/device/code', zValidator('json', deviceCodeSchema), async (c) => {
  const { client_id } = c.req.valid('json')
  
  const deviceCode = generateId('dev')
  const userCode = Math.random().toString(36).substring(2, 8).toUpperCase()
  const expiresAt = new Date(Date.now() + 15 * 60 * 1000) // 15 minutes

  // Store device code
  await c.env.SESSIONS.put(
    `device:${deviceCode}`,
    JSON.stringify({
      user_code: userCode,
      client_id,
      status: 'pending',
      expires_at: expiresAt.toISOString(),
    }),
    { expirationTtl: 900 } // 15 minutes
  )

  return c.json({
    device_code: deviceCode,
    user_code: userCode,
    verification_uri: 'https://regctl.cloud/device',
    verification_uri_complete: `https://regctl.cloud/device?code=${userCode}`,
    expires_in: 900,
    interval: 5,
  })
})

// Poll for device code
authRouter.post('/device/token', async (c) => {
  const body = await c.req.parseBody()
  const deviceCode = body.device_code as string

  if (!deviceCode) {
    throw new HTTPException(400, { message: 'device_code required' })
  }

  const session = await c.env.SESSIONS.get(`device:${deviceCode}`)
  if (!session) {
    throw new HTTPException(400, { message: 'Invalid device code' })
  }

  const data = JSON.parse(session)

  if (data.status === 'pending') {
    return c.json({ error: 'authorization_pending' }, 400)
  }

  if (data.status === 'denied') {
    return c.json({ error: 'access_denied' }, 400)
  }

  if (data.status === 'approved' && data.user_id) {
    // Get user
    const user = await c.env.DB.prepare(
      'SELECT * FROM users WHERE id = ?'
    ).bind(data.user_id).first<User>()

    if (!user) {
      throw new HTTPException(500, { message: 'User not found' })
    }

    // Generate token
    const token = await sign({
      sub: user.id,
      email: user.email,
      role: user.role,
      exp: Math.floor(Date.now() / 1000) + (30 * 24 * 60 * 60), // 30 days for CLI
    }, c.env.JWT_SECRET)

    // Delete device code
    await c.env.SESSIONS.delete(`device:${deviceCode}`)

    return c.json({
      access_token: token,
      token_type: 'Bearer',
      expires_in: 30 * 24 * 60 * 60,
    })
  }

  return c.json({ error: 'invalid_request' }, 400)
})

// Logout
authRouter.post('/logout', async (c) => {
  const authorization = c.req.header('Authorization')
  if (authorization) {
    const [, token] = authorization.split(' ')
    if (token) {
      await c.env.SESSIONS.delete(`token:${token}`)
    }
  }
  
  return c.json({ success: true })
})