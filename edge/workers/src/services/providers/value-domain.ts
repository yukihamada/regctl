import { DomainProvider, DomainAvailability, DomainInfo, RegisterOptions, TransferResult } from './base'
import { generateId } from '../../utils/id'

// VALUE-DOMAIN API Provider
// Documentation: https://www.value-domain.com/api/doc/domain/
// API Base URL: https://api.value-domain.com/v1
// Authentication: Bearer token via Authorization header
export class ValueDomainProvider extends DomainProvider {
  constructor(apiKey: string) {
    super(apiKey, 'https://api.value-domain.com/v1')
  }

  async checkAvailability(domain: string): Promise<DomainAvailability> {
    // Use the domain search endpoint which doesn't require authentication
    const queryString = `?domainnames=${encodeURIComponent(domain)}`
    const response = await this.request('/domainsearch', {}, queryString)

    // VALUE-DOMAIN returns availability info in the response
    const availability = response.domains?.[domain] || response[domain]
    
    return {
      available: availability?.available ?? true,
      premium: availability?.premium ?? false,
      price: availability?.price,
      currency: 'JPY',
    }
  }

  async getDomainInfo(domain: string): Promise<DomainInfo> {
    // First, get the domain list to find the domain_id
    const domainsResponse = await this.request('/domains', {
      headers: {
        'Authorization': `Bearer ${this.apiKey}`,
      },
    })

    // Find the domain in the list
    const domains = domainsResponse.domains || []
    const domainData = domains.find((d: { name?: string; domainname?: string; id?: string }) => d.name === domain || d.domainname === domain)
    
    if (!domainData) {
      throw new Error(`Domain ${domain} not found in account`)
    }

    // Get detailed domain information using domain_id
    const response = await this.request(`/domains/${domainData.id}`, {
      headers: {
        'Authorization': `Bearer ${this.apiKey}`,
      },
    })

    return {
      name: response.domainname || response.name || domain,
      status: this.mapStatus(response.status),
      expires_at: response.expires_at || response.expires,
      auto_renew: response.auto_renew === true || response.autorenew === '1',
      locked: response.locked === true || response.locked === '1',
      privacy_enabled: response.privacy_enabled === true || response.privacy === '1',
      nameservers: response.nameservers || [],
      registrant: response.registrant ? {
        name: response.registrant.name || response.registrant_name,
        email: response.registrant.email || response.registrant_email,
        organization: response.registrant.organization || response.registrant_org,
      } : undefined,
    }
  }

