import { Hono } from 'hono'
import { zValidator } from '@hono/zod-validator'
import { z } from 'zod'
import { authMiddleware } from '../middleware/auth'
import type { Env, Domain, DNSRecord, User, ParsedZoneRecord } from '../types'
import { generateId } from '../utils/id'
import { ValueDomainProvider } from '../services/providers/value-domain'
import { Route53Provider } from '../services/providers/route53'
import { PorkbunProvider } from '../services/providers/porkbun'

export const dnsRouter = new Hono<{ Bindings: Env }>()

// Apply auth middleware
dnsRouter.use('*', authMiddleware())

// Schemas
const createRecordSchema = z.object({
  type: z.enum(['A', 'AAAA', 'CNAME', 'MX', 'TXT', 'NS', 'SOA', 'SRV', 'CAA']),
  name: z.string(),
  content: z.string(),
  ttl: z.number().min(60).max(86400).default(3600),
  priority: z.number().optional(),
  proxied: z.boolean().default(false),
})

const updateRecordSchema = z.object({
  content: z.string().optional(),
  ttl: z.number().min(60).max(86400).optional(),
  priority: z.number().optional(),
  proxied: z.boolean().optional(),
})

// List DNS records for a domain
dnsRouter.get('/:domain/records', async (c) => {
  const user = c.get('user') as User
  const domainName = c.req.param('domain')
  const { type, sync } = c.req.query()

  // Check domain ownership
  const domain = await c.env.DB.prepare(
    'SELECT * FROM domains WHERE domain = ? AND user_id = ?'
  ).bind(domainName, user.id).first<Domain>()

  if (!domain) {
    return c.json({ error: 'Domain not found' }, 404)
  }

  // Sync DNS records from provider if requested
  if (sync === 'true') {
    try {
      await syncDNSRecordsFromProvider(domain, c.env)
    } catch (error) {
      console.error('Failed to sync DNS records:', error)
    }
  }

  let query = 'SELECT * FROM dns_records WHERE domain_id = ?'
  const params: (string | number)[] = [domain.id]

  if (type) {
    query += ' AND type = ?'
    params.push(type)
  }

  query += ' ORDER BY type, name'

  const { results } = await c.env.DB.prepare(query).bind(...params).all<DNSRecord>()

  return c.json({
    records: results,
    domain: domainName,
  })
})

// Get DNS record details
dnsRouter.get('/:domain/records/:recordId', async (c) => {
  const user = c.get('user') as User
  const domainName = c.req.param('domain')
  const recordId = c.req.param('recordId')

  // Check domain ownership
  const domain = await c.env.DB.prepare(
    'SELECT * FROM domains WHERE domain = ? AND user_id = ?'
  ).bind(domainName, user.id).first<Domain>()

  if (!domain) {
    return c.json({ error: 'Domain not found' }, 404)
  }

  const record = await c.env.DB.prepare(
    'SELECT * FROM dns_records WHERE id = ? AND domain_id = ?'
  ).bind(recordId, domain.id).first<DNSRecord>()

  if (!record) {
    return c.json({ error: 'Record not found' }, 404)
  }

  return c.json(record)
})

