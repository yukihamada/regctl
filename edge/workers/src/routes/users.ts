import { Hono } from 'hono'
import { z } from 'zod'
import { nanoid } from 'nanoid'
import bcrypt from 'bcryptjs'
import type { Env } from '../types'
import { authMiddleware } from '../middleware/auth'
import { generateId } from '../utils/id'

export const usersRouter = new Hono<{ Bindings: Env }>()

// Schemas
const registerSchema = z.object({
  email: z.string().email(),
  password: z.string().min(8),
  name: z.string().optional(),
})

const updateProfileSchema = z.object({
  name: z.string().optional(),
  email: z.string().email().optional(),
})

const changePasswordSchema = z.object({
  currentPassword: z.string(),
  newPassword: z.string().min(8),
})

// Register new user
usersRouter.post('/register', async (c) => {
  try {
    const body = await c.req.json()
    const { email, password, name } = registerSchema.parse(body)

    // Check if user already exists
    const existingUser = await c.env.DB.prepare(
      'SELECT id FROM users WHERE email = ?'
    ).bind(email).first()

    if (existingUser) {
      return c.json({ error: 'Email already registered' }, 409)
    }

    // Hash password
    const passwordHash = await bcrypt.hash(password, 10)

    // Create user
    const userId = generateId('usr')
    const now = new Date().toISOString()

    await c.env.DB.prepare(
      `INSERT INTO users (id, email, password_hash, name, created_at, updated_at)
       VALUES (?, ?, ?, ?, ?, ?)`
    ).bind(userId, email, passwordHash, name || null, now, now).run()

    // Generate verification token
    const verificationToken = nanoid(32)
    const expiresAt = new Date()
    expiresAt.setHours(expiresAt.getHours() + 24)

    await c.env.DB.prepare(
      `INSERT INTO email_verifications (token, user_id, expires_at)
       VALUES (?, ?, ?)`
    ).bind(verificationToken, userId, expiresAt.toISOString()).run()

    // TODO: Send verification email

    return c.json({
      id: userId,
      email,
      name,
      emailVerified: false,
      message: 'Registration successful. Please check your email to verify your account.',
    }, 201)
  } catch (error) {
    if (error instanceof z.ZodError) {
      return c.json({ error: 'Invalid input', details: error.errors }, 400)
    }
    throw error
  }
})

// Get current user profile
usersRouter.get('/me', authMiddleware(), async (c) => {
  const userId = c.get('userId')

  const user = await c.env.DB.prepare(
    `SELECT id, email, name, email_verified, subscription_status, subscription_plan,
            api_usage_count, api_usage_reset_at, created_at
     FROM users WHERE id = ?`
  ).bind(userId).first()

  if (!user) {
    return c.json({ error: 'User not found' }, 404)
  }

  return c.json(user)
})

// Update user profile
usersRouter.patch('/me', authMiddleware(), async (c) => {
  try {
    const userId = c.get('userId')
    const body = await c.req.json()
    const updates = updateProfileSchema.parse(body)

    const updateFields = []
    const values = []

    if (updates.name !== undefined) {
      updateFields.push('name = ?')
      values.push(updates.name)
    }

    if (updates.email !== undefined) {
      // Check if email is already taken
      const existingUser = await c.env.DB.prepare(
        'SELECT id FROM users WHERE email = ? AND id != ?'
      ).bind(updates.email, userId).first()

      if (existingUser) {
        return c.json({ error: 'Email already in use' }, 409)
      }

      updateFields.push('email = ?')
      updateFields.push('email_verified = ?')
      values.push(updates.email, false)
    }

    if (updateFields.length === 0) {
      return c.json({ error: 'No fields to update' }, 400)
    }

    updateFields.push('updated_at = ?')
    values.push(new Date().toISOString())
    values.push(userId)

    await c.env.DB.prepare(
      `UPDATE users SET ${updateFields.join(', ')} WHERE id = ?`
    ).bind(...values).run()

    return c.json({ message: 'Profile updated successfully' })
  } catch (error) {
    if (error instanceof z.ZodError) {
      return c.json({ error: 'Invalid input', details: error.errors }, 400)
    }
    throw error
  }
})

// Change password
usersRouter.post('/me/change-password', authMiddleware(), async (c) => {
  try {
    const userId = c.get('userId')
    const body = await c.req.json()
    const { currentPassword, newPassword } = changePasswordSchema.parse(body)

    // Get current password hash
    const user = await c.env.DB.prepare(
      'SELECT password_hash FROM users WHERE id = ?'
    ).bind(userId).first()

    if (!user) {
      return c.json({ error: 'User not found' }, 404)
    }

    // Verify current password
    const isValid = await bcrypt.compare(currentPassword, user.password_hash)
    if (!isValid) {
      return c.json({ error: 'Current password is incorrect' }, 401)
    }

    // Hash new password
    const newPasswordHash = await bcrypt.hash(newPassword, 10)

    // Update password
    await c.env.DB.prepare(
      'UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?'
    ).bind(newPasswordHash, new Date().toISOString(), userId).run()

    return c.json({ message: 'Password changed successfully' })
  } catch (error) {
    if (error instanceof z.ZodError) {
      return c.json({ error: 'Invalid input', details: error.errors }, 400)
    }
    throw error
  }
})

