import type { ComponentType, SVGProps } from "react"
import { useMemo, useState } from "react"
import {
  Activity,
  AlertTriangle,
  ArrowRight,
  BarChart3,
  Calendar,
  CheckCircle2,
  Crown,
  Gift,
  Globe2,
  Loader2,
  Moon,
  Shield,
  Sparkles,
  Sun,
  Sunrise,
  Sunset,
  type LucideIcon,
} from "lucide-react"

import { SectionHeading } from "@/components/sections/section-heading"
import { ServerListRow } from "@/components/server-list-row"
import { ServerChipsRow } from "@/components/server-chip"
import { SubscriptionUrlCard } from "@/components/subscription-url-card"
import { flagFor } from "@/lib/flags"
import { useLocale, type Locale } from "@/lib/i18n"
import {
  useMe,
  useServerIncidents,
  useServerUptimeDaily,
  useSubscriptionStatus,
} from "@/lib/queries"
import type {
  DailyUptimePoint,
  ServerEntry,
  ServerIncident,
  SubscriptionStatus,
} from "@/lib/api"
import { cn } from "@/lib/utils"

type TimeOfDay = "morning" | "afternoon" | "evening" | "night"

function getTimeOfDayKey(date = new Date()): TimeOfDay {
  const hour = date.getHours()
  if (hour >= 5 && hour < 12) return "morning"
  if (hour >= 12 && hour < 17) return "afternoon"
  if (hour >= 17 && hour < 22) return "evening"
  return "night"
}

const timeIcons: Record<TimeOfDay, ComponentType<SVGProps<SVGSVGElement>>> = {
  morning: Sunrise,
  afternoon: Sun,
  evening: Sunset,
  night: Moon,
}

const localeMap: Record<Locale, string> = {
  en: "en-US",
  ru: "ru-RU",
}

function makeDateFmt(locale: Locale) {
  return new Intl.DateTimeFormat(localeMap[locale], {
    month: "short",
    day: "numeric",
    year: "numeric",
  })
}

function makeShortDateFmt(locale: Locale) {
  return new Intl.DateTimeFormat(localeMap[locale], {
    month: "short",
    day: "numeric",
  })
}

function parseDate(value: string | undefined | null): Date | null {
  if (!value) return null
  const d = new Date(value)
  return Number.isNaN(d.getTime()) ? null : d
}

function safeFormat(fmt: Intl.DateTimeFormat, date: Date | null): string {
  return date ? fmt.format(date) : ""
}

type PlanMeta = {
  Icon: LucideIcon
  gradient: string
  iconColor: string
}

const planMeta: Record<string, PlanMeta> = {
  friend: {
    Icon: Gift,
    gradient: "from-amber-500/12 via-orange-500/8",
    iconColor: "text-amber-500/20",
  },
  starter: {
    Icon: Sparkles,
    gradient: "from-sky-500/12 via-blue-500/8",
    iconColor: "text-sky-500/20",
  },
  basic: {
    Icon: Shield,
    gradient: "from-emerald-500/12 via-teal-500/8",
    iconColor: "text-emerald-500/20",
  },
  pro: {
    Icon: Crown,
    gradient: "from-violet-500/12 via-fuchsia-500/8",
    iconColor: "text-violet-500/20",
  },
}

type DayStatus = "ok" | "minor" | "major"

type UptimeDay = {
  date: Date
  status: DayStatus
  fraction: number
}

function normalizeUptime(raw: number): number {
  if (!Number.isFinite(raw) || raw < 0) return 0
  return raw > 1.5 ? raw / 100 : raw
}

function uptimeStatus(fraction: number): DayStatus {
  if (fraction >= 0.999) return "ok"
  if (fraction >= 0.95) return "minor"
  return "major"
}

function mapUptimePoint(point: DailyUptimePoint): UptimeDay | null {
  const date = parseDate(point.date)
  if (!date) return null
  const fraction = normalizeUptime(point.uptime)
  return { date, fraction, status: uptimeStatus(fraction) }
}

