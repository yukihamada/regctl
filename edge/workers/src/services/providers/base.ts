export interface DomainAvailability {
  available: boolean
  premium?: boolean
  price?: number
  currency?: string
}

export interface DomainInfo {
  name: string
  status: string
  expires_at: string
  auto_renew: boolean
  locked: boolean
  privacy_enabled: boolean
  nameservers: string[]
  registrant?: {
    name: string
    email: string
    organization?: string
  }
}

export interface RegisterOptions {
  years: number
  auto_renew: boolean
  privacy_enabled: boolean
  nameservers?: string[]
  registrant?: {
    name: string
    email: string
    phone: string
    organization?: string
    address: {
      street: string
      city: string
      state: string
      postal_code: string
      country: string
    }
  }
}

export interface TransferResult {
  transfer_id: string
  status: string
  estimated_completion: string
}

export abstract class DomainProvider {
  protected apiKey: string
  protected endpoint: string

  constructor(apiKey: string, endpoint: string) {
    this.apiKey = apiKey
    this.endpoint = endpoint
  }

  abstract checkAvailability(domain: string): Promise<DomainAvailability>
  abstract getDomainInfo(domain: string): Promise<DomainInfo>
  abstract registerDomain(domain: string, options: RegisterOptions): Promise<DomainInfo>
  abstract transferDomain(domain: string, authCode: string): Promise<TransferResult>
  abstract getAuthCode(domain: string): Promise<string>
  abstract updateNameservers(domain: string, nameservers: string[]): Promise<void>
  abstract renewDomain(domain: string, years: number): Promise<void>
  abstract setAutoRenew(domain: string, enabled: boolean): Promise<void>
  abstract setPrivacy(domain: string, enabled: boolean): Promise<void>
  abstract lockDomain(domain: string): Promise<void>
  abstract unlockDomain(domain: string): Promise<void>

  protected async request(path: string, options: RequestInit = {}, queryString = ''): Promise<Record<string, unknown>> {
    const url = `${this.endpoint}${path}${queryString}`
    const response = await fetch(url, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
    })

    if (!response.ok) {
      const error = await response.text()
      throw new Error(`Provider API error: ${response.status} - ${error}`)
    }

    return response.json()
  }
}