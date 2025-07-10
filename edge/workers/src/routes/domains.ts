import { Hono } from 'hono'
import { zValidator } from '@hono/zod-validator'
import { z } from 'zod'
import { authenticate, authorize } from '../middleware/auth'
import type { Env, Domain, User } from '../types'
import { ValueDomainProvider } from '../services/providers/value-domain'
import { Route53Provider } from '../services/providers/route53'
import { PorkbunProvider } from '../services/providers/porkbun'

export const domainsRouter = new Hono<{ Bindings: Env }>()

// Apply auth middleware
domainsRouter.use('*', authenticate())

// Schemas
const createDomainSchema = z.object({
  name: z.string().regex(/^[a-z0-9]+([\-\.]{1}[a-z0-9]+)*\.[a-z]{2,}$/),
  registrar: z.enum(['value-domain', 'route53', 'porkbun']),
  auto_renew: z.boolean().default(true),
  privacy_enabled: z.boolean().default(true),
})

const transferDomainSchema = z.object({
  from_registrar: z.enum(['value-domain', 'route53', 'porkbun']),
  to_registrar: z.enum(['value-domain', 'route53', 'porkbun']),
  auth_code: z.string().optional(),
})

// List domains
domainsRouter.get('/', async (c) => {
  const user = c.get('user') as User
  const { page = '1', limit = '20', registrar, status } = c.req.query()

  let query = 'SELECT * FROM domains WHERE owner_id = ?'
  const params: any[] = [user.id]

  if (registrar) {
    query += ' AND registrar = ?'
    params.push(registrar)
  }

  if (status) {
    query += ' AND status = ?'
    params.push(status)
  }

  query += ' ORDER BY created_at DESC LIMIT ? OFFSET ?'
  params.push(parseInt(limit), (parseInt(page) - 1) * parseInt(limit))

  const { results } = await c.env.DB.prepare(query).bind(...params).all<Domain>()

  // Get total count
  let countQuery = 'SELECT COUNT(*) as total FROM domains WHERE owner_id = ?'
  const countParams: any[] = [user.id]
  
  if (registrar) {
    countQuery += ' AND registrar = ?'
    countParams.push(registrar)
  }
  
  if (status) {
    countQuery += ' AND status = ?'
    countParams.push(status)
  }

  const { total } = await c.env.DB.prepare(countQuery).bind(...countParams).first<{ total: number }>() || { total: 0 }

  return c.json({
    domains: results,
    pagination: {
      page: parseInt(page),
      limit: parseInt(limit),
      total,
      pages: Math.ceil(total / parseInt(limit)),
    },
  })
})

// Get domain details
domainsRouter.get('/:domain', async (c) => {
  const user = c.get('user') as User
  const domainName = c.req.param('domain')

  const domain = await c.env.DB.prepare(
    'SELECT * FROM domains WHERE name = ? AND owner_id = ?'
  ).bind(domainName, user.id).first<Domain>()

  if (!domain) {
    return c.json({ error: 'Domain not found' }, 404)
  }

  // Get additional info from provider
  try {
    const provider = getProvider(domain.registrar, c.env)
    const details = await provider.getDomainInfo(domain.name)
    
    return c.json({
      ...domain,
      provider_details: details,
    })
  } catch (error) {
    return c.json(domain)
  }
})

