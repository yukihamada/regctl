import { Hono } from 'hono'
import type { Env, User } from '../types'
import { authMiddleware, authorize } from '../middleware/auth'
import { 
  SUBDOMAIN_CONFIG, 
  getSubdomainConfig, 
  getServiceSubdomains,
  generateCloudflareRecords,
  validateSubdomainConfig 
} from '../config/subdomains'

export const subdomainsRouter = new Hono<{ Bindings: Env }>()

// Apply auth middleware for admin routes
subdomainsRouter.use('/admin/*', authMiddleware())
subdomainsRouter.use('/admin/*', authorize('admin'))

// Public endpoint to get subdomain configuration
subdomainsRouter.get('/', async (c) => {
  const validation = validateSubdomainConfig()
  
  return c.json({
    success: true,
    subdomains: SUBDOMAIN_CONFIG,
    validation,
    stats: {
      total: SUBDOMAIN_CONFIG.length,
      services: getServiceSubdomains().length,
      redirects: SUBDOMAIN_CONFIG.filter(s => s.type === 'redirect').length,
      static: SUBDOMAIN_CONFIG.filter(s => s.type === 'static').length
    }
  })
})

// Get specific subdomain configuration
subdomainsRouter.get('/:subdomain', async (c) => {
  const subdomain = c.req.param('subdomain')
  const config = getSubdomainConfig(subdomain)
  
  if (!config) {
    return c.json({
      success: false,
      error: 'Subdomain configuration not found',
      subdomain
    }, 404)
  }
  
  return c.json({
    success: true,
    subdomain,
    config
  })
})

// Generate Cloudflare DNS configuration
subdomainsRouter.get('/admin/cloudflare-records', async (c) => {
  const records = generateCloudflareRecords()
  
  return c.json({
    success: true,
    dns_records: records,
    total_records: records.length,
    cloudflare_api_commands: records.map(record => ({
      description: `Create DNS record for ${record.name}`,
      curl_command: `curl -X POST "https://api.cloudflare.com/client/v4/zones/{zone_id}/dns_records" \\
  -H "Authorization: Bearer {cloudflare_api_token}" \\
  -H "Content-Type: application/json" \\
  --data '${JSON.stringify({
    type: record.type,
    name: record.name,
    content: record.value,
    proxied: record.proxied,
    ttl: record.ttl || 1
  })}'`
    }))
  })
})

// Health check for all subdomains
subdomainsRouter.get('/admin/health-check', async (c) => {
  const results = []
  
  for (const config of getServiceSubdomains()) {
    try {
      const startTime = Date.now()
      const response = await fetch(`https://${config.target}`, {
        method: 'HEAD',
        signal: AbortSignal.timeout(5000) // 5 second timeout
      })
      const responseTime = Date.now() - startTime
      
      results.push({
        subdomain: config.name,
        target: config.target,
        status: response.status,
        ok: response.ok,
        response_time_ms: responseTime,
        headers: Object.fromEntries(response.headers.entries())
      })
    } catch (error) {
      results.push({
        subdomain: config.name,
        target: config.target,
        status: 0,
        ok: false,
        error: error instanceof Error ? error.message : 'Unknown error',
        response_time_ms: null
      })
    }
  }
  
  const healthyCount = results.filter(r => r.ok).length
  const totalCount = results.length
  
  return c.json({
    success: true,
    health_summary: {
      healthy: healthyCount,
      total: totalCount,
      percentage: totalCount > 0 ? Math.round((healthyCount / totalCount) * 100) : 0
    },
    checks: results,
    timestamp: new Date().toISOString()
  })
})

// Deploy configuration - generates deployment instructions
subdomainsRouter.get('/admin/deploy-instructions', async (c) => {
  const user = c.get('user') as User
  
  const instructions = {
    overview: 'Complete regctl.com service deployment with subdomain configuration',
    prerequisites: [
      'regctl.com domain registered with VALUE-DOMAIN',
      'Cloudflare account with API token',
      'Cloudflare zone for regctl.com domain',
      'Cloudflare Pages projects created',
      'Worker deployed to api.regctl.com route'
    ],
    steps: [
      {
        step: 1,
        title: 'Update Cloudflare DNS Records',
        description: 'Create DNS records for all subdomains',
        actions: [
          'Use the /admin/cloudflare-records endpoint to get DNS commands',
          'Execute curl commands to create DNS records',
          'Verify DNS propagation with dig commands'
        ]
      },
      {
        step: 2,
        title: 'Deploy Worker Route Configuration',
        description: 'Update wrangler.toml with new routes',
        actions: [
          'Add api.regctl.com/* route to wrangler.toml',
          'Deploy worker with: npm run deploy',
          'Test API endpoints on new domain'
        ]
      },
      {
        step: 3,
        title: 'Deploy Pages Projects',
        description: 'Set up Cloudflare Pages for frontend applications',
        actions: [
          'Create regctl-site project for main website',
          'Create regctl-app project for dashboard',
          'Create regctl-docs project for documentation',
          'Configure custom domains in Pages dashboard'
        ]
      },
      {
        step: 4,
        title: 'Update CORS and Security',
        description: 'Configure cross-origin and security settings',
        actions: [
          'Update CORS origins in worker to include regctl.com domains',
          'Configure SSL/TLS settings in Cloudflare',
          'Set up security headers and rate limiting'
        ]
      },
      {
        step: 5,
        title: 'Test and Validate',
        description: 'Verify all services are working correctly',
        actions: [
          'Test API endpoints on api.regctl.com',
          'Verify app.regctl.com loads correctly',
          'Check docs.regctl.com accessibility',
          'Test regctl CLI with new API endpoint'
        ]
      }
    ],
    validation_endpoints: [
      'https://api.regctl.com/health',
      'https://app.regctl.com',
      'https://docs.regctl.com',
      'https://regctl.com'
    ],
    estimated_time: '30-45 minutes',
    user_info: {
      user_id: user.id,
      role: user.role,
      timestamp: new Date().toISOString()
    }
  }
  
  return c.json({
    success: true,
    deployment_instructions: instructions
  })
})