import {
  QueryClient,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query"

import { api, hasInitData } from "@/lib/api"

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      gcTime: 5 * 60_000,
      retry: 1,
      refetchOnWindowFocus: true,
      enabled: hasInitData(),
    },
    mutations: {
      retry: 0,
    },
  },
})

export const queryKeys = {
  rates: ["rates"] as const,
  me: ["me"] as const,
  plans: ["plans"] as const,
  subscriptionStatus: ["subscription", "status"] as const,
  subscriptionKey: ["subscription", "key"] as const,
  switchServerInfo: ["switch-server", "info"] as const,
  servers: ["servers"] as const,
  serverUptimeDaily: (serverId: string | undefined, days?: number) =>
    ["servers", "uptime-daily", serverId ?? "", days ?? 30] as const,
  serverIncidents: (params?: { hours?: number; limit?: number }) =>
    ["servers", "incidents", params ?? {}] as const,
  balance: ["balance"] as const,
  balanceTransactions: (params?: { limit?: number; offset?: number }) =>
    ["balance", "transactions", params ?? {}] as const,
  referralLink: ["referral", "link"] as const,
  referralStats: ["referral", "stats"] as const,
  referralUsers: ["referral", "users"] as const,
  paymentStatus: (id: string) => ["payment", "status", id] as const,
}

export function useRates() {
  return useQuery({
    queryKey: queryKeys.rates,
    queryFn: () => api.rates(),
    enabled: true,
    staleTime: 60_000,
  })
}

export function useMe() {
  return useQuery({ queryKey: queryKeys.me, queryFn: () => api.me() })
}

export function usePlans() {
  return useQuery({ queryKey: queryKeys.plans, queryFn: () => api.plans() })
}

export function useSubscriptionStatus() {
  return useQuery({
    queryKey: queryKeys.subscriptionStatus,
    queryFn: () => api.subscriptionStatus(),
  })
}

export function useSubscriptionKey() {
  return useQuery({
    queryKey: queryKeys.subscriptionKey,
    queryFn: () => api.subscriptionKey(),
    enabled: hasInitData(),
  })
}

export function useSwitchServerInfo() {
  return useQuery({
    queryKey: queryKeys.switchServerInfo,
    queryFn: () => api.switchServerInfo(),
  })
}

export function useBuySubscription() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.buy,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.subscriptionStatus })
      qc.invalidateQueries({ queryKey: queryKeys.balance })
    },
  })
}

export function useActivateTrial() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.trial,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.subscriptionStatus })
    },
  })
}

export function useSwitchServer() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.switchServer,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.subscriptionStatus })
      qc.invalidateQueries({ queryKey: queryKeys.subscriptionKey })
      qc.invalidateQueries({ queryKey: queryKeys.switchServerInfo })
      qc.invalidateQueries({ queryKey: queryKeys.balance })
    },
  })
}


export function useServers() {
  return useQuery({
    queryKey: queryKeys.servers,
    queryFn: () => api.servers(),
  })
}

export function useServerUptimeDaily(serverId: string | undefined, days?: number) {
  return useQuery({
    queryKey: queryKeys.serverUptimeDaily(serverId, days),
    queryFn: () => api.serverUptimeDaily(serverId!, days),
    enabled: !!serverId,
    staleTime: 5 * 60_000,
  })
}

export function useServerIncidents(params?: { hours?: number; limit?: number }) {
  return useQuery({
    queryKey: queryKeys.serverIncidents(params),
    queryFn: () => api.serverIncidents(params),
    staleTime: 60_000,
  })
}


export function useBalance() {
  return useQuery({
    queryKey: queryKeys.balance,
    queryFn: () => api.balance(),
  })
}

export function useBalanceTransactions(params?: { limit?: number; offset?: number }) {
  return useQuery({
    queryKey: queryKeys.balanceTransactions(params),
    queryFn: () => api.balanceTransactions(params),
  })
}

export function usePayWithBalance() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.balancePay,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.balance })
      qc.invalidateQueries({ queryKey: queryKeys.subscriptionStatus })
    },
  })
}

export function useTopupInit() {
  return useMutation({ mutationFn: api.balanceTopup })
}


export function useApplyPromo() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.promoApply,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.balance })
      qc.invalidateQueries({ queryKey: queryKeys.subscriptionStatus })
    },
  })
}


export function useReferralLink() {
  return useQuery({
    queryKey: queryKeys.referralLink,
    queryFn: () => api.referralLink(),
  })
}

export function useReferralStats() {
  return useQuery({
    queryKey: queryKeys.referralStats,
    queryFn: () => api.referralStats(),
  })
}

export function useReferralUsers() {
  return useQuery({
    queryKey: queryKeys.referralUsers,
    queryFn: () => api.referralUsers(),
  })
}

export function useApplyReferral() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.referralApply,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.referralStats })
      qc.invalidateQueries({ queryKey: queryKeys.referralUsers })
    },
  })
}


export function usePaymentStatus(paymentId: string | null, intervalMs = 2_000) {
  return useQuery({
    queryKey: paymentId ? queryKeys.paymentStatus(paymentId) : ["payment", "noop"],
    queryFn: () => api.paymentStatus(paymentId!),
    enabled: !!paymentId,
    refetchInterval: (q) => {
      const data = q.state.data
      if (data?.status === "succeeded" || data?.status === "failed" || data?.status === "expired") {
        return false
      }
      return intervalMs
    },
  })
}
