import { DomainProvider, DomainAvailability, DomainInfo, RegisterOptions, TransferResult } from './base'
import { generateId } from '../../utils/id'
import type { Env } from '../../types'

export class Route53Provider extends DomainProvider {
  private secretKey: string
  private env?: Env

  constructor(accessKeyId: string, secretAccessKey: string, env?: Env) {
    super(accessKeyId, 'https://route53domains.amazonaws.com')
    this.secretKey = secretAccessKey
    this.env = env
  }

  async checkAvailability(domain: string): Promise<DomainAvailability> {
    // AWS Route53 Domains API call
    const response = await this.awsRequest('CheckDomainAvailability', {
      DomainName: domain,
    })

    return {
      available: response.Availability === 'AVAILABLE',
      premium: response.Availability === 'AVAILABLE_PREMIUM',
      price: response.Price,
      currency: 'USD',
    }
  }

  async getDomainInfo(domain: string): Promise<DomainInfo> {
    const response = await this.awsRequest('GetDomainDetail', {
      DomainName: domain,
    })

    return {
      name: response.DomainName,
      status: this.mapStatus(response.StatusList?.[0] || 'UNKNOWN'),
      expires_at: response.ExpirationDate,
      auto_renew: response.AutoRenew || false,
      locked: response.AdminContact?.PrivacyProtectAdminContact || false,
      privacy_enabled: response.PrivacyProtectAdminContact || false,
      nameservers: response.Nameservers?.map((ns: { Name: string }) => ns.Name) || [],
      registrant: response.AdminContact ? {
        name: `${response.AdminContact.FirstName} ${response.AdminContact.LastName}`,
        email: response.AdminContact.Email,
        organization: response.AdminContact.OrganizationName,
      } : undefined,
    }
  }

  async registerDomain(domain: string, options: RegisterOptions): Promise<DomainInfo> {
    const contact = options.registrant ? {
      FirstName: options.registrant.name.split(' ')[0],
      LastName: options.registrant.name.split(' ').slice(1).join(' ') || 'N/A',
      ContactType: options.registrant.organization ? 'COMPANY' : 'PERSON',
      OrganizationName: options.registrant.organization,
      AddressLine1: options.registrant.address.street,
      City: options.registrant.address.city,
      State: options.registrant.address.state,
      CountryCode: options.registrant.address.country,
      ZipCode: options.registrant.address.postal_code,
      PhoneNumber: options.registrant.phone,
      Email: options.registrant.email,
    } : undefined

    const response = await this.awsRequest('RegisterDomain', {
      DomainName: domain,
      DurationInYears: options.years,
      AutoRenew: options.auto_renew,
      AdminContact: contact,
      RegistrantContact: contact,
      TechContact: contact,
      PrivacyProtectAdminContact: options.privacy_enabled,
      PrivacyProtectRegistrantContact: options.privacy_enabled,
      PrivacyProtectTechContact: options.privacy_enabled,
    })

    return {
      id: generateId('dom'),
      name: domain,
      expires_at: new Date(Date.now() + options.years * 365 * 24 * 60 * 60 * 1000).toISOString(),
      nameservers: options.nameservers || [],
      operation_id: response.OperationId,
    }
  }

  async transferDomain(domain: string, authCode: string): Promise<TransferResult> {
    const response = await this.awsRequest('TransferDomain', {
      DomainName: domain,
      AuthCode: authCode,
      DurationInYears: 1,
      AutoRenew: true,
    })

    return {
      transfer_id: response.OperationId || generateId('xfr'),
      status: 'pending',
      estimated_completion: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
    }
  }

  async getAuthCode(domain: string): Promise<string> {
    const response = await this.awsRequest('RetrieveDomainAuthCode', {
      DomainName: domain,
    })

    return response.AuthCode
  }

  async updateNameservers(domain: string, nameservers: string[]): Promise<void> {
    await this.awsRequest('UpdateDomainNameservers', {
      DomainName: domain,
      Nameservers: nameservers.map(ns => ({ Name: ns })),
    })
  }

  async renewDomain(domain: string, years: number): Promise<void> {
    await this.awsRequest('RenewDomain', {
      DomainName: domain,
      DurationInYears: years,
    })
  }

  async setAutoRenew(domain: string, _enabled: boolean): Promise<void> {
    await this.awsRequest('EnableDomainAutoRenew', {
      DomainName: domain,
    })
  }

  async setPrivacy(_domain: string, _enabled: boolean): Promise<void> {
    // Route53 doesn't have a direct privacy toggle API
    // Would need to update contact privacy settings
    // Privacy settings update not implemented for Route53
  }

  async lockDomain(domain: string): Promise<void> {
    await this.awsRequest('EnableDomainTransferLock', {
      DomainName: domain,
    })
  }

  async unlockDomain(domain: string): Promise<void> {
    await this.awsRequest('DisableDomainTransferLock', {
      DomainName: domain,
    })
  }

  private async awsRequest(action: string, params: Record<string, unknown>): Promise<Record<string, unknown>> {
    // Simplified AWS API request - in production, use proper AWS SDK v4 signing
    const timestamp = new Date().toISOString()
    
    const response = await this.request('/', {
      method: 'POST',
      headers: {
        'X-Amz-Target': `Route53Domains_v20140515.${action}`,
        'Content-Type': 'application/x-amz-json-1.1',
        'X-Amz-Date': timestamp,
        // In production, add proper AWS signature headers
      },
      body: JSON.stringify(params),
    })

    return response
  }

  private mapStatus(status: string): string {
    const statusMap: Record<string, string> = {
      'ACTIVE': 'active',
      'INACTIVE': 'inactive',
      'EXPIRED': 'expired',
      'TRANSFERRED': 'transferred',
      'PENDING': 'pending',
    }
    return statusMap[status] || 'unknown'
  }
}