function distinctRegionCount(servers: ServerEntry[] | undefined): number {
  if (!servers || servers.length === 0) return 0
  const codes = new Set<string>()
  for (const s of servers) codes.add((s.country ?? "").toUpperCase())
  return codes.size
}

export function MainSection({
  onShowKey,
}: {
  onShowKey?: () => void
}) {
  const { t } = useLocale()
  const meQuery = useMe()
  const statusQuery = useSubscriptionStatus()
  const [timeKey] = useState<TimeOfDay>(() => getTimeOfDayKey())
  const Icon = timeIcons[timeKey]
  const greeting = t.main.greetings[timeKey]
  const handle = meQuery.data?.username
    ? `@${meQuery.data.username}`
    : (meQuery.data?.first_name ?? "")

  const status = statusQuery.data
  const sub = status?.subscription ?? null
  const showActive = !!status?.active && !!sub
  const servers = sub?.servers ?? []
  const subscriptionUrl = sub?.subscription_url ?? ""

  const incidentsQuery = useServerIncidents({ hours: 720, limit: 10 })

  const incidents = incidentsQuery.data ?? []
  const showUptime = showActive && servers.length > 0
  const showIncidents = incidents.length > 0

  return (
    <section className="px-6 pb-16">
      <SectionHeading icon={Icon}>{greeting},</SectionHeading>
      <p className="text-muted-foreground mt-1 text-3xl leading-[1.1] font-medium tracking-tight sm:text-4xl">
        {handle}
      </p>

      {showActive && status && (
        <>
          <h3 className="text-foreground mt-10 text-lg font-semibold tracking-tight">
            {t.main.yourSubscription}
          </h3>
          <ActiveSubscriptionBlock
            status={status}
            onShowKey={onShowKey}
          />
        </>
      )}

      {showActive && subscriptionUrl && (
        <div className="mt-4">
          <SubscriptionUrlCard url={subscriptionUrl} />
        </div>
      )}

      {showActive && servers.length > 0 && (
        <>
          <div className="mt-10 flex items-baseline justify-between gap-3">
            <h3 className="text-foreground text-lg font-semibold tracking-tight">
              {t.servers.title}
            </h3>
            <p className="text-muted-foreground text-xs font-medium tracking-tight tabular-nums">
              {t.subscription.serversList.count(servers.length)}
              {distinctRegionCount(servers) > 1 &&
                ` · ${t.subscription.serversList.regionCount(distinctRegionCount(servers))}`}
            </p>
          </div>
          <p className="text-muted-foreground mt-1 text-sm">
            {t.servers.intro}
          </p>
          <ul className="mt-4 space-y-3">
            {servers.map((s) => (
              <li key={s.id}>
                <ServerListRow server={s} />
              </li>
            ))}
          </ul>
        </>
      )}

      {showActive && servers.length === 0 && (
        <p className="text-muted-foreground mt-6 text-sm">
          {t.servers.empty}
        </p>
      )}

      {!showActive && (
        <NoSubscriptionBlock onShowKey={onShowKey} />
      )}

      {showUptime && (
        <>
          <h3 className="text-foreground mt-10 text-lg font-semibold tracking-tight">
            {t.main.serverUptime}
          </h3>
          <ul className="mt-3 space-y-2.5">
            {servers.map((s) => (
              <li key={s.id}>
                <ServerUptimeCard server={s} />
              </li>
            ))}
          </ul>
        </>
      )}

      {showIncidents && (
        <>
          <h3 className="text-foreground mt-10 text-lg font-semibold tracking-tight">
            {t.main.recentIncidents}
          </h3>
          <ul className="mt-3 space-y-2">
            {incidents.map((inc, i) => (
              <IncidentRow key={`${inc.server_id ?? "?"}-${inc.started_at}-${i}`} incident={inc} />
            ))}
          </ul>
        </>
      )}
    </section>
  )
}

