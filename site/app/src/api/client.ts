import axios, { AxiosInstance, AxiosError } from 'axios'

// API base URL - use environment variable or default
const API_BASE_URL = import.meta.env.VITE_API_URL || 'https://api.regctl.cloud/v1'

// Create axios instance
export const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
  withCredentials: true,
})

// Token management
let authToken: string | null = localStorage.getItem('auth_token')

export const setAuthToken = (token: string | null) => {
  authToken = token
  if (token) {
    localStorage.setItem('auth_token', token)
    apiClient.defaults.headers.common['Authorization'] = `Bearer ${token}`
  } else {
    localStorage.removeItem('auth_token')
    delete apiClient.defaults.headers.common['Authorization']
  }
}

// Initialize token if exists
if (authToken) {
  apiClient.defaults.headers.common['Authorization'] = `Bearer ${authToken}`
}

// Request interceptor
apiClient.interceptors.request.use(
  (config) => {
    // Add timestamp to prevent caching
    if (config.method === 'get') {
      config.params = {
        ...config.params,
        _t: Date.now(),
      }
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interceptor
apiClient.interceptors.response.use(
  (response) => {
    return response
  },
  async (error: AxiosError) => {
    const originalRequest = error.config

    // Handle 401 Unauthorized
    if (error.response?.status === 401 && originalRequest) {
      // Clear token and redirect to login
      setAuthToken(null)
      window.location.href = '/login'
    }

    // Handle other errors
    if (error.response?.data) {
      const errorData = error.response.data as any
      throw new Error(errorData.error || errorData.message || 'An error occurred')
    }

    throw error
  }
)

// API error class
export class ApiError extends Error {
  constructor(
    message: string,
    public status?: number,
    public code?: string,
    public details?: any
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

// Helper function to handle API errors
export const handleApiError = (error: any): ApiError => {
  if (error instanceof AxiosError) {
    const data = error.response?.data as any
    return new ApiError(
      data?.error || data?.message || error.message,
      error.response?.status,
      data?.code,
      data?.details
    )
  }
  return new ApiError(error.message || 'Unknown error')
}