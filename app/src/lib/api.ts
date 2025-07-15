import axios from 'axios'

const API_URL = import.meta.env.VITE_API_URL || 'https://regctl-api.yukihamada.workers.dev'

export const api = axios.create({
  baseURL: API_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Add auth token to requests
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Handle auth errors
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export const auth = {
  async login(email: string, password: string) {
    const { data } = await api.post('/api/v1/auth/login', { email, password })
    localStorage.setItem('token', data.token)
    return data
  },

  async register(email: string, password: string, name?: string) {
    const { data } = await api.post('/api/v1/users/register', { email, password, name })
    return data
  },

  async logout() {
    localStorage.removeItem('token')
    window.location.href = '/login'
  },

  async getProfile() {
    const { data } = await api.get('/api/v1/users/me')
    return data
  },

  isAuthenticated() {
    return !!localStorage.getItem('token')
  }
}

export const domains = {
  async list(params?: { page?: number; limit?: number; registrar?: string }) {
    const { data } = await api.get('/api/v1/domains', { params })
    return data
  },

  async get(id: string) {
    const { data } = await api.get(`/api/v1/domains/${id}`)
    return data
  },

  async create(domain: { name: string; registrar: string }) {
    const { data } = await api.post('/api/v1/domains', domain)
    return data
  },

  async update(id: string, updates: any) {
    const { data } = await api.patch(`/api/v1/domains/${id}`, updates)
    return data
  },

  async delete(id: string) {
    const { data } = await api.delete(`/api/v1/domains/${id}`)
    return data
  }
}

export const dns = {
  async listRecords(domainId: string) {
    const { data } = await api.get(`/api/v1/dns/${domainId}/records`)
    return data
  },

  async createRecord(domainId: string, record: any) {
    const { data } = await api.post(`/api/v1/dns/${domainId}/records`, record)
    return data
  },

  async updateRecord(domainId: string, recordId: string, updates: any) {
    const { data } = await api.patch(`/api/v1/dns/${domainId}/records/${recordId}`, updates)
    return data
  },

  async deleteRecord(domainId: string, recordId: string) {
    const { data } = await api.delete(`/api/v1/dns/${domainId}/records/${recordId}`)
    return data
  }
}

export const billing = {
  async getPlans() {
    const { data } = await api.get('/api/v1/billing/plans')
    return data
  },

  async getSubscription() {
    const { data } = await api.get('/api/v1/billing/subscription')
    return data
  },

  async createCheckout(planId: string, interval: 'month' | 'year' = 'month') {
    const { data } = await api.post('/api/v1/billing/checkout', { planId, interval })
    return data
  },

  async cancelSubscription() {
    const { data } = await api.post('/api/v1/billing/cancel')
    return data
  },

  async getUsage() {
    const { data } = await api.get('/api/v1/billing/usage')
    return data
  },

  async getInvoices() {
    const { data } = await api.get('/api/v1/billing/invoices')
    return data
  }
}