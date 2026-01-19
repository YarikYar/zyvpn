export interface User {
  id: number
  username?: string
  first_name?: string
  last_name?: string
  language_code?: string
  referral_code: string
  referred_by?: number
  balance: number // Balance in TON
  created_at: string
  updated_at: string
  subscription?: Subscription
}

export interface Subscription {
  id: string
  user_id: number
  plan_id: string
  server_id?: string
  status: 'active' | 'expired' | 'cancelled'
  xui_client_id: string
  xui_email: string
  connection_key: string
  started_at?: string
  expires_at?: string
  traffic_limit: number
  traffic_used: number
  max_devices: number
  created_at: string
}

export interface Plan {
  id: string
  name: string
  description: string
  duration_days: number
  traffic_gb: number
  max_devices: number
  price_ton: number
  price_stars: number
  price_usd: number
  is_active: boolean
  sort_order: number
}

export interface Payment {
  id: string
  user_id: number
  subscription_id?: string
  plan_id: string
  provider: 'ton' | 'stars' | 'balance'
  amount: number
  currency: string
  status: 'pending' | 'completed' | 'failed'
  external_id?: string
  created_at: string
  completed_at?: string
}

export interface TONPaymentInfo {
  payment_id: string
  wallet_address: string
  amount: string
  comment: string
  deep_link: string
}

export interface ReferralStats {
  total_referrals: number
  pending_referrals: number
  credited_bonus_ton: number
}

export interface BalanceTransaction {
  id: string
  user_id: number
  amount: number
  type: 'referral_bonus' | 'giveaway' | 'subscription_payment' | 'refund' | 'manual' | 'top_up' | 'promo_code'
  description?: string
  reference_id?: string
  balance_before: number
  balance_after: number
  created_at: string
}

export interface SubscriptionStatus {
  active: boolean
  subscription?: Subscription
  days_remaining: number
  traffic_gb: {
    used: number
    limit: number
    remaining: number
  }
}

export interface ExchangeRates {
  ton_usd: number
  usd_rub: number
  ton_rub: number
}
