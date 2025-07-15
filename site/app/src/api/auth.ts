import { apiClient, setAuthToken, handleApiError } from './client'

export interface User {
  id: string
  email: string
  name: string
  role: 'admin' | 'user' | 'viewer'
  organization_id?: string
}

export interface LoginRequest {
  email: string
  password: string
}

export interface RegisterRequest {
  email: string
  password: string
  name: string
  organization_name?: string
}

export interface AuthResponse {
  token: string
  user: User
}

export interface DeviceCodeResponse {
  device_code: string
  user_code: string
  verification_uri: string
  verification_uri_complete: string
  expires_in: number
  interval: number
}

export interface TokenResponse {
  access_token: string
  token_type: string
  expires_in: number
}

class AuthService {
  async login(data: LoginRequest): Promise<AuthResponse> {
    try {
      const response = await apiClient.post<AuthResponse>('/auth/login', data)
      setAuthToken(response.data.token)
      return response.data
    } catch (error) {
      throw handleApiError(error)
    }
  }

  async register(data: RegisterRequest): Promise<AuthResponse> {
    try {
      const response = await apiClient.post<AuthResponse>('/auth/register', data)
      setAuthToken(response.data.token)
      return response.data
    } catch (error) {
      throw handleApiError(error)
    }
  }

  async logout(): Promise<void> {
    try {
      await apiClient.post('/auth/logout')
    } finally {
      setAuthToken(null)
    }
  }

  async getCurrentUser(): Promise<User> {
    try {
      const response = await apiClient.get<{ user: User }>('/auth/me')
      return response.data.user
    } catch (error) {
      throw handleApiError(error)
    }
  }

  async refreshToken(): Promise<string> {
    try {
      const response = await apiClient.post<{ token: string }>('/auth/refresh')
      setAuthToken(response.data.token)
      return response.data.token
    } catch (error) {
      throw handleApiError(error)
    }
  }

  async requestPasswordReset(email: string): Promise<void> {
    try {
      await apiClient.post('/auth/password/reset', { email })
    } catch (error) {
      throw handleApiError(error)
    }
  }

  async resetPassword(token: string, newPassword: string): Promise<void> {
    try {
      await apiClient.post('/auth/password/reset/confirm', {
        token,
        password: newPassword,
      })
    } catch (error) {
      throw handleApiError(error)
    }
  }

  async changePassword(currentPassword: string, newPassword: string): Promise<void> {
    try {
      await apiClient.post('/auth/password/change', {
        current_password: currentPassword,
        new_password: newPassword,
      })
    } catch (error) {
      throw handleApiError(error)
    }
  }

  // Device flow for CLI authentication
  async getDeviceCode(clientId: string = 'regctl-cli'): Promise<DeviceCodeResponse> {
    try {
      const response = await apiClient.post<DeviceCodeResponse>('/auth/device/code', {
        client_id: clientId,
      })
      return response.data
    } catch (error) {
      throw handleApiError(error)
    }
  }

  async pollDeviceToken(deviceCode: string): Promise<TokenResponse | null> {
    try {
      const response = await apiClient.post<TokenResponse>('/auth/device/token', {
        device_code: deviceCode,
      })
      return response.data
    } catch (error: any) {
      // Handle expected polling errors
      if (error.response?.data?.error === 'authorization_pending') {
        return null
      }
      throw handleApiError(error)
    }
  }

  async approveDevice(userCode: string): Promise<void> {
    try {
      await apiClient.post('/auth/device/approve', { user_code: userCode })
    } catch (error) {
      throw handleApiError(error)
    }
  }

  isAuthenticated(): boolean {
    return !!localStorage.getItem('auth_token')
  }

  getToken(): string | null {
    return localStorage.getItem('auth_token')
  }
}

export const authService = new AuthService()