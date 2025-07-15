import { Hono } from 'hono'
import type { Env } from '../types'
import { ValueDomainProvider } from '../services/providers/value-domain'

export const testValueDomainRouter = new Hono<{ Bindings: Env }>()

// Test VALUE-DOMAIN API connectivity and explore endpoints
testValueDomainRouter.get('/connectivity', async (c) => {
  try {
    if (!c.env.VALUE_DOMAIN_API_KEY) {
      return c.json({
        success: false,
        error: 'VALUE_DOMAIN_API_KEY not configured'
      }, 400)
    }

    const provider = new ValueDomainProvider(c.env.VALUE_DOMAIN_API_KEY)
    
    // Test with a simple domain availability check
    const result = await provider.checkAvailability('test-regctl-example.com')
    
    return c.json({
      success: true,
      api_key_configured: true,
      api_key_length: c.env.VALUE_DOMAIN_API_KEY.length,
      api_key_prefix: c.env.VALUE_DOMAIN_API_KEY.substring(0, 10) + '...',
      test_domain: 'test-regctl-example.com',
      test_result: result
    })
  } catch (error) {
    return c.json({
      success: false,
      error: error instanceof Error ? error.message : 'Unknown error',
      api_key_configured: !!c.env.VALUE_DOMAIN_API_KEY
    }, 500)
  }
})

// Test VALUE-DOMAIN API endpoints directly to understand limitations
testValueDomainRouter.get('/api-exploration', async (c) => {
  if (!c.env.VALUE_DOMAIN_API_KEY) {
    return c.json({
      success: false,
      error: 'VALUE_DOMAIN_API_KEY not configured'
    }, 400)
  }

  const baseUrl = 'https://api.value-domain.com/v1'
  const headers = {
    'Authorization': `Bearer ${c.env.VALUE_DOMAIN_API_KEY}`,
    'Content-Type': 'application/json'
  }

  const tests = []

  // Test 1: Get domains list
  try {
    const response = await fetch(`${baseUrl}/domains`, { headers })
    const data = await response.text()
    tests.push({
      endpoint: 'GET /domains',
      status: response.status,
      success: response.ok,
      response: data.substring(0, 500) + (data.length > 500 ? '...' : '')
    })
  } catch (error) {
    tests.push({
      endpoint: 'GET /domains',
      success: false,
      error: error instanceof Error ? error.message : 'Unknown error'
    })
  }

  // Test 2: Try domain search endpoint
  try {
    const response = await fetch(`${baseUrl}/domainsearch?domainnames=regctl.com`, { headers })
    const data = await response.text()
    tests.push({
      endpoint: 'GET /domainsearch',
      status: response.status,
      success: response.ok,
      response: data.substring(0, 500) + (data.length > 500 ? '...' : '')
    })
  } catch (error) {
    tests.push({
      endpoint: 'GET /domainsearch',
      success: false,
      error: error instanceof Error ? error.message : 'Unknown error'
    })
  }

  // Test 3: Try to get TLD info
  try {
    const response = await fetch(`${baseUrl}/tlds`, { headers })
    const data = await response.text()
    tests.push({
      endpoint: 'GET /tlds',
      status: response.status,
      success: response.ok,
      response: data.substring(0, 500) + (data.length > 500 ? '...' : '')
    })
  } catch (error) {
    tests.push({
      endpoint: 'GET /tlds',
      success: false,
      error: error instanceof Error ? error.message : 'Unknown error'
    })
  }

  // Test 4: Check if registration endpoint exists without actually registering
  try {
    const response = await fetch(`${baseUrl}/domains`, {
      method: 'POST',
      headers,
      body: JSON.stringify({
        domainname: 'regctl.com',
        action: 'check' // Try with check action instead of actual registration
      })
    })
    const data = await response.text()
    tests.push({
      endpoint: 'POST /domains (check)',
      status: response.status,
      success: response.ok,
      response: data.substring(0, 500) + (data.length > 500 ? '...' : '')
    })
  } catch (error) {
    tests.push({
      endpoint: 'POST /domains (check)',
      success: false,
      error: error instanceof Error ? error.message : 'Unknown error'
    })
  }

  return c.json({
    success: true,
    api_base_url: baseUrl,
    tests,
    summary: {
      total_tests: tests.length,
      successful: tests.filter(t => t.success).length,
      failed: tests.filter(t => !t.success).length
    }
  })
})