// Create DNS record
dnsRouter.post('/:domain/records', zValidator('json', createRecordSchema), async (c) => {
  const user = c.get('user') as User
  const domainName = c.req.param('domain')
  const data = c.req.valid('json')

  // Check domain ownership
  const domain = await c.env.DB.prepare(
    'SELECT * FROM domains WHERE domain = ? AND user_id = ?'
  ).bind(domainName, user.id).first<Domain>()

  if (!domain) {
    return c.json({ error: 'Domain not found' }, 404)
  }

  // Validate record type specific rules
  if (data.type === 'MX' && !data.priority) {
    return c.json({ error: 'MX records require priority' }, 400)
  }

  if (data.type === 'CNAME' && data.name === '@') {
    return c.json({ error: 'CNAME records cannot be created at root' }, 400)
  }

  // Create record
  const recordId = generateId('dns')
  const now = new Date().toISOString()

  await c.env.DB.prepare(
    `INSERT INTO dns_records (
      id, domain_id, type, name, content, ttl, priority, proxied, created_at, updated_at
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
  ).bind(
    recordId,
    domain.id,
    data.type,
    data.name,
    data.content,
    data.ttl,
    data.priority || null,
    data.proxied,
    now,
    now
  ).run()

  // Use Cloudflare API to create the actual DNS record
  try {
    await createCloudflareRecord(c.env, domainName, {
      type: data.type,
      name: data.name === '@' ? domainName : `${data.name}.${domainName}`,
      content: data.content,
      ttl: data.ttl,
      priority: data.priority,
      proxied: data.proxied,
    })
  } catch (error) {
    // Rollback database change
    await c.env.DB.prepare('DELETE FROM dns_records WHERE id = ?').bind(recordId).run()
    console.error('Failed to create DNS record:', error)
    return c.json({ error: 'Failed to create DNS record' }, 500)
  }

  return c.json({
    id: recordId,
    domain_id: domain.id,
    type: data.type,
    name: data.name,
    content: data.content,
    ttl: data.ttl,
    priority: data.priority,
    proxied: data.proxied,
    created_at: now,
    updated_at: now,
  }, 201)
})

// Update DNS record
dnsRouter.patch('/:domain/records/:recordId', zValidator('json', updateRecordSchema), async (c) => {
  const user = c.get('user') as User
  const domainName = c.req.param('domain')
  const recordId = c.req.param('recordId')
  const data = c.req.valid('json')

  // Check domain ownership
  const domain = await c.env.DB.prepare(
    'SELECT * FROM domains WHERE domain = ? AND user_id = ?'
  ).bind(domainName, user.id).first<Domain>()

  if (!domain) {
    return c.json({ error: 'Domain not found' }, 404)
  }

  const record = await c.env.DB.prepare(
    'SELECT * FROM dns_records WHERE id = ? AND domain_id = ?'
  ).bind(recordId, domain.id).first<DNSRecord>()

  if (!record) {
    return c.json({ error: 'Record not found' }, 404)
  }

  // Update record
  const updates: string[] = []
  const values: (string | number | null)[] = []

  if (data.content !== undefined) {
    updates.push('content = ?')
    values.push(data.content)
  }

  if (data.ttl !== undefined) {
    updates.push('ttl = ?')
    values.push(data.ttl)
  }

  if (data.priority !== undefined) {
    updates.push('priority = ?')
    values.push(data.priority)
  }

  if (data.proxied !== undefined) {
    updates.push('proxied = ?')
    values.push(data.proxied)
  }

  if (updates.length > 0) {
    updates.push('updated_at = ?')
    values.push(new Date().toISOString())
    values.push(recordId)

    await c.env.DB.prepare(
      `UPDATE dns_records SET ${updates.join(', ')} WHERE id = ?`
    ).bind(...values).run()

    // Update Cloudflare record
    try {
      await updateCloudflareRecord(c.env, domainName, record.id, {
        content: data.content || record.content,
        ttl: data.ttl || record.ttl,
        priority: data.priority !== undefined ? data.priority : record.priority,
        proxied: data.proxied !== undefined ? data.proxied : record.proxied,
      })
    } catch (error) {
      console.error('Failed to update DNS record:', error)
      return c.json({ error: 'Failed to update DNS record' }, 500)
    }
  }

  return c.json({ success: true })
})

// Delete DNS record
dnsRouter.delete('/:domain/records/:recordId', async (c) => {
  const user = c.get('user') as User
  const domainName = c.req.param('domain')
  const recordId = c.req.param('recordId')

  // Check domain ownership
  const domain = await c.env.DB.prepare(
    'SELECT * FROM domains WHERE domain = ? AND user_id = ?'
  ).bind(domainName, user.id).first<Domain>()

  if (!domain) {
    return c.json({ error: 'Domain not found' }, 404)
  }

  const record = await c.env.DB.prepare(
    'SELECT * FROM dns_records WHERE id = ? AND domain_id = ?'
  ).bind(recordId, domain.id).first<DNSRecord>()

  if (!record) {
    return c.json({ error: 'Record not found' }, 404)
  }

  // Delete from database
  await c.env.DB.prepare('DELETE FROM dns_records WHERE id = ?').bind(recordId).run()

  // Delete from Cloudflare
  try {
    await deleteCloudflareRecord(c.env, domainName, record.id)
  } catch (error) {
    console.error('Failed to delete DNS record:', error)
  }

  return c.json({ success: true })
})

// Import DNS records from zone file
dnsRouter.post('/:domain/import', async (c) => {
  const user = c.get('user') as User
  const domainName = c.req.param('domain')
  const body = await c.req.parseBody()
  const zoneFile = body.zone_file as string

  if (!zoneFile) {
    return c.json({ error: 'Zone file required' }, 400)
  }

  // Check domain ownership
  const domain = await c.env.DB.prepare(
    'SELECT * FROM domains WHERE domain = ? AND user_id = ?'
  ).bind(domainName, user.id).first<Domain>()

  if (!domain) {
    return c.json({ error: 'Domain not found' }, 404)
  }

  // Parse zone file (simplified parser)
  const records = parseZoneFile(zoneFile, domainName)
  const imported: DNSRecord[] = []

  for (const record of records) {
    try {
      const recordId = generateId('dns')
      const now = new Date().toISOString()

      await c.env.DB.prepare(
        `INSERT INTO dns_records (
          id, domain_id, type, name, content, ttl, priority, proxied, created_at, updated_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
      ).bind(
        recordId,
        domain.id,
        record.type,
        record.name,
        record.content,
        record.ttl,
        record.priority || null,
        false,
        now,
        now
      ).run()

      imported.push({
        id: recordId,
        ...record,
      })
    } catch (error) {
      console.error('Failed to import record:', error)
    }
  }

  return c.json({
    imported: imported.length,
    records: imported,
  })
})

