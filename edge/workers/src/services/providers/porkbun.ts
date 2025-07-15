import { DomainProvider, DomainAvailability, DomainInfo, RegisterOptions, TransferResult } from './base'
import { generateId } from '../../utils/id'

export class PorkbunProvider extends DomainProvider {
  private apiSecret: string

  constructor(apiKey: string, apiSecret: string) {
    super(apiKey, 'https://porkbun.com/api/json/v3')
    this.apiSecret = apiSecret
  }

  async checkAvailability(domain: string): Promise<DomainAvailability> {
    const response = await this.request('/domain/pricing', {
      method: 'POST',
      body: JSON.stringify({
        apikey: this.apiKey,
        secretapikey: this.apiSecret,
      }),
    })

    const tld = domain.substring(domain.lastIndexOf('.'))
    const pricing = response.pricing[tld]

    if (!pricing) {
      return { available: false }
    }

    const checkResponse = await this.request('/domain/check', {
      method: 'POST',
      body: JSON.stringify({
        apikey: this.apiKey,
        secretapikey: this.apiSecret,
        domain,
      }),
    })

    return {
      available: checkResponse.available === 'yes',
      premium: pricing.premium === 'yes',
      price: parseFloat(pricing.registration),
      currency: 'USD',
    }
  }

  async getDomainInfo(domain: string): Promise<DomainInfo> {
    const response = await this.request('/domain/info', {
      method: 'POST',
      body: JSON.stringify({
        apikey: this.apiKey,
        secretapikey: this.apiSecret,
        domain,
      }),
    })

    const nsResponse = await this.request('/domain/getnameservers', {
      method: 'POST',
      body: JSON.stringify({
        apikey: this.apiKey,
        secretapikey: this.apiSecret,
        domain,
      }),
    })

    return {
      name: response.domain,
      status: response.status === 'active' ? 'active' : 'inactive',
      expires_at: response.expire_date,
      auto_renew: response.auto_renew === 'yes',
      locked: response.locked === 'yes',
      privacy_enabled: response.whois_privacy === 'yes',
      nameservers: nsResponse.nameservers || [],
      registrant: {
        name: response.registrant_name,
        email: response.registrant_email,
        organization: response.registrant_company,
      },
    }
  }

  async registerDomain(domain: string, options: RegisterOptions): Promise<DomainInfo> {
    const response = await this.request('/domain/create', {
      method: 'POST',
      body: JSON.stringify({
        apikey: this.apiKey,
        secretapikey: this.apiSecret,
        domain,
        years: options.years,
        auto_renew: options.auto_renew,
        whois_privacy: options.privacy_enabled,
        registrant: options.registrant ? {
          first_name: options.registrant.name.split(' ')[0],
          last_name: options.registrant.name.split(' ').slice(1).join(' ') || 'N/A',
          company: options.registrant.organization,
          email: options.registrant.email,
          phone: options.registrant.phone,
          address: options.registrant.address.street,
          city: options.registrant.address.city,
          state: options.registrant.address.state,
          zip: options.registrant.address.postal_code,
          country: options.registrant.address.country,
        } : undefined,
      }),
    })

    if (options.nameservers && options.nameservers.length > 0) {
      await this.updateNameservers(domain, options.nameservers)
    }

    return {
      id: generateId('dom'),
      name: domain,
      expires_at: new Date(Date.now() + options.years * 365 * 24 * 60 * 60 * 1000).toISOString(),
      nameservers: options.nameservers || response.nameservers || [],
      order_id: response.order_id,
    }
  }

  async transferDomain(domain: string, authCode: string): Promise<TransferResult> {
    const response = await this.request('/domain/transfer', {
      method: 'POST',
      body: JSON.stringify({
        apikey: this.apiKey,
        secretapikey: this.apiSecret,
        domain,
        auth_code: authCode,
      }),
    })

    return {
      transfer_id: response.order_id || generateId('xfr'),
      status: 'pending',
      estimated_completion: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
    }
  }

  async getAuthCode(domain: string): Promise<string> {
    const response = await this.request('/domain/getauthcode', {
      method: 'POST',
      body: JSON.stringify({
        apikey: this.apiKey,
        secretapikey: this.apiSecret,
        domain,
      }),
    })

    return response.auth_code
  }

  async updateNameservers(domain: string, nameservers: string[]): Promise<void> {
    await this.request('/domain/updatenameservers', {
      method: 'POST',
      body: JSON.stringify({
        apikey: this.apiKey,
        secretapikey: this.apiSecret,
        domain,
        nameservers,
      }),
    })
  }

  async renewDomain(domain: string, years: number): Promise<void> {
    await this.request('/domain/renew', {
      method: 'POST',
      body: JSON.stringify({
        apikey: this.apiKey,
        secretapikey: this.apiSecret,
        domain,
        years,
      }),
    })
  }

  async setAutoRenew(domain: string, enabled: boolean): Promise<void> {
    await this.request('/domain/setautorenew', {
      method: 'POST',
      body: JSON.stringify({
        apikey: this.apiKey,
        secretapikey: this.apiSecret,
        domain,
        auto_renew: enabled,
      }),
    })
  }

  async setPrivacy(domain: string, enabled: boolean): Promise<void> {
    await this.request('/domain/setprivacy', {
      method: 'POST',
      body: JSON.stringify({
        apikey: this.apiKey,
        secretapikey: this.apiSecret,
        domain,
        whois_privacy: enabled,
      }),
    })
  }

  async lockDomain(domain: string): Promise<void> {
    await this.request('/domain/setlock', {
      method: 'POST',
      body: JSON.stringify({
        apikey: this.apiKey,
        secretapikey: this.apiSecret,
        domain,
        lock: true,
      }),
    })
  }

  async unlockDomain(domain: string): Promise<void> {
    await this.request('/domain/setlock', {
      method: 'POST',
      body: JSON.stringify({
        apikey: this.apiKey,
        secretapikey: this.apiSecret,
        domain,
        lock: false,
      }),
    })
  }
}