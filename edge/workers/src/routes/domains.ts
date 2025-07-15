import { Hono } from 'hono'
import { zValidator } from '@hono/zod-validator'
import { z } from 'zod'
import { authMiddleware, authorize } from '../middleware/auth'
import type { Env, Domain, User } from '../types'
import { ValueDomainProvider } from '../services/providers/value-domain'
import { Route53Provider } from '../services/providers/route53'
import { PorkbunProvider } from '../services/providers/porkbun'

export const domainsRouter = new Hono<{ Bindings: Env }>()

// Apply auth middleware
domainsRouter.use('*', authMiddleware())

// Schemas
const createDomainSchema = z.object({
  name: z.string().regex(/^[a-z0-9]+([-.]{1}[a-z0-9]+)*\.[a-z]{2,}$/),
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
  const { page = '1', limit = '20', registrar, status, sync } = c.req.query()

  // If sync=true, sync domains from providers first
  if (sync === 'true') {
    await syncDomainsFromProviders(user, c.env)
  }

  let query = 'SELECT * FROM domains WHERE user_id = ?'
  const params: (string | number)[] = [user.id]

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

  const { results } = await c.env.DB.prepare(query).bind(...params).all()

  // Convert database results to proper types
  const domains = results.map((result: Record<string, unknown>) => ({
    ...result,
    auto_renew: Boolean(result.auto_renew),
    privacy_enabled: Boolean(result.privacy_enabled),
    locked: Boolean(result.locked),
    nameservers: result.nameservers ? JSON.parse(result.nameservers as string) : []
  }))

  // Get total count
  let countQuery = 'SELECT COUNT(*) as total FROM domains WHERE user_id = ?'
  const countParams: (string | number)[] = [user.id]
  
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
    domains,
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
  const { refresh } = c.req.query()

  const domain = await c.env.DB.prepare(
    'SELECT * FROM domains WHERE domain = ? AND user_id = ?'
  ).bind(domainName, user.id).first<Domain>()

  if (!domain) {
    return c.json({ error: 'Domain not found' }, 404)
  }

  // Get additional info from provider
  let providerDetails = null
  let dnsRecords = null
  
  try {
    const provider = getProvider(domain.registrar, c.env)
    
    // Get fresh data from provider if refresh=true or if last update was > 1 hour ago
    const shouldRefresh = refresh === 'true' || 
      (new Date().getTime() - new Date(domain.updated_at).getTime()) > 3600000

    if (shouldRefresh) {
      providerDetails = await provider.getDomainInfo(domain.name)
      
      // Update local database with fresh info
      const now = new Date().toISOString()
      await c.env.DB.prepare(
        `UPDATE domains SET 
          status = ?, expires_at = ?, auto_renew = ?, locked = ?, 
          privacy_enabled = ?, nameservers = ?, updated_at = ?
        WHERE id = ?`
      ).bind(
        providerDetails.status,
        providerDetails.expires_at,
        providerDetails.auto_renew,
        providerDetails.locked,
        providerDetails.privacy_enabled,
        JSON.stringify(providerDetails.nameservers),
        now,
        domain.id
      ).run()

      // Get DNS records if it's a VALUE-DOMAIN domain
      if (domain.registrar === 'value-domain' && provider instanceof ValueDomainProvider) {
        try {
          dnsRecords = await provider.getDNSRecords(domain.name)
        } catch (error) {
          // DNS records might not be available
          console.error('Failed to fetch DNS records:', error)
        }
      }
    }
    
    return c.json({
      ...domain,
      auto_renew: Boolean(domain.auto_renew),
      privacy_enabled: Boolean(domain.privacy_enabled),
      locked: Boolean(domain.locked),
      nameservers: domain.nameservers ? JSON.parse(domain.nameservers) : [],
      provider_details: providerDetails,
      dns_records: dnsRecords,
      last_refreshed: shouldRefresh ? new Date().toISOString() : domain.updated_at
    })
  } catch (error) {
    if (c.env.ENVIRONMENT === 'development') {
      console.error('Failed to get provider details:', error)
    }
    return c.json({
      ...domain,
      auto_renew: Boolean(domain.auto_renew),
      privacy_enabled: Boolean(domain.privacy_enabled),
      locked: Boolean(domain.locked),
      nameservers: domain.nameservers ? JSON.parse(domain.nameservers) : [],
      provider_details: null,
      dns_records: null,
      error: 'Failed to fetch provider details'
    })
  }
})

