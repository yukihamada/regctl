import { DomainProvider, DomainAvailability, DomainInfo, RegisterOptions, TransferResult } from './base'
import { generateId } from '../../utils/id'

export class ValueDomainProvider extends DomainProvider {
  constructor(apiKey: string) {
    super(apiKey, 'https://api.value-domain.com/v1')
  }

  async checkAvailability(domain: string): Promise<DomainAvailability> {
    const response = await this.request('/domain/check', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.apiKey}`,
      },
      body: JSON.stringify({ domain }),
    })

    return {
      available: response.available,
      premium: response.premium || false,
      price: response.price,
      currency: 'JPY',
    }
  }

  async getDomainInfo(domain: string): Promise<DomainInfo> {
    const response = await this.request(`/domain/${domain}`, {
      headers: {
        'Authorization': `Bearer ${this.apiKey}`,
      },
    })

    return {
      name: response.domainname,
      status: this.mapStatus(response.status),
      expires_at: response.expires,
      auto_renew: response.autorenew === '1',
      locked: response.locked === '1',
      privacy_enabled: response.privacy === '1',
      nameservers: response.nameservers || [],
      registrant: {
        name: response.registrant_name,
        email: response.registrant_email,
        organization: response.registrant_org,
      },
    }
  }

  async registerDomain(domain: string, options: RegisterOptions): Promise<any> {
    const response = await this.request('/domain/register', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.apiKey}`,
      },
      body: JSON.stringify({
        domain,
        period: options.years,
        autorenew: options.auto_renew ? '1' : '0',
        privacy: options.privacy_enabled ? '1' : '0',
        nameservers: options.nameservers,
        registrant: options.registrant ? {
          name: options.registrant.name,
          email: options.registrant.email,
          phone: options.registrant.phone,
          organization: options.registrant.organization,
          address1: options.registrant.address.street,
          city: options.registrant.address.city,
          state: options.registrant.address.state,
          postcode: options.registrant.address.postal_code,
          country: options.registrant.address.country,
        } : undefined,
      }),
    })

    return {
      id: generateId('dom'),
      name: domain,
      expires_at: response.expires,
      nameservers: options.nameservers || ['ns1.value-domain.com', 'ns2.value-domain.com'],
    }
  }

  async transferDomain(domain: string, authCode: string): Promise<TransferResult> {
    const response = await this.request('/domain/transfer', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.apiKey}`,
      },
      body: JSON.stringify({
        domain,
        authcode: authCode,
      }),
    })

    return {
      transfer_id: response.transfer_id || generateId('xfr'),
      status: 'pending',
      estimated_completion: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
    }
  }

  async getAuthCode(domain: string): Promise<string> {
    const response = await this.request(`/domain/${domain}/authcode`, {
      headers: {
        'Authorization': `Bearer ${this.apiKey}`,
      },
    })

    return response.authcode
  }

  async updateNameservers(domain: string, nameservers: string[]): Promise<void> {
    await this.request(`/domain/${domain}/nameservers`, {
      method: 'PUT',
      headers: {
        'Authorization': `Bearer ${this.apiKey}`,
      },
      body: JSON.stringify({ nameservers }),
    })
  }

  async renewDomain(domain: string, years: number): Promise<void> {
    await this.request('/domain/renew', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.apiKey}`,
      },
      body: JSON.stringify({
        domain,
        period: years,
      }),
    })
  }

  async setAutoRenew(domain: string, enabled: boolean): Promise<void> {
    await this.request(`/domain/${domain}/autorenew`, {
      method: 'PUT',
      headers: {
        'Authorization': `Bearer ${this.apiKey}`,
      },
      body: JSON.stringify({
        autorenew: enabled ? '1' : '0',
      }),
    })
  }

  async setPrivacy(domain: string, enabled: boolean): Promise<void> {
    await this.request(`/domain/${domain}/privacy`, {
      method: 'PUT',
      headers: {
        'Authorization': `Bearer ${this.apiKey}`,
      },
      body: JSON.stringify({
        privacy: enabled ? '1' : '0',
      }),
    })
  }

  async lockDomain(domain: string): Promise<void> {
    await this.request(`/domain/${domain}/lock`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.apiKey}`,
      },
    })
  }

  async unlockDomain(domain: string): Promise<void> {
    await this.request(`/domain/${domain}/unlock`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.apiKey}`,
      },
    })
  }

  private mapStatus(status: string): string {
    const statusMap: Record<string, string> = {
      'active': 'active',
      'expired': 'expired',
      'pending': 'pending',
      'transferring': 'transferring',
      'suspended': 'suspended',
    }
    return statusMap[status] || 'unknown'
  }
}