// Register new domain
domainsRouter.post('/', zValidator('json', createDomainSchema), async (c) => {
  const user = c.get('user') as User
  const data = c.req.valid('json')

  // Check if domain is available
  const provider = getProvider(data.registrar, c.env)
  const availability = await provider.checkAvailability(data.name)

  if (!availability.available) {
    return c.json({ error: 'Domain not available' }, 400)
  }

  // Register domain with provider
  try {
    const result = await provider.registerDomain(data.name, {
      auto_renew: data.auto_renew,
      privacy_enabled: data.privacy_enabled,
      years: 1,
    })

    // Save to database
    const now = new Date().toISOString()
    await c.env.DB.prepare(
      `INSERT INTO domains (
        id, name, registrar, status, owner_id, expires_at, 
        auto_renew, locked, privacy_enabled, nameservers, 
        created_at, updated_at
      ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
    ).bind(
      result.id,
      data.name,
      data.registrar,
      'active',
      user.id,
      result.expires_at,
      data.auto_renew,
      false,
      data.privacy_enabled,
      JSON.stringify(result.nameservers),
      now,
      now
    ).run()

    // Send webhook
    await c.env.WEBHOOKS.send({
      type: 'domain.registered',
      data: {
        domain: data.name,
        registrar: data.registrar,
        user_id: user.id,
      },
    })

    return c.json(result, 201)
  } catch (error) {
    console.error('Domain registration failed:', error)
    return c.json({ error: 'Failed to register domain' }, 500)
  }
})

// Transfer domain
domainsRouter.post('/:domain/transfer', 
  zValidator('json', transferDomainSchema),
  authorize('admin'),
  async (c) => {
    const user = c.get('user') as User
    const domainName = c.req.param('domain')
    const data = c.req.valid('json')

    // Get domain
    const domain = await c.env.DB.prepare(
      'SELECT * FROM domains WHERE name = ? AND owner_id = ?'
    ).bind(domainName, user.id).first<Domain>()

    if (!domain) {
      return c.json({ error: 'Domain not found' }, 404)
    }

    if (domain.registrar === data.to_registrar) {
      return c.json({ error: 'Domain already with target registrar' }, 400)
    }

    // Initiate transfer
    try {
      const fromProvider = getProvider(data.from_registrar, c.env)
      const toProvider = getProvider(data.to_registrar, c.env)

      // Get auth code if not provided
      let authCode = data.auth_code
      if (!authCode) {
        authCode = await fromProvider.getAuthCode(domain.name)
      }

      // Start transfer
      const result = await toProvider.transferDomain(domain.name, authCode)

      // Update database
      await c.env.DB.prepare(
        'UPDATE domains SET status = ?, updated_at = ? WHERE id = ?'
      ).bind('transferring', new Date().toISOString(), domain.id).run()

      // Send webhook
      await c.env.WEBHOOKS.send({
        type: 'domain.transfer.initiated',
        data: {
          domain: domain.name,
          from: data.from_registrar,
          to: data.to_registrar,
          user_id: user.id,
        },
      })

      return c.json({
        success: true,
        transfer_id: result.transfer_id,
        status: 'pending',
      })
    } catch (error) {
      console.error('Domain transfer failed:', error)
      return c.json({ error: 'Failed to initiate transfer' }, 500)
    }
  }
)

// Update domain settings
domainsRouter.patch('/:domain', async (c) => {
  const user = c.get('user') as User
  const domainName = c.req.param('domain')
  const body = await c.req.json()

  const domain = await c.env.DB.prepare(
    'SELECT * FROM domains WHERE name = ? AND owner_id = ?'
  ).bind(domainName, user.id).first<Domain>()

  if (!domain) {
    return c.json({ error: 'Domain not found' }, 404)
  }

  // Update allowed fields
  const updates: string[] = []
  const values: any[] = []

  if ('auto_renew' in body) {
    updates.push('auto_renew = ?')
    values.push(body.auto_renew)
  }

  if ('privacy_enabled' in body) {
    updates.push('privacy_enabled = ?')
    values.push(body.privacy_enabled)
  }

  if ('nameservers' in body && Array.isArray(body.nameservers)) {
    updates.push('nameservers = ?')
    values.push(JSON.stringify(body.nameservers))
    
    // Update nameservers with provider
    const provider = getProvider(domain.registrar, c.env)
    await provider.updateNameservers(domain.name, body.nameservers)
  }

  if (updates.length > 0) {
    updates.push('updated_at = ?')
    values.push(new Date().toISOString())
    values.push(domain.id)

    await c.env.DB.prepare(
      `UPDATE domains SET ${updates.join(', ')} WHERE id = ?`
    ).bind(...values).run()
  }

  return c.json({ success: true })
})

// Delete domain
domainsRouter.delete('/:domain', authorize('admin'), async (c) => {
  const user = c.get('user') as User
  const domainName = c.req.param('domain')

  const domain = await c.env.DB.prepare(
    'SELECT * FROM domains WHERE name = ? AND owner_id = ?'
  ).bind(domainName, user.id).first<Domain>()

  if (!domain) {
    return c.json({ error: 'Domain not found' }, 404)
  }

  // Mark as deleted (soft delete)
  await c.env.DB.prepare(
    'UPDATE domains SET status = ?, updated_at = ? WHERE id = ?'
  ).bind('deleted', new Date().toISOString(), domain.id).run()

  return c.json({ success: true })
})

// Helper function to get provider instance
function getProvider(registrar: string, env: Env) {
  switch (registrar) {
    case 'value-domain':
      return new ValueDomainProvider(env.VALUE_DOMAIN_API_KEY)
    case 'route53':
      return new Route53Provider(env.AWS_ACCESS_KEY_ID, env.AWS_SECRET_ACCESS_KEY)
    case 'porkbun':
      return new PorkbunProvider(env.PORKBUN_API_KEY, env.PORKBUN_API_SECRET)
    default:
      throw new Error(`Unknown registrar: ${registrar}`)
  }
}