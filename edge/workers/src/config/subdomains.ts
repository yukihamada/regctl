// Subdomain configuration for regctl.com service architecture
export interface SubdomainConfig {
  name: string
  purpose: string
  target: string
  type: 'service' | 'redirect' | 'static'
  cloudflare_record?: {
    type: 'CNAME' | 'A' | 'AAAA'
    value: string
    proxied: boolean
  }
  cors_origins?: string[]
}

export const SUBDOMAIN_CONFIG: SubdomainConfig[] = [
  {
    name: 'api.regctl.com',
    purpose: 'API Gateway - Main regctl API endpoints',
    target: 'regctl-api.yukihamada.workers.dev',
    type: 'service',
    cloudflare_record: {
      type: 'CNAME',
      value: 'regctl-api.yukihamada.workers.dev',
      proxied: true
    },
    cors_origins: [
      'https://regctl.com',
      'https://www.regctl.com', 
      'https://app.regctl.com',
      'https://docs.regctl.com'
    ]
  },
  {
    name: 'app.regctl.com',
    purpose: 'Web Dashboard - React application for domain management',
    target: 'regctl-app.pages.dev',
    type: 'service',
    cloudflare_record: {
      type: 'CNAME',
      value: 'regctl-app.pages.dev',
      proxied: true
    }
  },
  {
    name: 'docs.regctl.com',
    purpose: 'Documentation - GitBook or static docs site',
    target: 'regctl-docs.pages.dev',
    type: 'service',
    cloudflare_record: {
      type: 'CNAME',
      value: 'regctl-docs.pages.dev',
      proxied: true
    }
  },
  {
    name: 'www.regctl.com',
    purpose: 'Marketing Website - Landing page and marketing content',
    target: 'regctl.com',
    type: 'redirect',
    cloudflare_record: {
      type: 'CNAME',
      value: 'regctl-site.pages.dev',
      proxied: true
    }
  },
  {
    name: 'regctl.com',
    purpose: 'Root Domain - Main website (marketing/landing)',
    target: 'regctl-site.pages.dev',
    type: 'service',
    cloudflare_record: {
      type: 'CNAME',
      value: 'regctl-site.pages.dev',
      proxied: true
    }
  },
  {
    name: 'cdn.regctl.com',
    purpose: 'CDN Assets - Static assets, images, downloads',
    target: 'regctl-cdn.r2.dev',
    type: 'static',
    cloudflare_record: {
      type: 'CNAME',
      value: 'regctl-cdn.r2.dev',
      proxied: true
    }
  },
  {
    name: 'status.regctl.com',
    purpose: 'Status Page - Service health and uptime monitoring',
    target: 'regctl-status.pages.dev',
    type: 'service',
    cloudflare_record: {
      type: 'CNAME',
      value: 'regctl-status.pages.dev',
      proxied: true
    }
  }
]

// Get subdomain configuration by name
export function getSubdomainConfig(subdomain: string): SubdomainConfig | undefined {
  return SUBDOMAIN_CONFIG.find(config => config.name === subdomain)
}

// Get all service subdomains
export function getServiceSubdomains(): SubdomainConfig[] {
  return SUBDOMAIN_CONFIG.filter(config => config.type === 'service')
}

// Generate Cloudflare DNS records for all subdomains
export function generateCloudflareRecords(): Array<{
  name: string
  type: string
  value: string
  proxied: boolean
  ttl?: number
}> {
  return SUBDOMAIN_CONFIG
    .filter(config => config.cloudflare_record)
    .map(config => ({
      name: config.name,
      type: config.cloudflare_record!.type,
      value: config.cloudflare_record!.value,
      proxied: config.cloudflare_record!.proxied,
      ttl: config.cloudflare_record!.proxied ? undefined : 300
    }))
}

// Validate subdomain configuration
export function validateSubdomainConfig(): { valid: boolean; errors: string[] } {
  const errors: string[] = []
  
  // Check for duplicate subdomain names
  const names = SUBDOMAIN_CONFIG.map(config => config.name)
  const duplicates = names.filter((name, index) => names.indexOf(name) !== index)
  if (duplicates.length > 0) {
    errors.push(`Duplicate subdomain names: ${duplicates.join(', ')}`)
  }
  
  // Check for required fields
  SUBDOMAIN_CONFIG.forEach((config, index) => {
    if (!config.name) errors.push(`Config ${index}: name is required`)
    if (!config.purpose) errors.push(`Config ${index}: purpose is required`)
    if (!config.target) errors.push(`Config ${index}: target is required`)
    if (!config.type) errors.push(`Config ${index}: type is required`)
  })
  
  return {
    valid: errors.length === 0,
    errors
  }
}