function NoSubscriptionBlock({
  onShowKey,
}: {
  onShowKey?: () => void
}) {
  const { t } = useLocale()
  return (
    <div className="border-border bg-card relative mt-10 overflow-hidden rounded-2xl border bg-gradient-to-br from-sky-500/12 via-blue-500/8 to-transparent p-6 shadow-lg">
      <Globe2
        aria-hidden
        strokeWidth={1.2}
        className="text-sky-500/20 pointer-events-none absolute -right-3 -bottom-8 size-36 sm:-right-4 sm:-bottom-10 sm:size-44"
      />
      <p className="text-muted-foreground inline-flex items-center gap-2 text-xs font-semibold tracking-[0.18em] uppercase">
        <Sparkles className="size-3.5" strokeWidth={2.25} />
        {t.main.noSubscriptionTitle}
      </p>
      <p className="text-foreground relative mt-3 max-w-[80%] text-lg font-semibold leading-snug tracking-tight sm:text-xl">
        {t.main.noSubscriptionSubtitle}
      </p>
      <button
        type="button"
        onClick={onShowKey}
        className="bg-foreground text-background hover:bg-foreground/90 relative mt-5 inline-flex h-12 items-center justify-center gap-2 rounded-2xl px-5 text-sm font-semibold tracking-tight transition-colors"
      >
        {t.main.noSubscriptionCta}
        <ArrowRight className="size-4" strokeWidth={2.5} />
      </button>
    </div>
  )
}

function ActiveSubscriptionBlock({
  status,
  onShowKey,
}: {
  status: SubscriptionStatus
  onShowKey?: () => void
}) {
  const { t, locale } = useLocale()
  const sub = status.subscription
  if (!sub) return null
  const meta = planMeta[sub.plan_id] ?? planMeta.pro
  const startedAt = parseDate(sub.started_at)
  const endsAt = parseDate(sub.ends_at)
  const totalDays =
    startedAt && endsAt
      ? Math.max(
          1,
          Math.round(
            (endsAt.getTime() - startedAt.getTime()) / (1000 * 60 * 60 * 24),
          ),
        )
      : Math.max(1, status.days_remaining)
  const remainingDays = status.days_remaining
  const trafficLabel =
    status.traffic_gb.limit === null
      ? t.common.unlimited
      : `${status.traffic_gb.limit} ${t.common.gb}`
  const endDateLabel = safeFormat(makeDateFmt(locale), endsAt)
  const serverCount = sub.servers?.length ?? 0
  const regionCount = distinctRegionCount(sub.servers)

  return (
    <div
      className={cn(
        "border-border bg-card relative mt-3 overflow-hidden rounded-2xl border bg-gradient-to-br to-transparent p-5 shadow-lg",
        meta.gradient,
      )}
    >
      <meta.Icon
        aria-hidden
        strokeWidth={1.2}
        className={cn(
          "pointer-events-none absolute -right-3 -bottom-8 size-36 sm:-right-4 sm:-bottom-10 sm:size-44",
          meta.iconColor,
        )}
      />

      <div className="flex items-center justify-between gap-3">
        <p className="text-muted-foreground inline-flex items-center gap-2 text-xs font-semibold tracking-[0.18em] uppercase">
          <span aria-hidden className="relative inline-flex size-2">
            <span className="absolute inset-0 rounded-full bg-emerald-500" />
            <span className="absolute inset-0 animate-ping rounded-full bg-emerald-500/60" />
          </span>
          {t.main.activePrefix} · {sub.plan_name}
        </p>
        {onShowKey && (
          <button
            type="button"
            onClick={onShowKey}
            className="text-foreground/85 hover:text-foreground inline-flex shrink-0 items-center gap-1 text-xs font-semibold tracking-tight transition-colors"
          >
            {t.subscription.renew}
            <ArrowRight className="size-3.5" strokeWidth={2.5} />
          </button>
        )}
      </div>

      <div className="mt-2 flex items-end gap-2">
        <p className="text-foreground text-4xl font-bold tabular-nums sm:text-5xl">
          {remainingDays}
        </p>
        <p className="text-muted-foreground mb-1.5 text-sm font-medium">
          {t.main.daysLeft}
        </p>
      </div>

      <div className="mt-4 grid grid-cols-2 gap-3">
        <StatCard
          icon={Calendar}
          label={t.main.term}
          value={endDateLabel}
          meta={t.main.daysUsage(remainingDays, totalDays)}
        />
        <StatCard
          icon={BarChart3}
          label={t.main.traffic}
          value={`${status.traffic_gb.used.toFixed(1)} ${t.common.gb}`}
          meta={trafficLabel}
        />
      </div>

      {serverCount > 0 && (
        <div className="mt-4">
          <p className="text-muted-foreground text-[10px] font-semibold tracking-[0.16em] uppercase">
            {t.subscription.serversList.title} ·{" "}
            <span className="tabular-nums">
              {t.subscription.serversList.count(serverCount)}
              {regionCount > 1 &&
                ` · ${t.subscription.serversList.regionCount(regionCount)}`}
            </span>
          </p>
          <ServerChipsRow
            servers={sub.servers}
            className="mt-2"
            max={6}
          />
        </div>
      )}
    </div>
  )
}