  async registerDomain(domain: string, options: RegisterOptions): Promise<DomainInfo> {
    // ドメインをSLDとTLDに分解
    const domainParts = domain.split('.')
    const sld = domainParts.slice(0, -1).join('.')
    const tld = domainParts[domainParts.length - 1]
    
    // VALUE-DOMAIN API の正しい形式でリクエスト
    const registrationData = {
      registrar: "GMO", // VALUE-DOMAINのレジストラー
      sld: sld,         // Second Level Domain (例: "regctl")
      tld: tld,         // Top Level Domain (例: "com")
      years: options.years || 1,
      whois_proxy: options.privacy_enabled ? 1 : 0,
      ns: options.nameservers || ['ns1.value-domain.com', 'ns2.value-domain.com'],
      // 連絡先情報を正しい形式で送信
      contact: options.registrant ? {
        registrant: {
          firstname: options.registrant.name.split(' ')[0] || options.registrant.name,
          lastname: options.registrant.name.split(' ')[1] || '',
          organization: options.registrant.organization || '',
          country: options.registrant.address.country,
          postalcode: options.registrant.address.postal_code,
          state: options.registrant.address.state,
          city: options.registrant.address.city,
          address1: options.registrant.address.street,
          address2: '',
          email: options.registrant.email,
          phone: options.registrant.phone,
          fax: options.registrant.phone // 必須の場合はfaxも同じ番号
        },
        // admin も同じ情報で設定（VALUE-DOMAINで必須）
        admin: {
          firstname: options.registrant.name.split(' ')[0] || options.registrant.name,
          lastname: options.registrant.name.split(' ')[1] || '',
          organization: options.registrant.organization || '',
          country: options.registrant.address.country,
          postalcode: options.registrant.address.postal_code,
          state: options.registrant.address.state,
          city: options.registrant.address.city,
          address1: options.registrant.address.street,
          address2: '',
          email: options.registrant.email,
          phone: options.registrant.phone,
          fax: options.registrant.phone
        }
      } : undefined
    }

    console.log(`[VALUE-DOMAIN] Registering domain ${domain} with data:`, JSON.stringify(registrationData, null, 2))

    const response = await this.request('/domains', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.apiKey}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(registrationData),
    })

    console.log(`[VALUE-DOMAIN] Registration response:`, response)

    return {
      name: domain,
      status: response.status || 'pending',
      expires_at: response.expires_at || response.expirationdate || new Date(Date.now() + (options.years || 1) * 365 * 24 * 60 * 60 * 1000).toISOString(),
      auto_renew: options.auto_renew,
      locked: false,
      privacy_enabled: options.privacy_enabled,
      nameservers: options.nameservers || ['ns1.value-domain.com', 'ns2.value-domain.com'],
      registrant: options.registrant,
    }
  }

  async transferDomain(domain: string, authCode: string): Promise<TransferResult> {
    const response = await this.request('/domains', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.apiKey}`,
      },
      body: JSON.stringify({
        domainname: domain,
        transfer: true,
        auth_code: authCode,
      }),
    })

    return {
      transfer_id: response.id || response.transfer_id || generateId('xfr'),
      status: response.status || 'pending',
      estimated_completion: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
    }
  }

  async getAuthCode(domain: string): Promise<string> {
    // VALUE-DOMAIN API may have auth code in domain details or separate endpoint
    const response = await this.request(`/domains/${domain}/auth_code`, {
      headers: {
        'Authorization': `Bearer ${this.apiKey}`,
      },
    })

    return response.auth_code || response.authcode
  }

  async updateNameservers(domain: string, nameservers: string[]): Promise<void> {
    // Update nameservers using the correct VALUE-DOMAIN API endpoint
    const response = await this.request(`/domains/${domain}/nameserver`, {
      method: 'PUT',
      headers: {
        'Authorization': `Bearer ${this.apiKey}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        ns: nameservers,
      }),
    })
    
    console.log(`[VALUE-DOMAIN] Nameserver update response for ${domain}:`, response)
  }

  async renewDomain(domain: string, years: number): Promise<void> {
    await this.request(`/domains/${domain}/renew`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.apiKey}`,
      },
      body: JSON.stringify({
        period: years,
      }),
    })
  }

  async setAutoRenew(domain: string, enabled: boolean): Promise<void> {
    await this.request(`/domains/${domain}`, {
      method: 'PUT',
      headers: {
        'Authorization': `Bearer ${this.apiKey}`,
      },
      body: JSON.stringify({
        auto_renew: enabled,
      }),
    })
  }

  async setPrivacy(domain: string, enabled: boolean): Promise<void> {
    await this.request(`/domains/${domain}`, {
      method: 'PUT',
      headers: {
        'Authorization': `Bearer ${this.apiKey}`,
      },
      body: JSON.stringify({
        privacy_enabled: enabled,
      }),
    })
  }

  async lockDomain(domain: string): Promise<void> {
    await this.request(`/domains/${domain}`, {
      method: 'PUT',
      headers: {
        'Authorization': `Bearer ${this.apiKey}`,
      },
      body: JSON.stringify({
        locked: true,
      }),
    })
  }

  async unlockDomain(domain: string): Promise<void> {
    await this.request(`/domains/${domain}`, {
      method: 'PUT',
      headers: {
        'Authorization': `Bearer ${this.apiKey}`,
      },
      body: JSON.stringify({
        locked: false,
      }),
    })
  }

  // VALUE-DOMAIN specific DNS management methods
  async getDNSRecords(domain: string): Promise<Array<Record<string, unknown>>> {
    const response = await this.request(`/domains/${domain}/dns`, {
      headers: {
        'Authorization': `Bearer ${this.apiKey}`,
      },
    })

    return response.records || response.dns_records || []
  }

  async updateDNSRecords(domain: string, records: Array<Record<string, unknown>>): Promise<void> {
    await this.request(`/domains/${domain}/dns`, {
      method: 'PUT',
      headers: {
        'Authorization': `Bearer ${this.apiKey}`,
      },
      body: JSON.stringify({
        records: records,
      }),
    })
  }

  // Get list of all domains in account
  async listDomains(limit = 100, offset = 0): Promise<Array<Record<string, unknown>>> {
    const queryString = `?limit=${limit}&offset=${offset}`
    const response = await this.request('/domains', {
      headers: {
        'Authorization': `Bearer ${this.apiKey}`,
      },
    }, queryString)

    return response.domains || []
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