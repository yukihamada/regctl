import { Context, Next } from 'hono'
import { HTTPException } from 'hono/http-exception'
import { verify } from 'hono/jwt'
import type { Env, User } from '../types'

export const authenticate = (required = true) => {
  return async (c: Context<{ Bindings: Env }>, next: Next) => {
    const authorization = c.req.header('Authorization')
    
    if (!authorization) {
      if (required) {
        throw new HTTPException(401, { message: 'Authorization header required' })
      }
      return next()
    }

    const [scheme, token] = authorization.split(' ')
    
    if (scheme !== 'Bearer' || !token) {
      throw new HTTPException(401, { message: 'Invalid authorization format' })
    }

    try {
      // Check if token is in KV cache
      const cached = await c.env.SESSIONS.get(`token:${token}`)
      if (cached) {
        const session = JSON.parse(cached)
        if (new Date(session.expires_at) > new Date()) {
          c.set('user', session.user)
          return next()
        }
      }

      // Verify JWT
      const payload = await verify(token, c.env.JWT_SECRET)
      
      if (!payload.sub) {
        throw new HTTPException(401, { message: 'Invalid token' })
      }

      // Get user from database
      const result = await c.env.DB.prepare(
        'SELECT * FROM users WHERE id = ?'
      ).bind(payload.sub).first<User>()

      if (!result) {
        throw new HTTPException(401, { message: 'User not found' })
      }

      // Cache session
      await c.env.SESSIONS.put(
        `token:${token}`,
        JSON.stringify({
          user: result,
          expires_at: new Date(payload.exp! * 1000).toISOString(),
        }),
        { expirationTtl: 3600 } // 1 hour
      )

      c.set('user', result)
      await next()
    } catch (error) {
      throw new HTTPException(401, { message: 'Invalid token' })
    }
  }
}

export const authorize = (...roles: string[]) => {
  return async (c: Context, next: Next) => {
    const user = c.get('user') as User
    
    if (!user) {
      throw new HTTPException(401, { message: 'Not authenticated' })
    }

    if (roles.length > 0 && !roles.includes(user.role)) {
      throw new HTTPException(403, { message: 'Insufficient permissions' })
    }

    await next()
  }
}