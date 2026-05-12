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
  adminStats: ["admin", "stats"] as const,
  adminUsers: (params?: { search?: string; limit?: number; offset?: number }) =>
    ["admin", "users", params ?? {}] as const,
  adminUser: (id: number) => ["admin", "users", id] as const,
  adminBans: ["admin", "bans"] as const,
  adminBansIp: ["admin", "bans", "ip"] as const,
  adminPromos: ["admin", "promos"] as const,
  adminCashPending: ["admin", "cash", "pending"] as const,
  adminPlans: ["admin", "plans"] as const,
  adminServers: ["admin", "servers"] as const,
  adminSettings: ["admin", "settings"] as const,
  adminLogs: (params?: { limit?: number; offset?: number; level?: string }) =>
    ["admin", "logs", params ?? {}] as const,
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


export function useAdminStats() {
  return useQuery({
    queryKey: queryKeys.adminStats,
    queryFn: () => api.adminStats(),
    staleTime: 30_000,
  })
}

export function useAdminUsers(params?: {
  search?: string
  limit?: number
  offset?: number
}) {
  return useQuery({
    queryKey: queryKeys.adminUsers(params),
    queryFn: () => api.adminUsers(params),
    staleTime: 15_000,
  })
}

export function useAdminUser(id: number | null) {
  return useQuery({
    queryKey: queryKeys.adminUser(id ?? 0),
    queryFn: () => api.adminUser(id!),
    enabled: typeof id === "number" && id > 0,
  })
}

export function useAdminUserMutations(id: number | null) {
  const qc = useQueryClient()
  const invalidate = () => {
    if (typeof id === "number") {
      qc.invalidateQueries({ queryKey: queryKeys.adminUser(id) })
    }
    qc.invalidateQueries({ queryKey: ["admin", "users"] })
    qc.invalidateQueries({ queryKey: queryKeys.adminBans })
    qc.invalidateQueries({ queryKey: queryKeys.adminStats })
  }
  const balanceSet = useMutation({
    mutationFn: (body: { amount: number; reason?: string }) =>
      api.adminUserBalanceSet(id!, body),
    onSuccess: invalidate,
  })
  const balanceAdd = useMutation({
    mutationFn: (body: { amount: number; reason?: string }) =>
      api.adminUserBalanceAdd(id!, body),
    onSuccess: invalidate,
  })
  const subscriptionExtend = useMutation({
    mutationFn: (body: { days: number }) =>
      api.adminUserSubscriptionExtend(id!, body),
    onSuccess: invalidate,
  })
  const subscriptionCancel = useMutation({
    mutationFn: () => api.adminUserSubscriptionCancel(id!),
    onSuccess: invalidate,
  })
  const cashPayment = useMutation({
    mutationFn: (body: { plan_id: string; amount_rub?: number; note?: string }) =>
      api.adminUserCashPayment(id!, body),
    onSuccess: invalidate,
  })
  const ban = useMutation({
    mutationFn: (body?: { reason?: string }) => api.adminUserBan(id!, body),
    onSuccess: invalidate,
  })
  const unban = useMutation({
    mutationFn: () => api.adminUserUnban(id!),
    onSuccess: invalidate,
  })
  return {
    balanceSet,
    balanceAdd,
    subscriptionExtend,
    subscriptionCancel,
    cashPayment,
    ban,
    unban,
  }
}

export function useAdminBans() {
  return useQuery({
    queryKey: queryKeys.adminBans,
    queryFn: () => api.adminBans(),
  })
}

export function useAdminBansIp() {
  return useQuery({
    queryKey: queryKeys.adminBansIp,
    queryFn: () => api.adminBansIp(),
  })
}

export function useAdminUnbanIp() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.adminUnbanIp,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.adminBansIp })
    },
  })
}

export function useAdminPromos() {
  return useQuery({
    queryKey: queryKeys.adminPromos,
    queryFn: () => api.adminPromos(),
  })
}

