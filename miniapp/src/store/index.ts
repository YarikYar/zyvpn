import { create } from 'zustand'
import { api } from '../api/client'
import type { User, Plan, SubscriptionStatus, ReferralStats, ExchangeRates } from '../types'

interface Store {
  user: User | null
  plans: Plan[]
  subscriptionStatus: SubscriptionStatus | null
  referralStats: ReferralStats | null
  referralLink: string | null
  connectionKey: string | null
  rates: ExchangeRates | null
  selectedServerId: string | null
  balance: number
  loading: boolean
  error: string | null

  fetchUser: () => Promise<void>
  fetchPlans: () => Promise<void>
  fetchSubscriptionStatus: () => Promise<void>
  fetchConnectionKey: () => Promise<void>
  fetchReferralStats: () => Promise<void>
  fetchReferralLink: () => Promise<void>
  fetchRates: () => Promise<void>
  fetchBalance: () => Promise<void>
  setSelectedServerId: (id: string | null) => void
  setError: (error: string | null) => void
}

export const useStore = create<Store>((set) => ({
  user: null,
  plans: [],
  subscriptionStatus: null,
  referralStats: null,
  referralLink: null,
  connectionKey: null,
  rates: null,
  selectedServerId: null,
  balance: 0,
  loading: false,
  error: null,

  fetchUser: async () => {
    try {
      set({ loading: true, error: null })
      const data = await api.getMe()
      set({ user: data as User, loading: false })
    } catch (error) {
      set({ error: (error as Error).message, loading: false })
    }
  },

  fetchPlans: async () => {
    try {
      const data = await api.getPlans()
      set({ plans: data.plans })
    } catch (error) {
      set({ error: (error as Error).message })
    }
  },

  fetchSubscriptionStatus: async () => {
    try {
      const data = await api.getSubscriptionStatus()
      set({ subscriptionStatus: data })
    } catch (error) {
      set({ error: (error as Error).message })
    }
  },

  fetchConnectionKey: async () => {
    try {
      const data = await api.getSubscriptionKey()
      set({ connectionKey: data.key })
    } catch (error) {
      set({ connectionKey: null })
    }
  },

  fetchReferralStats: async () => {
    try {
      const data = await api.getReferralStats()
      set({ referralStats: data })
    } catch (error) {
      set({ error: (error as Error).message })
    }
  },

  fetchReferralLink: async () => {
    try {
      const data = await api.getReferralLink()
      set({ referralLink: data.link })
    } catch (error) {
      set({ error: (error as Error).message })
    }
  },

  fetchRates: async () => {
    try {
      const data = await api.getRates()
      set({ rates: data })
    } catch (error) {
      // Fallback rates
      set({ rates: { ton_usd: 5.0, usd_rub: 95.0, ton_rub: 475.0 } })
    }
  },

  fetchBalance: async () => {
    try {
      const data = await api.getBalance()
      set({ balance: data.balance || 0 })
    } catch (error) {
      set({ balance: 0 })
    }
  },

  setSelectedServerId: (id) => set({ selectedServerId: id }),
  setError: (error) => set({ error }),
}))