// Test domain availability check
testValueDomainRouter.get('/check/:domain', async (c) => {
  try {
    const domain = c.req.param('domain')
    
    if (!c.env.VALUE_DOMAIN_API_KEY) {
      return c.json({
        success: false,
        error: 'VALUE_DOMAIN_API_KEY not configured'
      }, 400)
    }

    const provider = new ValueDomainProvider(c.env.VALUE_DOMAIN_API_KEY)
    const result = await provider.checkAvailability(domain)
    
    return c.json({
      success: true,
      domain,
      result
    })
  } catch (error) {
    return c.json({
      success: false,
      domain: c.req.param('domain'),
      error: error instanceof Error ? error.message : 'Unknown error'
    }, 500)
  }
})

// Test domain pricing with comprehensive TLD information
testValueDomainRouter.get('/pricing/:domain', async (c) => {
  try {
    const domain = c.req.param('domain')
    
    if (!c.env.VALUE_DOMAIN_API_KEY) {
      return c.json({
        success: false,
        error: 'VALUE_DOMAIN_API_KEY not configured'
      }, 400)
    }

    const provider = new ValueDomainProvider(c.env.VALUE_DOMAIN_API_KEY)
    
    // Check availability and pricing through API
    const availability = await provider.checkAvailability(domain)
    
    // Extract TLD for pricing lookup
    const tld = domain.split('.').pop()?.toLowerCase()
    
    // Static pricing data based on documentation
    const tldPricing: Record<string, { register: number; renew: number; transfer: number; description: string }> = {
      'com': { register: 790, renew: 1950, transfer: 1950, description: '最安値・世界標準・企業サイト' },
      'org': { register: 1090, renew: 2259, transfer: 2259, description: '非営利団体・コミュニティ' },
      'net': { register: 1920, renew: 2259, transfer: 2259, description: 'ネットワーク・IT企業' },
      'jp': { register: 2035, renew: 3819, transfer: 3819, description: '日本の個人・法人・信頼性重視' },
      'cloud': { register: 230, renew: 2878, transfer: 2878, description: '新規最安値・クラウドサービス' },
      'dev': { register: 2663, renew: 2663, transfer: 2663, description: 'Google推奨・開発者ポートフォリオ' },
      'app': { register: 3335, renew: 3335, transfer: 3335, description: 'モバイル・Webアプリ' },
      'io': { register: 4820, renew: 8714, transfer: 8714, description: 'スタートアップ・API・SaaS' },
      'ai': { register: 16083, renew: 16083, transfer: 16083, description: 'AI・機械学習・フィンテック' },
      'tech': { register: 2290, renew: 4580, transfer: 4580, description: 'テクノロジー企業' },
      'online': { register: 580, renew: 3450, transfer: 3450, description: 'オンラインビジネス' },
      'store': { register: 690, renew: 5290, transfer: 5290, description: 'ECサイト・オンラインストア' },
      'info': { register: 380, renew: 2980, transfer: 2980, description: '情報サイト・ブログ' }
    }
    
    const pricing = tld ? tldPricing[tld] : null
    
    return c.json({
      success: true,
      domain,
      tld,
      availability,
      pricing: pricing ? {
        ...pricing,
        currency: 'JPY',
        prices_include_tax: true,
        price_date: '2024-07-15'
      } : null,
      comparison: pricing ? {
        vs_competitors: {
          'Route53 (.com)': { register: 1350, description: 'AWS Route 53' },
          'Porkbun (.com)': { register: 1200, description: 'Developer-friendly registrar' },
          'GoDaddy (.com)': { register: 1500, description: 'Popular commercial registrar' }
        },
        value_domain_advantage: pricing.register < 1000 ? 'Excellent value' : 
                               pricing.register < 2000 ? 'Good value' : 'Premium TLD'
      } : null,
      recommendations: {
        business: tld === 'com' ? 'Highly recommended for business' : 'Consider .com for business',
        tech: ['dev', 'io', 'tech', 'app'].includes(tld || '') ? 'Great for tech projects' : 'Consider .dev or .io for tech',
        japan: tld === 'jp' ? 'Perfect for Japan-focused business' : 'Consider .jp for Japanese market'
      }
    })
  } catch (error) {
    return c.json({
      success: false,
      domain: c.req.param('domain'),
      error: error instanceof Error ? error.message : 'Unknown error'
    }, 500)
  }
})

