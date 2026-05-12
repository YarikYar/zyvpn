import { useState } from "react"
import { AlertTriangle, Bug, FileText, Info, XOctagon } from "lucide-react"
import type { LucideIcon } from "lucide-react"

import { SegmentedToggle } from "@/components/ui/segmented-toggle"
import { useLocale } from "@/lib/i18n"
import { useAdminLogs } from "@/lib/queries"
import type { AdminLog } from "@/lib/api"
import { cn } from "@/lib/utils"

import {
  AdminEmpty,
  AdminError,
  AdminLoader,
  formatAdminDate,
} from "./shared"

type LogFilter = "all" | "info" | "warn" | "error" | "debug"

const levelIcon: Record<Exclude<LogFilter, "all">, LucideIcon> = {
  info: Info,
  warn: AlertTriangle,
  error: XOctagon,
  debug: Bug,
}

const levelTone: Record<Exclude<LogFilter, "all">, string> = {
  info: "bg-sky-500/15 text-sky-600 dark:text-sky-300",
  warn: "bg-amber-500/15 text-amber-600 dark:text-amber-300",
  error: "bg-rose-500/15 text-rose-600 dark:text-rose-300",
  debug: "bg-muted text-muted-foreground",
}

export function AdminLogsView() {
  const { t, locale } = useLocale()
  const [filter, setFilter] = useState<LogFilter>("all")

  const query = useAdminLogs({
    level: filter === "all" ? undefined : filter,
    limit: 100,
  })

  return (
    <div className="mt-6">
      <SegmentedToggle<LogFilter>
        value={filter}
        onChange={setFilter}
        options={[
          { value: "all", label: t.admin.logs.filterAll, icon: FileText },
          { value: "info", label: t.admin.logs.level.info, icon: Info },
          { value: "warn", label: t.admin.logs.level.warn, icon: AlertTriangle },
          { value: "error", label: t.admin.logs.level.error, icon: XOctagon },
        ]}
      />

      {query.isPending && <AdminLoader />}
      {query.error && <AdminError error={query.error} />}
      {query.data && query.data.length === 0 && (
        <AdminEmpty>{t.admin.logs.empty}</AdminEmpty>
      )}

      {query.data && query.data.length > 0 && (
        <ul className="mt-4 space-y-2">
          {query.data.map((log, i) => (
            <LogRow key={log.id ?? `${log.created_at ?? i}`} log={log} locale={locale} />
          ))}
        </ul>
      )}
    </div>
  )
}

function LogRow({
  log,
  locale,
}: {
  log: AdminLog
  locale: "en" | "ru"
}) {
  const lvl = ((log.level ?? "info") as string).toLowerCase()
  const known: Exclude<LogFilter, "all"> =
    lvl === "warn" || lvl === "error" || lvl === "debug" ? lvl : "info"
  const Icon = levelIcon[known]
  const tone = levelTone[known]
  const date = formatAdminDate(log.created_at, locale)

  return (
    <li className="bg-card border-border flex items-start gap-3 rounded-2xl border p-4">
      <span
        className={cn(
          "inline-flex size-9 shrink-0 items-center justify-center rounded-xl",
          tone,
        )}
      >
        <Icon className="size-4" strokeWidth={2.25} />
      </span>
      <div className="min-w-0 flex-1">
        <p className="text-foreground text-sm leading-snug">
          {log.message}
        </p>
        <p className="text-muted-foreground mt-1 truncate text-[11px] tabular-nums">
          {date}
          {log.source && <> · {log.source}</>}
          {log.user_id !== undefined && <> · ID {log.user_id}</>}
        </p>
      </div>
    </li>
  )
}