function StatCard({
  icon: Icon,
  label,
  value,
  meta,
}: {
  icon: LucideIcon
  label: string
  value: string
  meta?: string
}) {
  return (
    <div className="border-border/60 bg-background relative rounded-xl border p-3">
      <div className="flex items-center gap-1.5">
        <Icon className="text-muted-foreground size-3.5" strokeWidth={2} />
        <p className="text-muted-foreground text-[10px] font-semibold tracking-[0.16em] uppercase">
          {label}
        </p>
      </div>
      <p className="text-foreground mt-1 text-base font-semibold tabular-nums">
        {value}
      </p>
      {meta && (
        <p className="text-muted-foreground mt-0.5 text-[11px] tabular-nums">
          {meta}
        </p>
      )}
    </div>
  )
}

const dayBarTone: Record<DayStatus, string> = {
  ok: "bg-emerald-500",
  minor: "bg-amber-500",
  major: "bg-red-500",
}

function ServerUptimeCard({ server }: { server: ServerEntry }) {
  const { t, locale } = useLocale()
  const uptimeQuery = useServerUptimeDaily(server.id, 30)
  const days = useMemo<UptimeDay[]>(
    () =>
      (uptimeQuery.data?.days ?? [])
        .map(mapUptimePoint)
        .filter((d): d is UptimeDay => d !== null),
    [uptimeQuery.data],
  )

  const code = (server.country ?? "").toUpperCase()
  const Flag = flagFor(code)
  const meta = code
    ? t.servers.countries[code as keyof typeof t.servers.countries]
    : undefined
  const country = meta?.country ?? code
  const city =
    meta?.city ?? server.city?.trim() ?? server.name?.trim() ?? ""

  if (uptimeQuery.isPending) {
    return (
      <div className="border-border bg-card flex items-center gap-3 rounded-2xl border p-4">
        <span className="bg-muted size-8 shrink-0 animate-pulse rounded-lg" />
        <div className="min-w-0 flex-1 space-y-1.5">
          <span className="bg-muted block h-3 w-24 animate-pulse rounded" />
          <span className="bg-muted/70 block h-2 w-16 animate-pulse rounded" />
        </div>
        <Loader2 className="text-muted-foreground size-4 animate-spin" />
      </div>
    )
  }

  if (days.length === 0) {
    return null
  }

  const okCount = days.filter((d) => d.status === "ok").length
  const totalFraction =
    days.reduce((acc, d) => acc + d.fraction, 0) / Math.max(1, days.length)
  const uptimePct = totalFraction * 100
  const hasIssues = okCount < days.length
  const firstDate = days[0]?.date
  const dateFmt = makeShortDateFmt(locale)

  const gradient = hasIssues
    ? "from-amber-500/10 via-amber-500/3"
    : "from-emerald-500/10 via-emerald-500/3"

  const statusLabel: Record<DayStatus, string> = {
    ok: t.main.operational,
    minor: t.main.minorIncident,
    major: t.main.majorIncident,
  }

  return (
    <div
      className={cn(
        "border-border bg-card relative overflow-hidden rounded-2xl border bg-gradient-to-br to-transparent p-4 shadow-sm",
        gradient,
      )}
    >
      <div className="relative flex items-center justify-between gap-3">
        <div className="inline-flex min-w-0 items-center gap-2.5">
          {Flag ? (
            <Flag aria-hidden className="h-5 w-[30px] shrink-0 rounded-[3px] object-cover" />
          ) : (
            <Activity className="text-muted-foreground size-4 shrink-0" strokeWidth={2.25} />
          )}
          <div className="min-w-0">
            <p className="text-foreground truncate text-sm font-semibold tracking-tight">
              {country}
            </p>
            {city && (
              <p className="text-muted-foreground -mt-0.5 truncate text-[11px]">
                {city}
              </p>
            )}
          </div>
        </div>
        <p
          className={cn(
            "shrink-0 text-sm font-semibold tabular-nums",
            hasIssues
              ? "text-amber-600 dark:text-amber-400"
              : "text-emerald-600 dark:text-emerald-400",
          )}
        >
          {uptimePct.toFixed(1)}%
        </p>
      </div>

      <div className="relative mt-3 flex items-stretch gap-[2px]">
        {days.map((d, i) => (
          <span
            key={i}
            title={`${safeFormat(dateFmt, d.date)} · ${statusLabel[d.status]} · ${(d.fraction * 100).toFixed(2)}%`}
            className={cn(
              "h-5 flex-1 rounded-[2px] transition-colors duration-300 ease-out",
              dayBarTone[d.status],
              d.status === "ok" && "opacity-90",
            )}
          />
        ))}
      </div>

      <div className="text-muted-foreground relative mt-2 flex justify-between text-[10px] tabular-nums">
        <span>{safeFormat(dateFmt, firstDate ?? null)}</span>
        <span>{t.common.today}</span>
      </div>
    </div>
  )
}