// Get all supported TLD pricing
testValueDomainRouter.get('/pricing-all', async (c) => {
  const allPricing = {
    'popular': {
      'com': { register: 790, renew: 1950, transfer: 1950, description: '最安値・世界標準・企業サイト', category: 'business' },
      'org': { register: 1090, renew: 2259, transfer: 2259, description: '非営利団体・コミュニティ', category: 'nonprofit' },
      'net': { register: 1920, renew: 2259, transfer: 2259, description: 'ネットワーク・IT企業', category: 'tech' }
    },
    'japan': {
      'jp': { register: 2035, renew: 3819, transfer: 3819, description: '日本の個人・法人・信頼性重視', category: 'japan' },
      'co.jp': { register: 'contact', renew: 'contact', transfer: 'contact', description: '日本企業専用・最高信頼度', category: 'japan_business' }
    },
    'developer': {
      'cloud': { register: 230, renew: 2878, transfer: 2878, description: '新規最安値・クラウドサービス', category: 'cloud' },
      'dev': { register: 2663, renew: 2663, transfer: 2663, description: 'Google推奨・開発者ポートフォリオ', category: 'developer' },
      'app': { register: 3335, renew: 3335, transfer: 3335, description: 'モバイル・Webアプリ', category: 'developer' },
      'io': { register: 4820, renew: 8714, transfer: 8714, description: 'スタートアップ・API・SaaS', category: 'startup' },
      'tech': { register: 2290, renew: 4580, transfer: 4580, description: 'テクノロジー企業', category: 'tech' }
    },
    'premium': {
      'ai': { register: 16083, renew: 16083, transfer: 16083, description: 'AI・機械学習・フィンテック', category: 'ai' }
    },
    'budget': {
      'info': { register: 380, renew: 2980, transfer: 2980, description: '情報サイト・ブログ', category: 'content' },
      'online': { register: 580, renew: 3450, transfer: 3450, description: 'オンラインビジネス', category: 'business' },
      'store': { register: 690, renew: 5290, transfer: 5290, description: 'ECサイト・オンラインストア', category: 'ecommerce' }
    }
  }
  
  return c.json({
    success: true,
    pricing: allPricing,
    currency: 'JPY',
    prices_include_tax: true,
    total_tlds: Object.values(allPricing).reduce((sum, category) => sum + Object.keys(category).length, 0),
    last_updated: '2024-07-15',
    note: 'Prices are for VALUE-DOMAIN and may change. Contact for .co.jp pricing.',
    recommendations: {
      budget_friendly: ['info', 'cloud', 'online'],
      business: ['com', 'jp', 'co.jp'],
      tech: ['dev', 'io', 'tech', 'app'],
      premium: ['ai']
    }
  })
})

