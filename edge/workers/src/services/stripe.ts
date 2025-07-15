export class StripeService {
  private apiKey: string
  private baseUrl = 'https://api.stripe.com/v1'

  constructor(apiKey: string) {
    this.apiKey = apiKey
  }

  private async request(
    path: string,
    options: RequestInit = {}
  ): Promise<Record<string, unknown>> {
    const response = await fetch(`${this.baseUrl}${path}`, {
      ...options,
      headers: {
        'Authorization': `Bearer ${this.apiKey}`,
        'Content-Type': 'application/x-www-form-urlencoded',
        ...options.headers,
      },
    })

    if (!response.ok) {
      const error = await response.json()
      throw new Error(error.error?.message || 'Stripe API error')
    }

    return response.json()
  }

  // Customer methods
  async createCustomer(data: {
    email: string
    name?: string
    metadata?: Record<string, string>
  }) {
    const params = new URLSearchParams()
    params.append('email', data.email)
    if (data.name) params.append('name', data.name)
    if (data.metadata) {
      Object.entries(data.metadata).forEach(([key, value]) => {
        params.append(`metadata[${key}]`, value)
      })
    }

    return this.request('/customers', {
      method: 'POST',
      body: params,
    })
  }

  async getCustomer(customerId: string) {
    return this.request(`/customers/${customerId}`)
  }

  // Subscription methods
  async createSubscription(data: {
    customer: string
    items: Array<{ price: string; quantity?: number }>
    trial_period_days?: number
    metadata?: Record<string, string>
  }) {
    const params = new URLSearchParams()
    params.append('customer', data.customer)
    
    data.items.forEach((item, index) => {
      params.append(`items[${index}][price]`, item.price)
      if (item.quantity) {
        params.append(`items[${index}][quantity]`, item.quantity.toString())
      }
    })

    if (data.trial_period_days) {
      params.append('trial_period_days', data.trial_period_days.toString())
    }

    if (data.metadata) {
      Object.entries(data.metadata).forEach(([key, value]) => {
        params.append(`metadata[${key}]`, value)
      })
    }

    return this.request('/subscriptions', {
      method: 'POST',
      body: params,
    })
  }

  async getSubscription(subscriptionId: string) {
    return this.request(`/subscriptions/${subscriptionId}`)
  }

  async updateSubscription(
    subscriptionId: string,
    data: {
      items?: Array<{ price: string; quantity?: number }>
      proration_behavior?: 'create_prorations' | 'none' | 'always_invoice'
    }
  ) {
    const params = new URLSearchParams()

    if (data.items) {
      // Clear existing items
      params.append('items[0][clear]', 'true')
      
      // Add new items
      data.items.forEach((item, index) => {
        params.append(`items[${index + 1}][price]`, item.price)
        if (item.quantity) {
          params.append(`items[${index + 1}][quantity]`, item.quantity.toString())
        }
      })
    }

    if (data.proration_behavior) {
      params.append('proration_behavior', data.proration_behavior)
    }

    return this.request(`/subscriptions/${subscriptionId}`, {
      method: 'POST',
      body: params,
    })
  }

  async cancelSubscription(
    subscriptionId: string,
    options: {
      at_period_end?: boolean
      cancellation_details?: { comment?: string }
    } = {}
  ) {
    const params = new URLSearchParams()
    
    if (options.at_period_end !== undefined) {
      params.append('cancel_at_period_end', options.at_period_end.toString())
    }

    if (options.cancellation_details?.comment) {
      params.append('cancellation_details[comment]', options.cancellation_details.comment)
    }

    return this.request(`/subscriptions/${subscriptionId}`, {
      method: 'POST',
      body: params,
    })
  }

  // Checkout Session methods
  async createCheckoutSession(data: {
    customer?: string
    customer_email?: string
    line_items: Array<{
      price: string
      quantity: number
    }>
    mode: 'payment' | 'setup' | 'subscription'
    success_url: string
    cancel_url: string
    metadata?: Record<string, string>
  }) {
    const params = new URLSearchParams()
    
    if (data.customer) {
      params.append('customer', data.customer)
    } else if (data.customer_email) {
      params.append('customer_email', data.customer_email)
    }

    params.append('mode', data.mode)
    params.append('success_url', data.success_url)
    params.append('cancel_url', data.cancel_url)

    data.line_items.forEach((item, index) => {
      params.append(`line_items[${index}][price]`, item.price)
      params.append(`line_items[${index}][quantity]`, item.quantity.toString())
    })

    if (data.metadata) {
      Object.entries(data.metadata).forEach(([key, value]) => {
        params.append(`metadata[${key}]`, value)
      })
    }

    return this.request('/checkout/sessions', {
      method: 'POST',
      body: params,
    })
  }

  // Invoice methods
  async listInvoices(customerId: string, limit: number = 10) {
    const params = new URLSearchParams()
    params.append('customer', customerId)
    params.append('limit', limit.toString())

    return this.request(`/invoices?${params}`)
  }

  async getInvoice(invoiceId: string) {
    return this.request(`/invoices/${invoiceId}`)
  }

  // Usage Record methods (for metered billing)
  async createUsageRecord(
    subscriptionItemId: string,
    data: {
      quantity: number
      timestamp?: number
      action?: 'increment' | 'set'
    }
  ) {
    const params = new URLSearchParams()
    params.append('quantity', data.quantity.toString())
    
    if (data.timestamp) {
      params.append('timestamp', data.timestamp.toString())
    }

    if (data.action) {
      params.append('action', data.action)
    }

    return this.request(`/subscription_items/${subscriptionItemId}/usage_records`, {
      method: 'POST',
      body: params,
    })
  }

  // Webhook verification
  async verifyWebhook(
    payload: string,
    signature: string,
    webhookSecret: string
  ): Promise<Record<string, unknown>> {
    // Extract timestamp and signature from header
    const elements = signature.split(',')
    let timestamp = ''
    let sig = ''

    for (const element of elements) {
      const [key, value] = element.split('=')
      if (key === 't') timestamp = value
      if (key === 'v1') sig = value
    }

    if (!timestamp || !sig) {
      throw new Error('Invalid signature format')
    }

    // Verify timestamp is within tolerance (5 minutes)
    const currentTime = Math.floor(Date.now() / 1000)
    const signatureTime = parseInt(timestamp)
    if (currentTime - signatureTime > 300) {
      throw new Error('Signature timestamp too old')
    }

    // Compute expected signature
    const signedPayload = `${timestamp}.${payload}`
    const encoder = new TextEncoder()
    const key = await crypto.subtle.importKey(
      'raw',
      encoder.encode(webhookSecret),
      { name: 'HMAC', hash: 'SHA-256' },
      false,
      ['sign']
    )

    const signature_buffer = await crypto.subtle.sign(
      'HMAC',
      key,
      encoder.encode(signedPayload)
    )

    const expectedSig = Array.from(new Uint8Array(signature_buffer))
      .map(b => b.toString(16).padStart(2, '0'))
      .join('')

    // Compare signatures
    if (sig !== expectedSig) {
      throw new Error('Invalid signature')
    }

    return JSON.parse(payload)
  }

  // Price methods
  async createPrice(data: {
    product: string
    unit_amount?: number
    currency: string
    recurring?: {
      interval: 'day' | 'week' | 'month' | 'year'
      interval_count?: number
    }
    usage_type?: 'metered' | 'licensed'
    billing_scheme?: 'per_unit' | 'tiered'
  }) {
    const params = new URLSearchParams()
    params.append('product', data.product)
    params.append('currency', data.currency)

    if (data.unit_amount !== undefined) {
      params.append('unit_amount', data.unit_amount.toString())
    }

    if (data.recurring) {
      params.append('recurring[interval]', data.recurring.interval)
      if (data.recurring.interval_count) {
        params.append('recurring[interval_count]', data.recurring.interval_count.toString())
      }
    }

    if (data.usage_type) {
      params.append('usage_type', data.usage_type)
    }

    if (data.billing_scheme) {
      params.append('billing_scheme', data.billing_scheme)
    }

    return this.request('/prices', {
      method: 'POST',
      body: params,
    })
  }

  // Product methods
  async createProduct(data: {
    name: string
    description?: string
    metadata?: Record<string, string>
  }) {
    const params = new URLSearchParams()
    params.append('name', data.name)

    if (data.description) {
      params.append('description', data.description)
    }

    if (data.metadata) {
      Object.entries(data.metadata).forEach(([key, value]) => {
        params.append(`metadata[${key}]`, value)
      })
    }

    return this.request('/products', {
      method: 'POST',
      body: params,
    })
  }

  // Payment method methods
  async attachPaymentMethod(
    paymentMethodId: string,
    customerId: string
  ) {
    const params = new URLSearchParams()
    params.append('customer', customerId)

    return this.request(`/payment_methods/${paymentMethodId}/attach`, {
      method: 'POST',
      body: params,
    })
  }

  async listPaymentMethods(
    customerId: string,
    type: string = 'card'
  ) {
    const params = new URLSearchParams()
    params.append('customer', customerId)
    params.append('type', type)

    return this.request(`/payment_methods?${params}`)
  }

  // Billing Portal session
  async createBillingPortalSession(data: {
    customer: string
    return_url: string
  }) {
    const params = new URLSearchParams()
    params.append('customer', data.customer)
    params.append('return_url', data.return_url)

    return this.request('/billing_portal/sessions', {
      method: 'POST',
      body: params,
    })
  }
}