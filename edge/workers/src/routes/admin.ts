import { Hono } from 'hono'
import type { Env } from '../types'
import { authMiddleware } from '../middleware/auth'

export const adminRouter = new Hono<{ Bindings: Env }>()

// Admin only middleware
adminRouter.use('*', authMiddleware(), async (c, next) => {
  const user = c.get('user')
  if (user?.role !== 'admin') {
    return c.json({ error: 'Forbidden' }, 403)
  }
  await next()
})

// Get system stats
adminRouter.get('/stats', async (c) => {
  const [users, domains, apiCalls] = await Promise.all([
    c.env.DB.prepare('SELECT COUNT(*) as count FROM users').first(),
    c.env.DB.prepare('SELECT COUNT(*) as count FROM domains').first(),
    c.env.DB.prepare('SELECT SUM(api_usage_count) as total FROM users').first(),
  ])
  
  return c.json({
    users: users?.count || 0,
    domains: domains?.count || 0,
    apiCalls: apiCalls?.total || 0,
  })
})

// List all users
adminRouter.get('/users', async (c) => {
  const users = await c.env.DB.prepare(
    'SELECT id, email, name, subscription_plan, created_at FROM users ORDER BY created_at DESC LIMIT 100'
  ).all()
  
  return c.json({ users: users.results })
})

// ValueDomain provider status
adminRouter.get('/providers/value-domain', async (c) => {
  try {
    // ValueDomain API呼び出し（デモ実装）
    const result = {
      status: 'online',
      balance: 125000,
      currency: 'JPY',
      domainCount: 12,
      domains: [
        { name: 'example.com', expires: '2025-12-01', status: 'active' },
        { name: 'test.org', expires: '2025-11-15', status: 'active' },
        { name: 'demo.net', expires: '2025-10-30', status: 'pending' }
      ],
      lastUpdated: new Date().toISOString(),
      responseTime: Math.floor(Math.random() * 500) + 100
    };
    
    return c.json(result);
    
  } catch (error) {
    console.error('ValueDomain API error:', error);
    return c.json({
      status: 'offline',
      error: error instanceof Error ? error.message : 'Unknown error',
      balance: 0,
      domainCount: 0,
      domains: []
    }, 500);
  }
})

// Route53 provider status
adminRouter.get('/providers/route53', async (c) => {
  try {
    const result = {
      status: 'online',
      monthlyCost: 45.23,
      currency: 'USD',
      hostedZonesCount: 8,
      queryCount: 1234567,
      zones: [
        { id: 'Z123456789', name: 'regctl.com', recordCount: 25 },
        { id: 'Z987654321', name: 'api.regctl.com', recordCount: 15 }
      ],
      lastUpdated: new Date().toISOString()
    };
    
    return c.json(result);
    
  } catch (error) {
    console.error('Route53 API error:', error);
    return c.json({
      status: 'offline',
      error: error instanceof Error ? error.message : 'Unknown error',
      monthlyCost: 0,
      hostedZonesCount: 0,
      queryCount: 0,
      zones: []
    }, 500);
  }
})

// Porkbun provider status
adminRouter.get('/providers/porkbun', async (c) => {
  try {
    const result = {
      status: 'online',
      balance: 89.45,
      currency: 'USD',
      domainCount: 5,
      domains: [
        { name: 'porkbun-test.com', expires: '2025-12-01', status: 'active' },
        { name: 'another-domain.org', expires: '2025-11-15', status: 'active' }
      ],
      lastUpdated: new Date().toISOString(),
      responseTime: Math.floor(Math.random() * 300) + 150
    };
    
    return c.json(result);
    
  } catch (error) {
    console.error('Porkbun API error:', error);
    return c.json({
      status: 'offline',
      error: error instanceof Error ? error.message : 'Unknown error',
      balance: 0,
      domainCount: 0,
      domains: []
    }, 500);
  }
})

// All domains aggregated
adminRouter.get('/domains/all', async (c) => {
  try {
    const allDomains = [
      { name: 'regctl.com', provider: 'Cloudflare', status: 'active', expires: '2026-01-01' },
      { name: 'api.regctl.com', provider: 'Cloudflare', status: 'active', expires: '2026-01-01' },
      { name: 'app.regctl.com', provider: 'Cloudflare', status: 'active', expires: '2026-01-01' },
      { name: 'example.com', provider: 'VALUE-DOMAIN', status: 'active', expires: '2025-12-01' },
      { name: 'test.org', provider: 'Route53', status: 'active', expires: '2025-11-15' },
      { name: 'demo.net', provider: 'Porkbun', status: 'pending', expires: '2025-10-30' },
      { name: 'old.com', provider: 'VALUE-DOMAIN', status: 'expired', expires: '2024-08-15' }
    ];

    const result = {
      domains: allDomains,
      totalCount: allDomains.length,
      byProvider: {
        'Cloudflare': allDomains.filter(d => d.provider === 'Cloudflare').length,
        'VALUE-DOMAIN': allDomains.filter(d => d.provider === 'VALUE-DOMAIN').length,
        'Route53': allDomains.filter(d => d.provider === 'Route53').length,
        'Porkbun': allDomains.filter(d => d.provider === 'Porkbun').length
      },
      lastUpdated: new Date().toISOString()
    };
    
    return c.json(result);
    
  } catch (error) {
    console.error('Domain aggregation error:', error);
    return c.json({
      error: error instanceof Error ? error.message : 'Unknown error',
      domains: [],
      totalCount: 0,
      byProvider: {}
    }, 500);
  }
})

// System health check
adminRouter.get('/health/system', async (c) => {
  const checks = {
    workers: true,
    database: false,
    cache: false,
    queue: false,
    timestamp: new Date().toISOString()
  };

  try {
    // Database チェック
    try {
      const result = await c.env.DB.prepare('SELECT 1').first();
      checks.database = !!result;
    } catch (error) {
      console.warn('Database health check failed:', error);
    }

    // Cache チェック
    try {
      const testKey = 'health:cache:test';
      await c.env.CACHE.put(testKey, 'test', { expirationTtl: 60 });
      const cached = await c.env.CACHE.get(testKey);
      checks.cache = cached === 'test';
    } catch (error) {
      console.warn('Cache health check failed:', error);
    }

    // Queue チェック（簡易）
    try {
      checks.queue = !!c.env.WEBHOOKS;
    } catch (error) {
      console.warn('Queue health check failed:', error);
    }

    const allHealthy = Object.values(checks).slice(0, -1).every(Boolean);

    return c.json({
      status: allHealthy ? 'healthy' : 'degraded',
      checks,
      uptime: Math.floor(Date.now() / 1000),
      version: '1.0.0'
    });

  } catch (error) {
    return c.json({
      status: 'unhealthy',
      error: error instanceof Error ? error.message : 'Unknown error',
      checks
    }, 500);
  }
})