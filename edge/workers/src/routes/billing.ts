import { Hono } from 'hono'
import { z } from 'zod'
import type { Env, StripeCheckoutSession, StripeSubscription, StripeInvoice } from '../types'
import { authMiddleware } from '../middleware/auth'
import { generateId } from '../utils/id'

export const billingRouter = new Hono<{ Bindings: Env }>()

// Pricing plans
const PLANS = {
  free: {
    id: 'free',
    name: 'Free',
    price: 0,
    interval: null,
    features: {
      domains: 3,
      dnsRecords: 100,
      apiCalls: 1000,
      webhooks: 0,
      customDns: false,
      apiKeys: 1,
      support: 'community'
    }
  },
  pro: {
    id: 'pro',
    name: 'Pro',
    price: 1900, // $19/month in cents
    interval: 'month',
    stripePriceId: 'price_pro_monthly',
    features: {
      domains: 50,
      dnsRecords: 5000,
      apiCalls: 100000,
      webhooks: 10,
      customDns: true,
      apiKeys: 10,
      support: 'email'
    }
  },
  enterprise: {
    id: 'enterprise',
    name: 'Enterprise',
    price: 9900, // $99/month in cents
    interval: 'month',
    stripePriceId: 'price_enterprise_monthly',
    features: {
      domains: -1, // unlimited
      dnsRecords: -1,
      apiCalls: -1,
      webhooks: -1,
      customDns: true,
      apiKeys: -1,
      support: 'priority'
    }
  }
}

// Get pricing plans
billingRouter.get('/plans', (c) => {
  return c.json(PLANS)
})

// Get current subscription
billingRouter.get('/subscription', authMiddleware(), async (c) => {
  const userId = c.get('userId')

  const user = await c.env.DB.prepare(
    `SELECT subscription_status, subscription_plan, subscription_expires_at, 
            stripe_customer_id, api_usage_count, api_usage_reset_at
     FROM users WHERE id = ?`
  ).bind(userId).first()

  if (!user) {
    return c.json({ error: 'User not found' }, 404)
  }

  const plan = PLANS[user.subscription_plan as keyof typeof PLANS] || PLANS.free

  return c.json({
    status: user.subscription_status,
    plan: {
      ...plan,
      current: true
    },
    expiresAt: user.subscription_expires_at,
    usage: {
      apiCalls: {
        used: user.api_usage_count,
        limit: plan.features.apiCalls,
        resetAt: user.api_usage_reset_at
      }
    }
  })
})

// Create checkout session
billingRouter.post('/checkout', authMiddleware(), async (c) => {
  try {
    const userId = c.get('userId')
    const body = await c.req.json()
    const { planId, interval } = z.object({
      planId: z.enum(['pro', 'enterprise']),
      interval: z.enum(['month', 'year']).optional().default('month')
    }).parse(body)

    const user = await c.env.DB.prepare(
      'SELECT email, stripe_customer_id FROM users WHERE id = ?'
    ).bind(userId).first()

    if (!user) {
      return c.json({ error: 'User not found' }, 404)
    }

    // Get Stripe price ID
    const plan = PLANS[planId]
    let priceId = plan.stripePriceId

    if (interval === 'year' && planId === 'pro') {
      priceId = 'price_pro_yearly'
    } else if (interval === 'year' && planId === 'enterprise') {
      priceId = 'price_enterprise_yearly'
    }

    // Create Stripe checkout session
    const stripe = await getStripeClient(c.env)
    
    const session = await stripe.checkout.sessions.create({
      customer: user.stripe_customer_id || undefined,
      customer_email: user.stripe_customer_id ? undefined : user.email,
      mode: 'subscription',
      payment_method_types: ['card'],
      line_items: [{
        price: priceId,
        quantity: 1,
      }],
      success_url: `${c.env.APP_URL}/billing/success?session_id={CHECKOUT_SESSION_ID}`,
      cancel_url: `${c.env.APP_URL}/billing`,
      metadata: {
        userId,
        planId,
        interval
      }
    })

    return c.json({
      checkoutUrl: session.url,
      sessionId: session.id
    })
  } catch (error) {
    if (error instanceof z.ZodError) {
      return c.json({ error: 'Invalid input', details: error.errors }, 400)
    }
    throw error
  }
})

