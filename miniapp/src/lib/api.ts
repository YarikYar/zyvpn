export const API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL ?? "https://api.zaruchevskiy.ru"

export class ApiError extends Error {
  status: number
  body: unknown
  constructor(message: string, status: number, body: unknown) {
    super(message)
    this.status = status
    this.body = body
  }
}

type TelegramWebAppMin = {
  initData?: string
}

function readInitData(): string | null {
  const tg = (window as { Telegram?: { WebApp?: TelegramWebAppMin } }).Telegram
    ?.WebApp
  if (tg?.initData?.length) return tg.initData

  if (typeof window !== "undefined" && window.location.hash) {
    const hash = window.location.hash.replace(/^#/, "")
    const params = new URLSearchParams(hash)
    const initDataRaw = params.get("tgWebAppData")
    if (initDataRaw && initDataRaw.length) return initDataRaw
  }
  return null
}

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE" | "PATCH"
  query?: Record<string, string | number | undefined>
  body?: unknown
  signal?: AbortSignal
  publicEndpoint?: boolean
}

async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const { method = "GET", query, body, signal, publicEndpoint } = opts

  const url = new URL(path, API_BASE_URL)
  if (query) {
    for (const [k, v] of Object.entries(query)) {
      if (v === undefined || v === null) continue
      url.searchParams.set(k, String(v))
    }
  }

  const headers: Record<string, string> = {
    Accept: "application/json",
  }
  if (body !== undefined) headers["Content-Type"] = "application/json"
  if (!publicEndpoint) {
    const initData = readInitData()
    if (!initData) {
      throw new ApiError(
        "Missing Telegram WebApp init data",
        401,
        { code: "no_init_data" },
      )
    }
    headers["X-Telegram-Init-Data"] = initData
  }

  const response = await fetch(url.toString(), {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
    signal,
  })

  const contentType = response.headers.get("content-type") ?? ""
  let payload: unknown = null
  if (contentType.includes("application/json")) {
    try {
      payload = await response.json()
    } catch {
      payload = null
    }
  } else if (response.status !== 204) {
    try {
      payload = await response.text()
    } catch {
      payload = null
    }
  }

  if (!response.ok) {
    const message =
      typeof payload === "object" && payload && "message" in payload
        ? String((payload as { message?: unknown }).message ?? response.statusText)
        : response.statusText
    throw new ApiError(message, response.status, payload)
  }

  return payload as T
}

function unwrapList<T>(value: unknown): T[] {
  if (Array.isArray(value)) return value as T[]
  if (value && typeof value === "object") {
    const obj = value as Record<string, unknown>
    for (const key of [
      "items",
      "data",
      "results",
      "transactions",
      "plans",
      "servers",
      "users",
      "referrals",
      "incidents",
      "rows",
    ]) {
      if (Array.isArray(obj[key])) return obj[key] as T[]
    }
  }
  return []
}

async function listRequest<T>(path: string, opts: RequestOptions = {}): Promise<T[]> {
  const raw = await request<unknown>(path, opts)
  return unwrapList<T>(raw)
}

export type Rates = { ton_usd: number; usd_rub: number; ton_rub: number }

export type User = {
  id: number
  username?: string
  first_name?: string
  last_name?: string
  photo_url?: string
  language_code?: string
  is_admin?: boolean
}

export type Plan = {
  id: string
  name: string
  duration_days: number
  traffic_gb: number | null
  price_ton: number
  price_usd?: number
  price_stars: number
  description?: string
  max_devices?: number
  is_active?: boolean
  visible_to_referrer_id?: number | null
  popular?: boolean
}

export type SubscriptionStatus = {
  active: boolean
  subscription: {
    plan_id: string
    plan_name: string
    started_at: string
    ends_at: string
    auto_renew?: boolean
    server_id?: string
  } | null
  days_remaining: number
  traffic_gb: { used: number; limit: number | null }
}

export type ServerStatus = "online" | "offline"

export type Server = {
  id: string
  country: string
  city?: string
  name?: string
  flag_emoji?: string
  ping_ms?: number
  load_percent?: number
  status?: ServerStatus | string
  status_since?: string
  uptime_24h?: number
  uptime_7d?: number
  traffic_all_time?: number
  is_active?: boolean
}

export type SwitchServerInfo = {
  price_ton: number
  free_switches_remaining: number
}

export type Balance = {
  balance: number
  currency: "TON"
}

export type BalanceTransactionType =
  | "purchase"
  | "topup"
  | "promo"
  | "referral_reward"
  | "switch_server"
  | "refund"

export type BalanceTransaction = {
  id?: string
  type?: BalanceTransactionType | string
  amount?: number
  amount_ton?: number
  description?: string
  created_at?: string
}

export type ReferralLink = { link: string; code: string }

export type ReferralStats = {
  invited: number
  paid: number
  ton_earned: number
  days_earned: number
}

export type ReferralUser = {
  id: number
  name: string
  joined_at: string
  paid: boolean
}

