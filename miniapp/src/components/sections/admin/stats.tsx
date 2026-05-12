import {
  BarChart3,
  Banknote,
  Calendar,
  Gift,
  ServerCog,
  ShieldAlert,
  TrendingUp,
  UserCheck,
  UserPlus,
  Users,
} from "lucide-react"

import { useLocale } from "@/lib/i18n"
import { useAdminStats } from "@/lib/queries"
import { formatTonBalance } from "@/lib/format"

import { AdminError, AdminLoader, StatTile } from "./shared"

export function AdminStatsView() {
  const { t } = useLocale()
  const query = useAdminStats()

  if (query.isPending) return <AdminLoader />
  if (query.error) return <AdminError error={query.error} />

  const s = query.data
  const tonTotal = s?.revenue_ton_total ?? 0
  const ton30 = s?.revenue_ton_30d ?? 0
  const rub30 = s?.revenue_rub_30d ?? 0
  const fmtCount = (n: number | undefined) =>
    typeof n === "number" ? n.toLocaleString() : "—"
  const fmtRub = (n: number | undefined) =>
    typeof n === "number" ? `${Math.round(n).toLocaleString()} ₽` : "—"
  const fmtTon = (n: number | undefined) =>
    typeof n === "number" ? `${formatTonBalance(n)} TON` : "—"

  return (
    <div className="mt-6 grid grid-cols-2 gap-3">
      <StatTile
        icon={Users}
        label={t.admin.stats.usersTotal}
        value={fmtCount(s?.users_total)}
      />
      <StatTile
        icon={UserCheck}
        label={t.admin.stats.usersActive}
        value={fmtCount(s?.users_active)}
      />
      <StatTile
        icon={UserPlus}
        label={t.admin.stats.usersNewToday}
        value={fmtCount(s?.users_new_today)}
      />
      <StatTile
        icon={Calendar}
        label={t.admin.stats.subsActive}
        value={fmtCount(s?.subscriptions_active)}
      />
      <StatTile
        icon={TrendingUp}
        label={t.admin.stats.revenueTon}
        value={fmtTon(tonTotal)}
        meta={`${t.admin.stats.revenue30d}: ${fmtTon(ton30)}`}
      />
      <StatTile
        icon={BarChart3}
        label={t.admin.stats.revenueRub}
        value={fmtRub(rub30)}
        meta={t.admin.stats.revenue30d}
      />
      <StatTile
        icon={Banknote}
        label={t.admin.stats.cashPending}
        value={fmtCount(s?.payments_pending)}
      />
      <StatTile
        icon={Gift}
        label={t.admin.stats.trialUsed}
        value={fmtCount(s?.trial_used)}
      />
      <StatTile
        icon={ServerCog}
        label={t.admin.stats.serversOnline}
        value={
          typeof s?.servers_online === "number" &&
          typeof s?.servers_total === "number"
            ? `${s.servers_online} / ${s.servers_total}`
            : fmtCount(s?.servers_online)
        }
      />
      <StatTile
        icon={ShieldAlert}
        label={t.admin.stats.bansActive}
        value={fmtCount(s?.bans_active)}
      />
    </div>
  )
}
