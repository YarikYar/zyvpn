import { useMemo } from "react"
import { Check, Globe2, Loader2 } from "lucide-react"

import type { Server } from "@/lib/api"
import { flagFor, regionTintFor } from "@/lib/flags"
import { useLocale } from "@/lib/i18n"
import { useAdminServers } from "@/lib/queries"
import { cn } from "@/lib/utils"

type Group = {
  code: string
  country: string
  servers: Server[]
}

function groupByCountry(servers: Server[]): Group[] {
  const byCode = new Map<string, Server[]>()
  for (const s of servers) {
    const code = (s.country ?? "").toUpperCase() || "??"
    const arr = byCode.get(code) ?? []
    arr.push(s)
    byCode.set(code, arr)
  }
  return Array.from(byCode.entries())
    .map(([code, list]) => ({
      code,
      country: code,
      servers: list.slice().sort((a, b) =>
        (a.city ?? a.name ?? "").localeCompare(b.city ?? b.name ?? ""),
      ),
    }))
    .sort((a, b) => a.code.localeCompare(b.code))
}

export function ServerPicker({
  selectedIds,
  onChange,
}: {
  selectedIds: string[]
  onChange: (ids: string[]) => void
}) {
  const { t } = useLocale()
  const serversQuery = useAdminServers()
  const selected = useMemo(() => new Set(selectedIds), [selectedIds])

  const groups = useMemo(
    () => groupByCountry(serversQuery.data ?? []),
    [serversQuery.data],
  )

  const regionsSelected = useMemo(() => {
    const codes = new Set<string>()
    for (const g of groups) {
      if (g.servers.some((s) => selected.has(s.id))) codes.add(g.code)
    }
    return codes.size
  }, [groups, selected])

  const toggle = (id: string) => {
    const next = new Set(selected)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    onChange(Array.from(next))
  }

  const toggleRegion = (group: Group) => {
    const allSelected = group.servers.every((s) => selected.has(s.id))
    const next = new Set(selected)
    if (allSelected) {
      for (const s of group.servers) next.delete(s.id)
    } else {
      for (const s of group.servers) next.add(s.id)
    }
    onChange(Array.from(next))
  }

  if (serversQuery.isPending) {
    return (
      <div className="border-border bg-background flex items-center justify-center gap-2 rounded-xl border px-3.5 py-6 text-xs">
        <Loader2 className="text-muted-foreground size-4 animate-spin" />
        <span className="text-muted-foreground">
          {t.admin.plans.serverPicker.loading}
        </span>
      </div>
    )
  }

  if (!serversQuery.data || serversQuery.data.length === 0) {
    return (
      <div className="border-border bg-background rounded-xl border px-3.5 py-4 text-center text-xs text-muted-foreground">
        {t.admin.plans.serverPicker.none}
      </div>
    )
  }

  return (
    <div className="space-y-3">
      <div className="text-muted-foreground border-border/60 bg-background flex items-center justify-between gap-2 rounded-xl border px-3.5 py-2 text-xs">
        <span>{t.admin.plans.serverPicker.intro}</span>
        <span className="text-foreground shrink-0 font-semibold tabular-nums">
          {t.admin.plans.serverPicker.summary(selected.size, regionsSelected)}
        </span>
      </div>

      <div className="space-y-2">
        {groups.map((group) => {
          const code = group.code
          const Flag = flagFor(code)
          const allSelected = group.servers.every((s) => selected.has(s.id))
          const countryLabel =
            t.servers.countries[code as keyof typeof t.servers.countries]
              ?.country ?? code
          const selectedHere = group.servers.filter((s) => selected.has(s.id)).length
          return (
            <div
              key={code}
              className={cn(
                "border-border bg-card overflow-hidden rounded-2xl border bg-gradient-to-br to-transparent",
                regionTintFor(code),
              )}
            >
              <div className="flex items-center justify-between gap-2 px-3.5 py-2.5">
                <div className="inline-flex items-center gap-2 min-w-0">
                  {Flag ? (
                    <Flag aria-hidden className="h-4 w-6 rounded-[3px] object-cover" />
                  ) : (
                    <Globe2 className="text-muted-foreground size-4" strokeWidth={2} />
                  )}
                  <span className="text-foreground truncate text-sm font-semibold tracking-tight">
                    {countryLabel}
                  </span>
                  <span className="text-muted-foreground shrink-0 text-[10px] font-semibold tracking-[0.16em] uppercase">
                    {selectedHere}/{group.servers.length}
                  </span>
                </div>
                <button
                  type="button"
                  onClick={() => toggleRegion(group)}
                  className="text-foreground/70 hover:text-foreground shrink-0 text-[11px] font-semibold tracking-tight"
                >
                  {allSelected
                    ? t.admin.plans.serverPicker.clearAll
                    : t.admin.plans.serverPicker.selectAll}
                </button>
              </div>
              <ul className="border-border/60 border-t">
                {group.servers.map((s) => {
                  const isSelected = selected.has(s.id)
                  const city =
                    s.city?.trim() || s.name?.trim() || s.id
                  return (
                    <li key={s.id} className="border-border/40 border-b last:border-b-0">
                      <button
                        type="button"
                        onClick={() => toggle(s.id)}
                        aria-pressed={isSelected}
                        className={cn(
                          "hover:bg-muted/40 flex w-full items-center justify-between gap-3 px-3.5 py-2.5 text-left transition-colors",
                          isSelected && "bg-muted/30",
                        )}
                      >
                        <div className="min-w-0">
                          <p className="text-foreground truncate text-sm font-medium">
                            {city}
                          </p>
                          <p className="text-muted-foreground mt-0.5 truncate text-[11px] tabular-nums">
                            {s.id}
                          </p>
                        </div>
                        <span
                          className={cn(
                            "inline-flex size-5 shrink-0 items-center justify-center rounded-md border transition-colors",
                            isSelected
                              ? "border-foreground bg-foreground text-background"
                              : "border-border bg-background",
                          )}
                        >
                          {isSelected && <Check className="size-3.5" strokeWidth={2.75} />}
                        </span>
                      </button>
                    </li>
                  )
                })}
              </ul>
            </div>
          )
        })}
      </div>
    </div>
  )
}