export type PaymentInit = {
  payment_id: string
  deep_link?: string
  invoice_url?: string
}

export type PaymentStatus = {
  payment_id: string
  status: "pending" | "succeeded" | "failed" | "expired"
  subscription_id?: string
}

export type PromoApplyResult = {
  ok: boolean
  type?: "balance" | "trial" | "discount"
  amount?: number
  message?: string
}

export type DailyUptimePoint = { date: string; uptime: number }

export type DailyUptime = {
  server_id: string
  days: DailyUptimePoint[]
}

export type ServerIncident = {
  country?: string
  duration_seconds?: number
  ended_at?: string
  flag_emoji?: string
  server_id?: string
  server_name?: string
  started_at: string
}

export const api = {
  health: () =>
    request<{ status: string }>("/health", { publicEndpoint: true }),
  rates: () =>
    request<Rates>("/api/rates", { publicEndpoint: true }),

  me: () => request<User>("/api/user/me"),

  plans: () => listRequest<Plan>("/api/plans"),
  subscriptionStatus: () =>
    request<SubscriptionStatus>("/api/subscription/status"),
  subscriptionKey: () =>
    request<{ key: string }>("/api/subscription/key"),
  buy: (input: {
    plan_id: string
    server_id?: string
    provider: "ton" | "stars" | "balance" | "cash"
  }) =>
    request<{ payment_id?: string; subscription_id?: string }>(
      "/api/subscription/buy",
      { method: "POST", body: input },
    ),
  trial: () =>
    request<{ ok: boolean }>("/api/subscription/trial", { method: "POST" }),
  switchServerInfo: () =>
    request<SwitchServerInfo>("/api/subscription/switch-server/info"),
  switchServer: (input: { server_id: string; provider?: "balance" | "ton" | "stars" }) =>
    request<{ ok: boolean; payment_id?: string }>(
      "/api/subscription/switch-server",
      { method: "POST", body: input },
    ),

  servers: () => listRequest<Server>("/api/servers"),
  serverUptimeDaily: (serverId: string, days?: number) =>
    request<DailyUptime>(`/api/servers/${encodeURIComponent(serverId)}/uptime/daily`, {
      query: { days },
    }),
  serverIncidents: (params?: { hours?: number; limit?: number }) =>
    listRequest<ServerIncident>("/api/servers/incidents", {
      query: params,
    }),

  paymentTonInit: (paymentId: string) =>
    request<PaymentInit>("/api/payment/ton/init", {
      query: { payment_id: paymentId },
    }),
  paymentTonCheck: (input: { payment_id: string; tx_hash: string }) =>
    request<PaymentStatus>("/api/payment/ton/check", {
      method: "POST",
      body: input,
    }),
  paymentStarsInit: (paymentId: string) =>
    request<PaymentInit>("/api/payment/stars/init", {
      query: { payment_id: paymentId },
    }),
  paymentStarsRefund: (paymentId: string) =>
    request<{ ok: boolean }>("/api/payment/stars/refund", {
      method: "POST",
      query: { payment_id: paymentId },
    }),
  paymentStatus: (paymentId: string) =>
    request<PaymentStatus>("/api/payment/status", {
      query: { payment_id: paymentId },
    }),

  balance: () => request<Balance>("/api/balance"),
  balanceTransactions: (params?: { limit?: number; offset?: number }) =>
    listRequest<BalanceTransaction>("/api/balance/transactions", {
      query: params,
    }),
  balancePay: (input: { plan_id: string }) =>
    request<{ subscription_id: string }>("/api/balance/pay", {
      method: "POST",
      body: input,
    }),
  balanceTopup: (input: { amount: number; provider: "ton" | "stars" }) =>
    request<{ payment_id: string }>("/api/balance/topup", {
      method: "POST",
      body: input,
    }),
  balanceTopupTon: (paymentId: string) =>
    request<PaymentInit>("/api/balance/topup/ton", {
      query: { payment_id: paymentId },
    }),
  balanceTopupStars: (paymentId: string) =>
    request<PaymentInit>("/api/balance/topup/stars", {
      query: { payment_id: paymentId },
    }),
  balanceTopupVerify: (input: { payment_id: string; tx_hash?: string }) =>
    request<PaymentStatus>("/api/balance/topup/verify", {
      method: "POST",
      body: input,
    }),

  promoApply: (input: { code: string }) =>
    request<PromoApplyResult>("/api/promo/apply", {
      method: "POST",
      body: input,
    }),
  promoValidate: (code: string) =>
    request<PromoApplyResult>("/api/promo/validate", {
      query: { code },
    }),

  referralLink: () => request<ReferralLink>("/api/referral/link"),
  referralStats: () => request<ReferralStats>("/api/referral/stats"),
  referralUsers: () => listRequest<ReferralUser>("/api/referral/users"),
  referralApply: (input: { code: string }) =>
    request<{ ok: boolean }>("/api/referral/apply", {
      method: "POST",
      body: input,
    }),
}

export function hasInitData() {
  return readInitData() !== null
}