// Register new domain
domainsRouter.post('/', zValidator('json', createDomainSchema), async (c) => {
  const user = c.get('user') as User
  const data = c.req.valid('json')

  // Check if domain is available first
  const provider = getProvider(data.registrar, c.env)
  
  try {
    const availability = await provider.checkAvailability(data.name)

    if (!availability.available) {
      return c.json({ 
        error: 'Domain not available',
        details: availability
      }, 400)
    }

    // For regctl.com specifically, log the registration attempt
    if (data.name === 'regctl.com') {
      console.log(`Attempting to register regctl.com via ${data.registrar}`)
      console.log(`Price: ${availability.price}`)
    }

    // Register domain with provider
    const result = await provider.registerDomain(data.name, {
      auto_renew: data.auto_renew,
      privacy_enabled: data.privacy_enabled,
      years: 1,
    })

    // Generate unique domain ID
    const domainId = `dom_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`

    // Save to database
    const now = new Date().toISOString()
    await c.env.DB.prepare(
      `INSERT INTO domains (
        id, domain, registrar, status, user_id, expires_at, 
        auto_renew, locked, privacy_enabled, nameservers, 
        created_at, updated_at
      ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
    ).bind(
      domainId,
      data.name,
      data.registrar,
      'active',
      user.id,
      result.expires_at,
      data.auto_renew,
      false,
      data.privacy_enabled,
      JSON.stringify(result.nameservers || ['ns1.value-domain.com', 'ns2.value-domain.com']),
      now,
      now
    ).run()

    // Send webhook notification
    try {
      await c.env.WEBHOOKS.send({
        type: 'domain.registered',
        data: {
          domain: data.name,
          registrar: data.registrar,
          user_id: user.id,
          price: availability.price,
        },
      })
    } catch (webhookError) {
      console.error('Webhook failed:', webhookError)
      // Don't fail the registration if webhook fails
    }

    return c.json({
      success: true,
      domain: data.name,
      registrar: data.registrar,
      status: 'active',
      expires_at: result.expires_at,
      price: availability.price,
      registration_id: domainId
    }, 201)
  } catch (error) {
    if (c.env.ENVIRONMENT === 'development') {
      console.error('Domain registration failed:', error)
    }
    
    // Return more detailed error information
    const errorMessage = error instanceof Error ? error.message : 'Unknown error'
    return c.json({ 
      error: 'Failed to register domain',
      details: errorMessage,
      domain: data.name,
      registrar: data.registrar
    }, 500)
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
      'SELECT * FROM domains WHERE domain = ? AND user_id = ?'
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
      if (c.env.ENVIRONMENT === 'development') {
        console.error('Domain transfer failed:', error)
      }
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
    'SELECT * FROM domains WHERE domain = ? AND user_id = ?'
  ).bind(domainName, user.id).first<Domain>()

  if (!domain) {
    return c.json({ error: 'Domain not found' }, 404)
  }

  // Update allowed fields
  const updates: string[] = []
  const values: (string | boolean | null)[] = []

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
    'SELECT * FROM domains WHERE domain = ? AND user_id = ?'
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

// Register regctl.com with service setup
domainsRouter.post('/register-regctl', async (c) => {
  const user = c.get('user') as User
  
  // Only allow admin users to register regctl.com
  if (user.role !== 'admin') {
    return c.json({ error: 'Admin access required' }, 403)
  }

  const domainName = 'regctl.com'
  const registrar = 'value-domain'

  try {
    // Check availability first
    const provider = getProvider(registrar, c.env)
    const availability = await provider.checkAvailability(domainName)

    if (!availability.available) {
      return c.json({ 
        error: 'regctl.com is not available',
        details: availability
      }, 400)
    }

    console.log(`Registering regctl.com via VALUE-DOMAIN at price: ${availability.price}`)

    // Register the domain
    const result = await provider.registerDomain(domainName, {
      auto_renew: true,
      privacy_enabled: false, // Don't hide regctl.com ownership
      years: 1,
    })

    // Generate unique domain ID
    const domainId = `dom_regctl_${Date.now()}`

    // Save to database
    const now = new Date().toISOString()
    await c.env.DB.prepare(
      `INSERT INTO domains (
        id, domain, registrar, status, user_id, expires_at, 
        auto_renew, locked, privacy_enabled, nameservers, 
        created_at, updated_at
      ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
    ).bind(
      domainId,
      domainName,
      registrar,
      'active',
      user.id,
      result.expires_at,
      true,
      false,
      false,
      JSON.stringify(['ns1.value-domain.com', 'ns2.value-domain.com']),
      now,
      now
    ).run()

    // Prepare subdomain configuration for next step
    const subdomains = [
      { name: 'api.regctl.com', purpose: 'API Gateway' },
      { name: 'app.regctl.com', purpose: 'Web Dashboard' },
      { name: 'docs.regctl.com', purpose: 'Documentation' },
      { name: 'www.regctl.com', purpose: 'Website' }
    ]

    return c.json({
      success: true,
      domain: domainName,
      registrar: registrar,
      status: 'active',
      expires_at: result.expires_at,
      price: availability.price,
      registration_id: domainId,
      subdomains: subdomains,
      next_steps: [
        'Configure Cloudflare DNS',
        'Set up subdomain routing',
        'Deploy services to subdomains'
      ]
    }, 201)

  } catch (error) {
    console.error('regctl.com registration failed:', error)
    const errorMessage = error instanceof Error ? error.message : 'Unknown error'
    return c.json({ 
      error: 'Failed to register regctl.com',
      details: errorMessage
    }, 500)
  }
})

// Sync domains endpoint
domainsRouter.post('/sync', async (c) => {
  const user = c.get('user') as User
  
  try {
    const syncResults = await syncDomainsFromProviders(user, c.env)
    return c.json({
      success: true,
      synchronized: syncResults.total,
      details: syncResults.details
    })
  } catch (error) {
    if (c.env.ENVIRONMENT === 'development') {
      console.error('Domain sync failed:', error)
    }
    return c.json({ error: 'Failed to sync domains' }, 500)
  }
})

// Helper function to sync domains from all providers
async function syncDomainsFromProviders(user: User, env: Env) {
  const syncResults = {
    total: 0,
    details: {
      'value-domain': 0,
      'route53': 0,
      'porkbun': 0
    }
  }

  // Get user's provider API keys from user settings or organization settings
  // For now, we'll use the global environment variables
  const providers = [
    { name: 'value-domain', key: env.VALUE_DOMAIN_API_KEY },
    { name: 'route53', key: env.AWS_ACCESS_KEY_ID },
    { name: 'porkbun', key: env.PORKBUN_API_KEY }
  ]

  for (const providerInfo of providers) {
    if (!providerInfo.key) continue

    try {
      const provider = getProvider(providerInfo.name, env)
      
      // Special handling for VALUE-DOMAIN which has listDomains method
      if (providerInfo.name === 'value-domain' && provider instanceof ValueDomainProvider) {
        const remoteDomains = await provider.listDomains(100, 0)
        
        for (const remoteDomain of remoteDomains) {
          const domainName = remoteDomain.name || remoteDomain.domainname
          if (!domainName) continue

          // Check if domain already exists in our database
          const existingDomain = await env.DB.prepare(
            'SELECT id FROM domains WHERE domain = ? AND user_id = ?'
          ).bind(domainName, user.id).first<{ id: string }>()

          if (!existingDomain) {
            // Add new domain to database
            const now = new Date().toISOString()
            const domainId = `dom_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
            
            await env.DB.prepare(
              `INSERT INTO domains (
                id, name, registrar, status, user_id, expires_at, 
                auto_renew, locked, privacy_enabled, nameservers, 
                created_at, updated_at
              ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
            ).bind(
              domainId,
              domainName,
              'value-domain',
              remoteDomain.status || 'active',
              user.id,
              remoteDomain.expires_at || new Date(Date.now() + 365 * 24 * 60 * 60 * 1000).toISOString(),
              remoteDomain.auto_renew || false,
              remoteDomain.locked || false,
              remoteDomain.privacy_enabled || false,
              JSON.stringify(remoteDomain.nameservers || ['ns1.value-domain.com', 'ns2.value-domain.com']),
              now,
              now
            ).run()

            syncResults.details['value-domain']++
            syncResults.total++
          }
        }
      }
      
      // TODO: Implement Route53 and Porkbun domain listing
      // These providers may not have direct domain listing APIs
      // or may require different approaches
      
    } catch (error) {
      console.error(`Failed to sync from ${providerInfo.name}:`, error)
    }
  }

  return syncResults
}

// Helper function to get provider instance
function getProvider(registrar: string, env: Env) {
  switch (registrar) {
    case 'value-domain':
      return new ValueDomainProvider(env.VALUE_DOMAIN_API_KEY)
    case 'route53':
      return new Route53Provider(env.AWS_ACCESS_KEY_ID, env.AWS_SECRET_ACCESS_KEY, env)
    case 'porkbun':
      return new PorkbunProvider(env.PORKBUN_API_KEY, env.PORKBUN_API_SECRET)
    default:
      throw new Error(`Unknown registrar: ${registrar}`)
  }
}