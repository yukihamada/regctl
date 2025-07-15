import { create } from 'zustand'
import { auth as authApi } from '../lib/api'

interface User {
  id: string
  email: string
  name?: string
  emailVerified: boolean
  subscriptionStatus: string
  subscriptionPlan: string
  created_at?: string
}

interface AuthStore {
  user: User | null
  isLoading: boolean
  error: string | null
  
  login: (email: string, password: string) => Promise<void>
  register: (email: string, password: string, name?: string) => Promise<void>
  logout: () => void
  fetchProfile: () => Promise<void>
  clearError: () => void
}

export const useAuthStore = create<AuthStore>((set) => ({
  user: null,
  isLoading: false,
  error: null,

  login: async (email: string, password: string) => {
    set({ isLoading: true, error: null })
    try {
      await authApi.login(email, password)
      const user = await authApi.getProfile()
      set({ user, isLoading: false })
    } catch (error: any) {
      set({ 
        error: error.response?.data?.error || 'Login failed', 
        isLoading: false 
      })
      throw error
    }
  },

  register: async (email: string, password: string, name?: string) => {
    set({ isLoading: true, error: null })
    try {
      await authApi.register(email, password, name)
      set({ isLoading: false })
    } catch (error: any) {
      set({ 
        error: error.response?.data?.error || 'Registration failed', 
        isLoading: false 
      })
      throw error
    }
  },

  logout: () => {
    authApi.logout()
    set({ user: null })
  },

  fetchProfile: async () => {
    if (!authApi.isAuthenticated()) return
    
    set({ isLoading: true })
    try {
      const user = await authApi.getProfile()
      set({ user, isLoading: false })
    } catch (error) {
      set({ user: null, isLoading: false })
    }
  },

  clearError: () => set({ error: null })
}))