// Cancel subscription
billingRouter.post('/cancel', authMiddleware(), async (c) => {
  const userId = c.get('userId')

  const user = await c.env.DB.prepare(
    'SELECT stripe_customer_id, subscription_status FROM users WHERE id = ?'
  ).bind(userId).first()

  if (!user || !user.stripe_customer_id) {
    return c.json({ error: 'No active subscription' }, 404)
  }

  if (user.subscription_status === 'free') {
    return c.json({ error: 'No active subscription to cancel' }, 400)
  }

  // Cancel in Stripe
  const stripe = await getStripeClient(c.env)
  
  const subscriptions = await stripe.subscriptions.list({
    customer: user.stripe_customer_id,
    status: 'active',
    limit: 1
  })

  if (subscriptions.data.length === 0) {
    return c.json({ error: 'No active subscription found' }, 404)
  }

  const subscription = await stripe.subscriptions.update(
    subscriptions.data[0].id,
    { cancel_at_period_end: true }
  )

  // Update user record
  await c.env.DB.prepare(
    `UPDATE users 
     SET subscription_status = 'canceling', 
         subscription_expires_at = ?,
         updated_at = ?
     WHERE id = ?`
  ).bind(
    new Date(subscription.current_period_end * 1000).toISOString(),
    new Date().toISOString(),
    userId
  ).run()

  return c.json({
    message: 'Subscription will be canceled at the end of the current period',
    expiresAt: new Date(subscription.current_period_end * 1000).toISOString()
  })
})

// Get usage
billingRouter.get('/usage', authMiddleware(), async (c) => {
  const userId = c.get('userId')

  const user = await c.env.DB.prepare(
    `SELECT subscription_plan, api_usage_count, api_usage_reset_at
     FROM users WHERE id = ?`
  ).bind(userId).first()

  if (!user) {
    return c.json({ error: 'User not found' }, 404)
  }

  const plan = PLANS[user.subscription_plan as keyof typeof PLANS] || PLANS.free

  // Get detailed usage for current period
  const startOfMonth = new Date()
  startOfMonth.setDate(1)
  startOfMonth.setHours(0, 0, 0, 0)

  const domainCount = await c.env.DB.prepare(
    'SELECT COUNT(*) as count FROM domains WHERE user_id = ? AND status = "active"'
  ).bind(userId).first()

  const dnsRecordCount = await c.env.DB.prepare(
    `SELECT COUNT(*) as count FROM dns_records 
     WHERE domain_id IN (SELECT id FROM domains WHERE user_id = ?)`
  ).bind(userId).first()

  const apiKeyCount = await c.env.DB.prepare(
    'SELECT COUNT(*) as count FROM api_keys WHERE user_id = ? AND is_active = true'
  ).bind(userId).first()

  const webhookCount = await c.env.DB.prepare(
    'SELECT COUNT(*) as count FROM webhooks WHERE user_id = ? AND is_active = true'
  ).bind(userId).first()

  return c.json({
    plan: user.subscription_plan,
    period: {
      start: startOfMonth.toISOString(),
      end: user.api_usage_reset_at
    },
    usage: {
      domains: {
        used: domainCount?.count || 0,
        limit: plan.features.domains,
        percentage: plan.features.domains === -1 ? 0 : 
          Math.round(((domainCount?.count || 0) / plan.features.domains) * 100)
      },
      dnsRecords: {
        used: dnsRecordCount?.count || 0,
        limit: plan.features.dnsRecords,
        percentage: plan.features.dnsRecords === -1 ? 0 :
          Math.round(((dnsRecordCount?.count || 0) / plan.features.dnsRecords) * 100)
      },
      apiCalls: {
        used: user.api_usage_count,
        limit: plan.features.apiCalls,
        percentage: plan.features.apiCalls === -1 ? 0 :
          Math.round((user.api_usage_count / plan.features.apiCalls) * 100)
      },
      apiKeys: {
        used: apiKeyCount?.count || 0,
        limit: plan.features.apiKeys
      },
      webhooks: {
        used: webhookCount?.count || 0,
        limit: plan.features.webhooks
      }
    }
  })
})

