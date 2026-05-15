import { useState } from "react"
import { Check, Copy, Globe2, KeyRound } from "lucide-react"
import { AnimatePresence, motion } from "motion/react"

import type { ServerEntry } from "@/lib/api"
import {
  flagFor,
  loadFromPing,
  pingTone,
  regionTintFor,
} from "@/lib/flags"
import { useLocale } from "@/lib/i18n"
import { copyToClipboard, triggerHaptic } from "@/lib/telegram"
import { cn } from "@/lib/utils"

const HOVER_SPRING = {
  type: "spring",
  stiffness: 360,
  damping: 26,
  mass: 0.6,
} as const

const FEEDBACK_SPRING = {
  type: "spring",
  stiffness: 420,
  damping: 28,
  mass: 0.8,
} as const

const loadDot: Record<"low" | "medium" | "high", string> = {
  low: "bg-emerald-400",
  medium: "bg-amber-300",
  high: "bg-red-300",
}

function isOffline(s: ServerEntry): boolean {
  return s.status === "offline"
}

export function ServerListRow({ server }: { server: ServerEntry }) {
  const { t } = useLocale()
  const [copied, setCopied] = useState(false)
  const code = (server.country ?? "").toUpperCase()
  const Flag = flagFor(code)
  const meta = code
    ? t.servers.countries[code as keyof typeof t.servers.countries]
    : undefined
  const country = meta?.country ?? code
  const city =
    meta?.city ?? server.city?.trim() ?? server.name?.trim() ?? ""
  const offline = isOffline(server)
  const ping = server.ping_ms ?? 0
  const load = loadFromPing(ping)
  const hasKey = !!server.connection_key

  const handleCopyKey = async () => {
    if (!server.connection_key) return
    const ok = await copyToClipboard(server.connection_key)
    if (!ok) return
    triggerHaptic({ type: "notification", status: "success" })
    setCopied(true)
    window.setTimeout(() => setCopied(false), 1600)
  }

  return (
    <motion.div
      whileHover={hasKey && !offline ? { y: -2 } : undefined}
      transition={HOVER_SPRING}
      className={cn(
        "group bg-card border-border relative w-full overflow-hidden rounded-2xl border bg-gradient-to-br to-transparent p-5 shadow-lg",
        regionTintFor(code),
        offline && "opacity-60 saturate-50",
      )}
    >
      {Flag ? (
        <Flag
          aria-hidden
          className="pointer-events-none absolute -right-3 -bottom-3 h-20 aspect-[3/2] rounded-xl opacity-25 transition-transform duration-500 ease-out group-hover:scale-105 group-hover:rotate-3 sm:-right-4 sm:-bottom-4 sm:h-24"
        />
      ) : (
        <Globe2
          aria-hidden
          strokeWidth={1.2}
          className="text-muted-foreground/20 pointer-events-none absolute -right-3 -bottom-3 size-20 sm:-right-4 sm:-bottom-4 sm:size-24"
        />
      )}

      <div className="relative flex items-start justify-between gap-4">
        <div className="min-w-0 flex-1">
          <p className="text-foreground text-xl font-semibold tracking-tight sm:text-2xl">
            {country}
          </p>
          {city && (
            <p className="text-muted-foreground mt-0.5 text-sm sm:text-[15px]">
              {city}
            </p>
          )}
        </div>
        <div className="shrink-0 text-right">
          {offline ? (
            <p className="inline-flex items-center gap-1.5 text-sm font-semibold tracking-tight text-rose-500 dark:text-rose-400">
              <span aria-hidden className="size-1.5 rounded-full bg-rose-500" />
              {t.servers.status.offline}
            </p>
          ) : (
            <>
              <p
                className={cn(
                  "text-2xl font-semibold tabular-nums sm:text-3xl",
                  pingTone(ping),
                )}
              >
                {ping}
                <span className="text-muted-foreground ml-1 text-xs font-medium tracking-wide sm:text-sm">
                  {t.common.msUnit}
                </span>
              </p>
              <p className="text-muted-foreground mt-0.5 inline-flex items-center gap-1.5 text-sm">
                <span
                  aria-hidden
                  className={cn("size-1.5 rounded-full", loadDot[load])}
                />
                {t.servers.load[load]}
              </p>
            </>
          )}
        </div>
      </div>

      {hasKey && (
        <button
          type="button"
          onClick={handleCopyKey}
          disabled={offline}
          className="border-border bg-background hover:bg-muted/60 text-foreground relative mt-4 inline-flex w-full items-center justify-center gap-2 rounded-xl border px-3 py-2 text-xs font-semibold tracking-tight transition-colors disabled:cursor-not-allowed disabled:opacity-60 sm:text-sm"
        >
          <AnimatePresence mode="wait" initial={false}>
            {copied ? (
              <motion.span
                key="copied"
                initial={{ opacity: 0, scale: 0.7 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.7 }}
                transition={FEEDBACK_SPRING}
                className="inline-flex items-center gap-2"
              >
                <Check className="size-3.5" strokeWidth={2.5} />
                {t.servers.keyCopied}
              </motion.span>
            ) : (
              <motion.span
                key="copy"
                initial={{ opacity: 0, scale: 0.7 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.7 }}
                transition={FEEDBACK_SPRING}
                className="inline-flex items-center gap-2"
              >
                <KeyRound className="size-3.5" strokeWidth={2.25} />
                {t.servers.copyKey}
                <Copy className="size-3.5 opacity-60" strokeWidth={2} />
              </motion.span>
            )}
          </AnimatePresence>
        </button>
      )}
    </motion.div>
  )
}