// Cloudflare API helpers
async function createCloudflareRecord(_env: Env, _domain: string, _record: Partial<DNSRecord>) {
  // This would integrate with Cloudflare's API
  // For now, it's a placeholder
  // Creating Cloudflare record: { domain, record }
}

async function updateCloudflareRecord(_env: Env, _domain: string, _recordId: string, _updates: Partial<DNSRecord>) {
  // Updating Cloudflare record: { domain, recordId, updates }
}

async function deleteCloudflareRecord(_env: Env, _domain: string, _recordId: string) {
  // Deleting Cloudflare record: { domain, recordId }
}

// Helper function to sync DNS records from provider
async function syncDNSRecordsFromProvider(domain: Domain, env: Env): Promise<void> {
  if (domain.registrar !== 'value-domain') {
    // Only VALUE-DOMAIN is supported for now
    return
  }

  const provider = getProvider(domain.registrar, env)
  if (!(provider instanceof ValueDomainProvider)) return

  try {
    const providerRecords = await provider.getDNSRecords(domain.name)
    
    // Clear existing records in database
    await env.DB.prepare('DELETE FROM dns_records WHERE domain_id = ?')
      .bind(domain.id).run()

    // Insert new records from provider
    for (const record of providerRecords) {
      const recordId = generateId('dns')
      const now = new Date().toISOString()

      await env.DB.prepare(
        `INSERT INTO dns_records (
          id, domain_id, type, name, content, ttl, priority, proxied, created_at, updated_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
      ).bind(
        recordId,
        domain.id,
        record.type,
        record.name,
        record.content,
        record.ttl || 3600,
        record.priority || null,
        record.proxied || false,
        now,
        now
      ).run()
    }
  } catch (error) {
    console.error('Failed to sync DNS records from provider:', error)
    throw error
  }
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

// Simple zone file parser
function parseZoneFile(zoneFile: string, domain: string): ParsedZoneRecord[] {
  const records: ParsedZoneRecord[] = []
  const lines = zoneFile.split('\n')

  for (const line of lines) {
    const trimmed = line.trim()
    if (!trimmed || trimmed.startsWith(';')) continue

    const parts = trimmed.split(/\s+/)
    if (parts.length < 4) continue

    const [name, ttl, , type, ...content] = parts

    records.push({
      name: name === '@' ? '@' : name.replace(`.${domain}`, ''),
      ttl: parseInt(ttl),
      type,
      content: content.join(' '),
    })
  }

  return records
}