// Get invoices
billingRouter.get('/invoices', authMiddleware(), async (c) => {
  const userId = c.get('userId')

  const user = await c.env.DB.prepare(
    'SELECT stripe_customer_id FROM users WHERE id = ?'
  ).bind(userId).first()

  if (!user || !user.stripe_customer_id) {
    return c.json({ invoices: [] })
  }

  const stripe = await getStripeClient(c.env)
  
  const invoices = await stripe.invoices.list({
    customer: user.stripe_customer_id,
    limit: 100
  })

  return c.json({
    invoices: invoices.data.map(invoice => ({
      id: invoice.id,
      number: invoice.number,
      amount: invoice.amount_paid,
      currency: invoice.currency,
      status: invoice.status,
      pdfUrl: invoice.invoice_pdf,
      createdAt: new Date(invoice.created * 1000).toISOString(),
      periodStart: new Date(invoice.period_start * 1000).toISOString(),
      periodEnd: new Date(invoice.period_end * 1000).toISOString()
    }))
  })
})

// Stripe webhook handler
billingRouter.post('/webhook', async (c) => {
  const signature = c.req.header('stripe-signature')
  if (!signature) {
    return c.json({ error: 'Missing signature' }, 400)
  }

  const body = await c.req.text()
  
  try {
    const stripe = await getStripeClient(c.env)
    const event = stripe.webhooks.constructEvent(
      body,
      signature,
      c.env.STRIPE_WEBHOOK_SECRET
    )

    switch (event.type) {
      case 'checkout.session.completed':
        await handleCheckoutCompleted(event.data.object, c.env)
        break
      
      case 'customer.subscription.updated':
        await handleSubscriptionUpdated(event.data.object, c.env)
        break
      
      case 'customer.subscription.deleted':
        await handleSubscriptionDeleted(event.data.object, c.env)
        break
      
      case 'invoice.payment_failed':
        await handlePaymentFailed(event.data.object, c.env)
        break
    }

    return c.json({ received: true })
  } catch (error) {
    console.error('Webhook error:', error)
    return c.json({ error: 'Webhook processing failed' }, 400)
  }
})

// Helper functions
async function getStripeClient(_env: Env) {
  // In a real implementation, you would use the Stripe SDK
  // For now, this is a placeholder
  throw new Error('Stripe integration not implemented')
}

async function handleCheckoutCompleted(session: StripeCheckoutSession, env: Env) {
  const userId = session.metadata.userId
  const planId = session.metadata.planId
  const customerId = session.customer

  // Update user subscription
  await env.DB.prepare(
    `UPDATE users 
     SET stripe_customer_id = ?,
         subscription_status = 'active',
         subscription_plan = ?,
         subscription_expires_at = ?,
         updated_at = ?
     WHERE id = ?`
  ).bind(
    customerId,
    planId,
    new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString(), // 30 days
    new Date().toISOString(),
    userId
  ).run()

  // Log event
  await env.DB.prepare(
    `INSERT INTO audit_logs (id, user_id, action, resource_type, metadata, created_at)
     VALUES (?, ?, ?, ?, ?, ?)`
  ).bind(
    generateId('log'),
    userId,
    'subscription.created',
    'billing',
    JSON.stringify({ planId, sessionId: session.id }),
    new Date().toISOString()
  ).run()
}

async function handleSubscriptionUpdated(_subscription: StripeSubscription, _env: Env) {
  // Implementation
}

async function handleSubscriptionDeleted(_subscription: StripeSubscription, _env: Env) {
  // Implementation
}

async function handlePaymentFailed(_invoice: StripeInvoice, _env: Env) {
  // Implementation
}