// Delete account
usersRouter.delete('/me', authMiddleware(), async (c) => {
  const userId = c.get('userId')

  // Start transaction
  const batch = []

  // Delete user data in correct order
  batch.push(c.env.DB.prepare('DELETE FROM webhook_deliveries WHERE webhook_id IN (SELECT id FROM webhooks WHERE user_id = ?)').bind(userId))
  batch.push(c.env.DB.prepare('DELETE FROM webhooks WHERE user_id = ?').bind(userId))
  batch.push(c.env.DB.prepare('DELETE FROM dns_records WHERE domain_id IN (SELECT id FROM domains WHERE user_id = ?)').bind(userId))
  batch.push(c.env.DB.prepare('DELETE FROM domains WHERE user_id = ?').bind(userId))
  batch.push(c.env.DB.prepare('DELETE FROM api_keys WHERE user_id = ?').bind(userId))
  batch.push(c.env.DB.prepare('DELETE FROM audit_logs WHERE user_id = ?').bind(userId))
  batch.push(c.env.DB.prepare('DELETE FROM email_verifications WHERE user_id = ?').bind(userId))
  batch.push(c.env.DB.prepare('DELETE FROM password_resets WHERE user_id = ?').bind(userId))
  batch.push(c.env.DB.prepare('DELETE FROM organization_members WHERE user_id = ?').bind(userId))
  batch.push(c.env.DB.prepare('DELETE FROM organizations WHERE user_id = ?').bind(userId))
  batch.push(c.env.DB.prepare('DELETE FROM users WHERE id = ?').bind(userId))

  await c.env.DB.batch(batch)

  return c.json({ message: 'Account deleted successfully' })
})

// Verify email
usersRouter.post('/verify-email/:token', async (c) => {
  const token = c.param('token')

  // Get verification record
  const verification = await c.env.DB.prepare(
    'SELECT user_id, expires_at FROM email_verifications WHERE token = ?'
  ).bind(token).first()

  if (!verification) {
    return c.json({ error: 'Invalid verification token' }, 400)
  }

  // Check if expired
  if (new Date(verification.expires_at) < new Date()) {
    return c.json({ error: 'Verification token expired' }, 400)
  }

  // Update user
  await c.env.DB.prepare(
    'UPDATE users SET email_verified = true, updated_at = ? WHERE id = ?'
  ).bind(new Date().toISOString(), verification.user_id).run()

  // Delete verification token
  await c.env.DB.prepare(
    'DELETE FROM email_verifications WHERE token = ?'
  ).bind(token).run()

  return c.json({ message: 'Email verified successfully' })
})

// Request password reset
usersRouter.post('/reset-password', async (c) => {
  try {
    const body = await c.req.json()
    const { email } = z.object({ email: z.string().email() }).parse(body)

    // Get user
    const user = await c.env.DB.prepare(
      'SELECT id FROM users WHERE email = ?'
    ).bind(email).first()

    if (!user) {
      // Don't reveal if user exists
      return c.json({ message: 'If an account exists with this email, you will receive a password reset link.' })
    }

    // Generate reset token
    const resetToken = nanoid(32)
    const expiresAt = new Date()
    expiresAt.setHours(expiresAt.getHours() + 1)

    await c.env.DB.prepare(
      `INSERT INTO password_resets (token, user_id, expires_at)
       VALUES (?, ?, ?)`
    ).bind(resetToken, user.id, expiresAt.toISOString()).run()

    // TODO: Send reset email

    return c.json({ message: 'If an account exists with this email, you will receive a password reset link.' })
  } catch (error) {
    if (error instanceof z.ZodError) {
      return c.json({ error: 'Invalid input', details: error.errors }, 400)
    }
    throw error
  }
})

// Reset password with token
usersRouter.post('/reset-password/:token', async (c) => {
  try {
    const token = c.param('token')
    const body = await c.req.json()
    const { password } = z.object({ password: z.string().min(8) }).parse(body)

    // Get reset record
    const reset = await c.env.DB.prepare(
      'SELECT user_id, expires_at FROM password_resets WHERE token = ?'
    ).bind(token).first()

    if (!reset) {
      return c.json({ error: 'Invalid reset token' }, 400)
    }

    // Check if expired
    if (new Date(reset.expires_at) < new Date()) {
      return c.json({ error: 'Reset token expired' }, 400)
    }

    // Hash new password
    const passwordHash = await bcrypt.hash(password, 10)

    // Update password
    await c.env.DB.prepare(
      'UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?'
    ).bind(passwordHash, new Date().toISOString(), reset.user_id).run()

    // Delete reset token
    await c.env.DB.prepare(
      'DELETE FROM password_resets WHERE token = ?'
    ).bind(token).run()

    return c.json({ message: 'Password reset successfully' })
  } catch (error) {
    if (error instanceof z.ZodError) {
      return c.json({ error: 'Invalid input', details: error.errors }, 400)
    }
    throw error
  }
})