function IncidentRow({ incident }: { incident: ServerIncident }) {
  const { t, locale } = useLocale()
  const startedAt = parseDate(incident.started_at)
  const endedAt = parseDate(incident.ended_at)
  const resolved = endedAt !== null
  const Icon = resolved ? CheckCircle2 : AlertTriangle
  const gradient = resolved
    ? "from-emerald-500/10 via-emerald-500/3"
    : "from-amber-500/12 via-amber-500/4"
  const dateLabel = safeFormat(makeDateFmt(locale), startedAt)

  const title = incident.server_name?.trim() || t.main.incidentUnknownServer
  const subtitleParts: string[] = []
  if (incident.country) subtitleParts.push(incident.country)
  if (dateLabel) subtitleParts.push(dateLabel)
  if (resolved) subtitleParts.push(t.main.resolved)

  const durationLabel = resolved
    ? incident.duration_seconds !== undefined
      ? t.main.incidentDuration(incident.duration_seconds)
      : ""
    : t.main.incidentOngoing

  return (
    <li
      className={cn(
        "border-border bg-card relative flex items-start gap-3 overflow-hidden rounded-2xl border bg-gradient-to-br to-transparent p-4 shadow-md",
        gradient,
      )}
    >
      <span
        className={cn(
          "inline-flex size-8 shrink-0 items-center justify-center rounded-lg",
          resolved
            ? "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400"
            : "bg-amber-500/15 text-amber-600 dark:text-amber-400",
        )}
      >
        <Icon className="size-4" strokeWidth={2.25} />
      </span>
      <div className="min-w-0 flex-1">
        <div className="flex items-baseline justify-between gap-2">
          <p className="text-foreground truncate text-sm font-semibold tracking-tight">
            {title}
          </p>
          {durationLabel && (
            <p className="text-muted-foreground shrink-0 text-xs font-medium tabular-nums">
              {durationLabel}
            </p>
          )}
        </div>
        <p className="text-muted-foreground mt-0.5 text-xs tabular-nums">
          {subtitleParts.join(" · ")}
        </p>
      </div>
    </li>
  )
}