// Test domain registration (WARNING: This will actually register domains!)
testValueDomainRouter.post('/register', async (c) => {
  try {
    const { domain, confirm } = await c.req.json() as { domain: string, confirm?: boolean }
    
    if (!c.env.VALUE_DOMAIN_API_KEY) {
      return c.json({
        success: false,
        error: 'VALUE_DOMAIN_API_KEY not configured'
      }, 400)
    }

    if (!domain) {
      return c.json({
        success: false,
        error: 'Domain name required'
      }, 400)
    }

    // Safety check - require explicit confirmation for actual registration
    if (!confirm) {
      return c.json({
        success: false,
        error: 'This endpoint will actually register domains! Add "confirm": true to proceed.',
        warning: 'This will charge your VALUE-DOMAIN account!'
      }, 400)
    }

    const provider = new ValueDomainProvider(c.env.VALUE_DOMAIN_API_KEY)
    
    // Check availability first
    const availability = await provider.checkAvailability(domain)
    if (!availability.available) {
      return c.json({
        success: false,
        error: 'Domain not available',
        availability
      }, 400)
    }

    console.log(`[TEST REGISTRATION] Attempting to register ${domain} via VALUE-DOMAIN`)
    
    // Register the domain with proper registrant info
    const result = await provider.registerDomain(domain, {
      auto_renew: true,
      privacy_enabled: false, // For regctl.com we want public whois
      years: 1,
      registrant: {
        name: 'Yuki Hamada',
        email: 'yuki@hamada.dev',
        phone: '+81.90-1234-5678', // VALUE-DOMAIN対応の国際電話番号形式
        organization: 'RegiOps',
        address: {
          street: '1-1-1 Chiyoda',
          city: 'Tokyo',
          state: 'Tokyo',
          postal_code: '100-0001',
          country: 'JP'
        }
      }
    })

    console.log(`[TEST REGISTRATION] Successfully registered ${domain}:`, result)
    
    return c.json({
      success: true,
      domain,
      availability,
      registration_result: result,
      warning: 'Domain has been actually registered and charged to your account!'
    })
  } catch (error) {
    console.error(`[TEST REGISTRATION] Failed to register domain:`, error)
    return c.json({
      success: false,
      domain: c.req.body ? JSON.parse(c.req.body).domain : 'unknown',
      error: error instanceof Error ? error.message : 'Unknown error'
    }, 500)
  }
})

// Update nameservers to Cloudflare
testValueDomainRouter.post('/update-nameservers', async (c) => {
  try {
    const { domain, nameservers } = await c.req.json() as { domain: string, nameservers: string[] }
    
    if (!c.env.VALUE_DOMAIN_API_KEY) {
      return c.json({
        success: false,
        error: 'VALUE_DOMAIN_API_KEY not configured'
      }, 400)
    }

    if (!domain || !nameservers || !Array.isArray(nameservers)) {
      return c.json({
        success: false,
        error: 'Domain and nameservers array required'
      }, 400)
    }

    const provider = new ValueDomainProvider(c.env.VALUE_DOMAIN_API_KEY)
    
    console.log(`[NAMESERVER UPDATE] Updating ${domain} nameservers to:`, nameservers)
    
    await provider.updateNameservers(domain, nameservers)

    return c.json({
      success: true,
      domain,
      nameservers,
      message: `Nameservers updated for ${domain}`
    })
  } catch (error) {
    return c.json({
      success: false,
      error: error instanceof Error ? error.message : 'Unknown error'
    }, 500)
  }
})

// Test multiple domains at once
testValueDomainRouter.post('/check-multiple', async (c) => {
  try {
    const { domains } = await c.req.json() as { domains: string[] }
    
    if (!c.env.VALUE_DOMAIN_API_KEY) {
      return c.json({
        success: false,
        error: 'VALUE_DOMAIN_API_KEY not configured'
      }, 400)
    }

    if (!domains || !Array.isArray(domains)) {
      return c.json({
        success: false,
        error: 'Invalid domains array'
      }, 400)
    }

    const provider = new ValueDomainProvider(c.env.VALUE_DOMAIN_API_KEY)
    const results = []
    
    for (const domain of domains) {
      try {
        const result = await provider.checkAvailability(domain)
        results.push({
          domain,
          success: true,
          result
        })
      } catch (error) {
        results.push({
          domain,
          success: false,
          error: error instanceof Error ? error.message : 'Unknown error'
        })
      }
    }
    
    return c.json({
      success: true,
      total_domains: domains.length,
      results
    })
  } catch (error) {
    return c.json({
      success: false,
      error: error instanceof Error ? error.message : 'Unknown error'
    }, 500)
  }
})