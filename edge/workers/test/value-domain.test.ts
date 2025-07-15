import { describe, it, expect, beforeEach, vi } from 'vitest'
import { ValueDomainProvider } from '../src/services/providers/value-domain'

describe('ValueDomainProvider', () => {
  let provider: ValueDomainProvider
  
  beforeEach(() => {
    provider = new ValueDomainProvider('test-api-key')
    
    // Mock fetch
    global.fetch = vi.fn()
  })

  describe('checkAvailability', () => {
    it('should check domain availability', async () => {
      const mockResponse = {
        domains: {
          'example.com': {
            available: true,
            premium: false,
            price: 1748
          }
        }
      }

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResponse)
      } as Response)

      const result = await provider.checkAvailability('example.com')

      expect(fetch).toHaveBeenCalledWith(
        'https://api.value-domain.com/v1/domainsearch?domainnames=example.com',
        expect.objectContaining({
          headers: expect.objectContaining({
            'Content-Type': 'application/json'
          })
        })
      )

      expect(result).toEqual({
        available: true,
        premium: false,
        price: 1748,
        currency: 'JPY'
      })
    })

    it('should handle unavailable domains', async () => {
      const mockResponse = {
        domains: {
          'example.com': {
            available: false
          }
        }
      }

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResponse)
      } as Response)

      const result = await provider.checkAvailability('example.com')

      expect(result.available).toBe(false)
    })
  })

  describe('getDomainInfo', () => {
    it('should get domain information', async () => {
      // Mock the domains list response
      const mockDomainsResponse = {
        domains: [
          {
            id: 'dom_123',
            name: 'example.com',
            domainname: 'example.com'
          }
        ]
      }

      // Mock the domain details response
      const mockDomainResponse = {
        domainname: 'example.com',
        status: 'active',
        expires_at: '2025-12-31T23:59:59Z',
        auto_renew: true,
        locked: false,
        privacy_enabled: true,
        nameservers: ['ns1.value-domain.com', 'ns2.value-domain.com'],
        registrant: {
          name: 'John Doe',
          email: 'john@example.com',
          organization: 'Example Corp'
        }
      }

      vi.mocked(fetch)
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve(mockDomainsResponse)
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve(mockDomainResponse)
        } as Response)

      const result = await provider.getDomainInfo('example.com')

      expect(fetch).toHaveBeenCalledTimes(2)
      expect(result).toEqual({
        name: 'example.com',
        status: 'active',
        expires_at: '2025-12-31T23:59:59Z',
        auto_renew: true,
        locked: false,
        privacy_enabled: true,
        nameservers: ['ns1.value-domain.com', 'ns2.value-domain.com'],
        registrant: {
          name: 'John Doe',
          email: 'john@example.com',
          organization: 'Example Corp'
        }
      })
    })

    it('should throw error for domain not found', async () => {
      const mockResponse = { domains: [] }

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResponse)
      } as Response)

      await expect(provider.getDomainInfo('nonexistent.com'))
        .rejects.toThrow('Domain nonexistent.com not found in account')
    })
  })

  describe('registerDomain', () => {
    it('should register a domain', async () => {
      const mockResponse = {
        id: 'dom_456',
        expires_at: '2025-12-31T23:59:59Z'
      }

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResponse)
      } as Response)

      const options = {
        years: 1,
        auto_renew: true,
        privacy_enabled: true
      }

      const result = await provider.registerDomain('newdomain.com', options)

      expect(fetch).toHaveBeenCalledWith(
        'https://api.value-domain.com/v1/domains',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            'Authorization': 'Bearer test-api-key',
            'Content-Type': 'application/json'
          }),
          body: JSON.stringify({
            domainname: 'newdomain.com',
            period: 1,
            auto_renew: true,
            privacy_enabled: true,
            nameservers: undefined,
            registrant: undefined
          })
        })
      )

      expect(result).toEqual({
        name: 'newdomain.com',
        status: 'pending',
        expires_at: '2025-12-31T23:59:59Z',
        auto_renew: true,
        locked: false,
        privacy_enabled: true,
        nameservers: ['ns1.value-domain.com', 'ns2.value-domain.com'],
        registrant: undefined
      })
    })
  })

  describe('listDomains', () => {
    it('should list domains', async () => {
      const mockResponse = {
        domains: [
          {
            id: 'dom_123',
            name: 'example.com',
            status: 'active'
          },
          {
            id: 'dom_456',
            name: 'test.com',
            status: 'active'
          }
        ]
      }

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResponse)
      } as Response)

      const result = await provider.listDomains(10, 0)

      expect(fetch).toHaveBeenCalledWith(
        'https://api.value-domain.com/v1/domains?limit=10&offset=0',
        expect.objectContaining({
          headers: expect.objectContaining({
            'Authorization': 'Bearer test-api-key'
          })
        })
      )

      expect(result).toEqual(mockResponse.domains)
    })
  })

  describe('updateNameservers', () => {
    it('should update nameservers', async () => {
      // Mock the getDomainInfo call
      const mockDomainsResponse = {
        domains: [{ id: 'dom_123', name: 'example.com' }]
      }
      const mockDomainResponse = { domainname: 'example.com' }

      vi.mocked(fetch)
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve(mockDomainsResponse)
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve(mockDomainResponse)
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve({})
        } as Response)

      const nameservers = ['ns1.custom.com', 'ns2.custom.com']
      await provider.updateNameservers('example.com', nameservers)

      expect(fetch).toHaveBeenCalledWith(
        'https://api.value-domain.com/v1/domains/example.com',
        expect.objectContaining({
          method: 'PUT',
          headers: expect.objectContaining({
            'Authorization': 'Bearer test-api-key'
          }),
          body: JSON.stringify({
            nameservers: nameservers
          })
        })
      )
    })
  })

  describe('error handling', () => {
    it('should handle API errors', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: false,
        status: 401,
        text: () => Promise.resolve('Unauthorized')
      } as Response)

      await expect(provider.checkAvailability('example.com'))
        .rejects.toThrow('Provider API error: 401 - Unauthorized')
    })
  })
})