import type { ServerEntry } from "@/lib/api"
import { flagFor } from "@/lib/flags"
import { useLocale } from "@/lib/i18n"
import { cn } from "@/lib/utils"

export function ServerChip({
  server,
  showCity = false,
  className,
}: {
  server: Pick<ServerEntry, "country" | "city" | "name">
  showCity?: boolean
  className?: string
}) {
  const { t } = useLocale()
  const code = (server.country ?? "").toUpperCase()
  const Flag = flagFor(code)
  const meta = code
    ? t.servers.countries[code as keyof typeof t.servers.countries]
    : undefined
  const country = meta?.country ?? code
  const city = meta?.city ?? server.city?.trim() ?? server.name?.trim() ?? ""

  return (
    <span
      className={cn(
        "border-border bg-card text-foreground inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs font-medium tracking-tight",
        className,
      )}
      title={city ? `${country} — ${city}` : country}
    >
      {Flag ? (
        <Flag aria-hidden className="h-3 w-[18px] rounded-[3px] object-cover" />
      ) : (
        <span aria-hidden className="bg-muted size-2 rounded-full" />
      )}
      <span>{country}</span>
      {showCity && city && (
        <span className="text-muted-foreground">· {city}</span>
      )}
    </span>
  )
}

export function ServerChipsRow({
  servers,
  max = 4,
  className,
}: {
  servers: Pick<ServerEntry, "id" | "country" | "city" | "name">[]
  max?: number
  className?: string
}) {
  const { t } = useLocale()
  const visible = servers.slice(0, max)
  const extra = servers.length - visible.length
  return (
    <div className={cn("flex flex-wrap gap-1.5", className)}>
      {visible.map((s) => (
        <ServerChip key={s.id} server={s} />
      ))}
      {extra > 0 && (
        <span className="border-border bg-muted text-muted-foreground inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-medium tracking-tight">
          {t.subscription.moreServers(extra)}
        </span>
      )}
    </div>
  )
}
