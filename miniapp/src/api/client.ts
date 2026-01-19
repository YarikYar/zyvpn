const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

function getInitData(): string {
  return window.Telegram?.WebApp?.initData || ''
}

async function request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const initData = getInitData()

  const response = await fetch(`${API_URL}${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      'X-Telegram-Init-Data': initData,
      ...options.headers,
    },
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Unknown error' }))
    throw new Error(error.error || `HTTP error ${response.status}`)
  }

  return response.json()
}

export const api = {
  // Rates (public, no auth)
  getRates: () => fetch(`${API_URL}/api/rates`).then(r => r.json()),

  // User
  getMe: () => request<{ id: number; subscription?: any }>('/api/user/me'),

  // Plans
  getPlans: () => request<{ plans: any[] }>('/api/plans'),

  // Subscription
  buySubscription: (planId: string, provider: 'ton' | 'stars') =>
    request<{ payment: any; ton_info?: any }>('/api/subscription/buy', {
      method: 'POST',
      body: JSON.stringify({ plan_id: planId, provider }),
    }),

  getSubscriptionKey: () => request<{ key: string }>('/api/subscription/key'),

  getSubscriptionStatus: () => request<any>('/api/subscription/status'),

  activateTrial: () =>
    request<{ success: boolean; subscription: any; key: string }>('/api/subscription/trial', {
      method: 'POST',
    }),

  // Payments
  initTONPayment: (paymentId: string) =>
    request<any>(`/api/payment/ton/init?payment_id=${paymentId}`),

  verifyTONPayment: (paymentId: string, txHash: string) =>
    request<{ success: boolean; key: string }>('/api/payment/ton/check', {
      method: 'POST',
      body: JSON.stringify({ payment_id: paymentId, tx_hash: txHash }),
    }),

  initStarsPayment: (paymentId: string) =>
    request<any>(`/api/payment/stars/init?payment_id=${paymentId}`),

  // Referrals
  getReferralStats: () => request<any>('/api/referral/stats'),

  getReferralLink: () => request<{ link: string; code: string }>('/api/referral/link'),

  applyReferralCode: (code: string) =>
    request<{ success: boolean; message: string }>('/api/referral/apply', {
      method: 'POST',
      body: JSON.stringify({ code }),
    }),

  // Balance
  getBalance: () => request<{ balance: number; currency: string }>('/api/balance'),

  getBalanceTransactions: (limit = 20, offset = 0) =>
    request<{ transactions: any[] }>(`/api/balance/transactions?limit=${limit}&offset=${offset}`),

  payFromBalance: (planId: string) =>
    request<{ success: boolean; new_balance: number; key: string }>('/api/balance/pay', {
      method: 'POST',
      body: JSON.stringify({ plan_id: planId }),
    }),

  // Balance top-up
  initTopUp: (amount: number, provider: 'ton' | 'stars') =>
    request<{ payment_id: string; amount: number; currency: string; provider: string }>('/api/balance/topup', {
      method: 'POST',
      body: JSON.stringify({ amount, provider }),
    }),

  getTopUpTONInfo: (paymentId: string) =>
    request<{ payment_id: string; wallet_address: string; amount: string; comment: string; deep_link: string }>(
      `/api/balance/topup/ton?payment_id=${paymentId}`
    ),

  initTopUpStars: (paymentId: string) =>
    request<{ payment_id: string; amount: number; currency: string; invoice_link: string }>(
      `/api/balance/topup/stars?payment_id=${paymentId}`
    ),

  verifyTopUp: (paymentId: string, txHash: string) =>
    request<{ success: boolean; new_balance: number }>('/api/balance/topup/verify', {
      method: 'POST',
      body: JSON.stringify({ payment_id: paymentId, tx_hash: txHash }),
    }),

  // Payment status polling
  getPaymentStatus: (paymentId: string) =>
    request<{
      payment_id: string
      status: 'pending' | 'awaiting_tx' | 'completed' | 'failed'
      amount: number
      currency: string
      key?: string
      new_balance?: number
    }>(`/api/payment/status?payment_id=${paymentId}`),

  // Promo codes
  applyPromoCode: (code: string) =>
    request<{ success: boolean; type: string; value: number; new_balance?: number; message: string }>('/api/promo/apply', {
      method: 'POST',
      body: JSON.stringify({ code }),
    }),

  validatePromoCode: (code: string) =>
    request<{ valid: boolean; type?: string; value?: number; description?: string; error?: string }>(
      `/api/promo/validate?code=${encodeURIComponent(code)}`
    ),

  // Admin API
  admin: {
    checkAccess: async () => {
      try {
        await request<any>('/api/admin/stats')
        return true
      } catch {
        return false
      }
    },

    getStats: () =>
      request<{ total_users: number; active_subscriptions: number; banned_users: number; active_promo_codes: number }>(
        '/api/admin/stats'
      ),

    listUsers: (limit = 50, offset = 0, search = '') =>
      request<{ users: any[]; total: number }>(
        `/api/admin/users?limit=${limit}&offset=${offset}${search ? `&search=${encodeURIComponent(search)}` : ''}`
      ),

    getUser: (userId: number) =>
      request<any>(`/api/admin/users/${userId}`),

    setBalance: (userId: number, balance: number) =>
      request<{ success: boolean }>(`/api/admin/users/${userId}/balance/set`, {
        method: 'POST',
        body: JSON.stringify({ balance }),
      }),

    addBalance: (userId: number, amount: number) =>
      request<{ success: boolean }>(`/api/admin/users/${userId}/balance/add`, {
        method: 'POST',
        body: JSON.stringify({ amount }),
      }),

    extendSubscription: (userId: number, days: number) =>
      request<{ success: boolean }>(`/api/admin/users/${userId}/subscription/extend`, {
        method: 'POST',
        body: JSON.stringify({ days }),
      }),

    cancelSubscription: (userId: number) =>
      request<{ success: boolean }>(`/api/admin/users/${userId}/subscription/cancel`, {
        method: 'POST',
      }),

    banUser: (userId: number, reason: string, expiresAt?: string) =>
      request<{ success: boolean }>(`/api/admin/users/${userId}/ban`, {
        method: 'POST',
        body: JSON.stringify({ reason, expires_at: expiresAt }),
      }),

    unbanUser: (userId: number) =>
      request<{ success: boolean }>(`/api/admin/users/${userId}/unban`, {
        method: 'POST',
      }),

    banIP: (ip: string, reason: string, expiresAt?: string) =>
      request<{ success: boolean }>('/api/admin/bans/ip', {
        method: 'POST',
        body: JSON.stringify({ ip, reason, expires_at: expiresAt }),
      }),

    unbanIP: (ip: string) =>
      request<{ success: boolean }>('/api/admin/bans/ip/unban', {
        method: 'POST',
        body: JSON.stringify({ ip }),
      }),

    listBans: (limit = 50, offset = 0) =>
      request<{ bans: any[] }>(`/api/admin/bans?limit=${limit}&offset=${offset}`),

    listPromoCodes: (limit = 50, offset = 0) =>
      request<{ promo_codes: any[] }>(`/api/admin/promo?limit=${limit}&offset=${offset}`),

    createPromoCode: (type: 'balance' | 'days', value: number, maxUses?: number, expiresAt?: string, description?: string) =>
      request<any>('/api/admin/promo', {
        method: 'POST',
        body: JSON.stringify({ type, value, max_uses: maxUses, expires_at: expiresAt, description }),
      }),

    createBulkPromoCodes: (count: number, type: 'balance' | 'days', value: number, maxUses?: number, expiresAt?: string, prefix?: string) =>
      request<{ codes: string[]; count: number }>('/api/admin/promo/bulk', {
        method: 'POST',
        body: JSON.stringify({ count, type, value, max_uses: maxUses, expires_at: expiresAt, prefix }),
      }),

    deactivatePromoCode: (code: string) =>
      request<{ success: boolean }>('/api/admin/promo/deactivate', {
        method: 'POST',
        body: JSON.stringify({ code }),
      }),

    getLogs: (limit = 50, offset = 0) =>
      request<{ logs: any[] }>(`/api/admin/logs?limit=${limit}&offset=${offset}`),

    // Plans
    listPlans: () =>
      request<{ plans: any[] }>('/api/admin/plans'),

    createPlan: (data: {
      name: string
      description: string
      duration_days: number
      traffic_gb: number
      max_devices: number
      price_ton: number
      price_stars: number
      price_usd: number
      sort_order: number
    }) =>
      request<any>('/api/admin/plans', {
        method: 'POST',
        body: JSON.stringify(data),
      }),

    updatePlan: (planId: string, data: {
      name?: string
      description?: string
      duration_days?: number
      traffic_gb?: number
      max_devices?: number
      price_ton?: number
      price_stars?: number
      price_usd?: number
      is_active?: boolean
      sort_order?: number
    }) =>
      request<any>(`/api/admin/plans/${planId}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),

    deletePlan: (planId: string) =>
      request<{ success: boolean }>(`/api/admin/plans/${planId}`, {
        method: 'DELETE',
      }),
  },
}