export function useAdminPromoCreate() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.adminPromoCreate,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.adminPromos })
    },
  })
}

export function useAdminPromoBulk() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.adminPromoBulk,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.adminPromos })
    },
  })
}

export function useAdminPromoDeactivate() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.adminPromoDeactivate,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.adminPromos })
    },
  })
}

export function useAdminCashPending() {
  return useQuery({
    queryKey: queryKeys.adminCashPending,
    queryFn: () => api.adminCashPending(),
    refetchInterval: 15_000,
  })
}

export function useAdminCashApprove() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.adminCashApprove,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.adminCashPending })
      qc.invalidateQueries({ queryKey: queryKeys.adminStats })
    },
  })
}

export function useAdminCashReject() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason?: string }) =>
      api.adminCashReject(id, reason ? { reason } : undefined),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.adminCashPending })
      qc.invalidateQueries({ queryKey: queryKeys.adminStats })
    },
  })
}

export function useAdminPlans() {
  return useQuery({
    queryKey: queryKeys.adminPlans,
    queryFn: () => api.adminPlans(),
  })
}

export function useAdminPlanCreate() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.adminPlanCreate,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.adminPlans })
      qc.invalidateQueries({ queryKey: queryKeys.plans })
    },
  })
}

export function useAdminPlanUpdate() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      id,
      input,
    }: {
      id: string
      input: Parameters<typeof api.adminPlanUpdate>[1]
    }) => api.adminPlanUpdate(id, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.adminPlans })
      qc.invalidateQueries({ queryKey: queryKeys.plans })
    },
  })
}

export function useAdminPlanDelete() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.adminPlanDelete,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.adminPlans })
      qc.invalidateQueries({ queryKey: queryKeys.plans })
    },
  })
}

export function useAdminServers() {
  return useQuery({
    queryKey: queryKeys.adminServers,
    queryFn: () => api.adminServers(),
  })
}

export function useAdminServerCreate() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.adminServerCreate,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.adminServers })
      qc.invalidateQueries({ queryKey: queryKeys.servers })
    },
  })
}

export function useAdminServerUpdate() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      id,
      input,
    }: {
      id: string
      input: Parameters<typeof api.adminServerUpdate>[1]
    }) => api.adminServerUpdate(id, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.adminServers })
      qc.invalidateQueries({ queryKey: queryKeys.servers })
    },
  })
}

export function useAdminServerDelete() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.adminServerDelete,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.adminServers })
      qc.invalidateQueries({ queryKey: queryKeys.servers })
    },
  })
}

export function useAdminServerTest() {
  return useMutation({
    mutationFn: api.adminServerTest,
  })
}

export function useAdminSettings() {
  return useQuery({
    queryKey: queryKeys.adminSettings,
    queryFn: () => api.adminSettings(),
  })
}

export function useAdminSettingMutations() {
  const qc = useQueryClient()
  const invalidate = () => {
    qc.invalidateQueries({ queryKey: queryKeys.adminSettings })
  }
  const topupBonus = useMutation({
    mutationFn: api.adminSettingTopupBonus,
    onSuccess: invalidate,
  })
  const referralBonus = useMutation({
    mutationFn: api.adminSettingReferralBonus,
    onSuccess: invalidate,
  })
  const referralBonusDays = useMutation({
    mutationFn: api.adminSettingReferralBonusDays,
    onSuccess: invalidate,
  })
  const regionSwitchPrice = useMutation({
    mutationFn: api.adminSettingRegionSwitchPrice,
    onSuccess: invalidate,
  })
  return { topupBonus, referralBonus, referralBonusDays, regionSwitchPrice }
}

export function useAdminLogs(params?: {
  limit?: number
  offset?: number
  level?: string
}) {
  return useQuery({
    queryKey: queryKeys.adminLogs(params),
    queryFn: () => api.adminLogs(params),
    refetchInterval: 30_000,
  })
}
