import { Loader2 } from "lucide-react"
import type { ComponentType, ReactNode, SVGProps } from "react"

import { useLocale, type Locale } from "@/lib/i18n"
import { cn } from "@/lib/utils"

const localeMap: Record<Locale, string> = {
  en: "en-US",
  ru: "ru-RU",
}

export function formatAdminDate(value: string | undefined | null, locale: Locale): string {
  if (!value) return ""
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return ""
  return new Intl.DateTimeFormat(localeMap[locale], {
    day: "2-digit",
    month: "2-digit",
    year: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(d)
}

export function formatAdminDateShort(value: string | undefined | null, locale: Locale): string {
  if (!value) return ""
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return ""
  return new Intl.DateTimeFormat(localeMap[locale], {
    day: "2-digit",
    month: "short",
  }).format(d)
}

export function AdminLoader({ className }: { className?: string }) {
  return (
    <div className={cn("flex items-center justify-center py-10", className)}>
      <Loader2 className="text-muted-foreground size-5 animate-spin" />
    </div>
  )
}

export function AdminError({ error }: { error: unknown }) {
  const { t } = useLocale()
  const message =
    error instanceof Error ? error.message : t.admin.common.error
  return (
    <p className="mt-4 text-sm text-rose-500 dark:text-rose-400">{message}</p>
  )
}

export function AdminEmpty({ children }: { children: ReactNode }) {
  return (
    <p className="text-muted-foreground mt-6 text-sm">
      {children}
    </p>
  )
}

export function AdminCard({
  children,
  className,
}: {
  children: ReactNode
  className?: string
}) {
  return (
    <div className={cn("bg-card border-border rounded-2xl border p-4", className)}>
      {children}
    </div>
  )
}

export function FormField({
  label,
  help,
  children,
}: {
  label: string
  help?: string
  children: ReactNode
}) {
  return (
    <label className="block">
      <span className="text-muted-foreground text-[10px] font-semibold tracking-[0.18em] uppercase">
        {label}
      </span>
      <div className="mt-1.5">{children}</div>
      {help && (
        <p className="text-muted-foreground mt-1.5 text-xs leading-snug">
          {help}
        </p>
      )}
    </label>
  )
}

export function TextInput({
  className,
  ...props
}: React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      {...props}
      className={cn(
        "border-border/60 bg-background text-foreground placeholder:text-muted-foreground focus-visible:border-foreground/40 w-full rounded-xl border px-3.5 py-2.5 text-sm tabular-nums outline-none transition-colors",
        className,
      )}
    />
  )
}

export function TextArea({
  className,
  ...props
}: React.TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return (
    <textarea
      {...props}
      className={cn(
        "border-border/60 bg-background text-foreground placeholder:text-muted-foreground focus-visible:border-foreground/40 w-full resize-none rounded-xl border px-3.5 py-2.5 text-sm outline-none transition-colors",
        className,
      )}
    />
  )
}

type ToggleProps = {
  checked: boolean
  onChange: (next: boolean) => void
  disabled?: boolean
}

export function Toggle({ checked, onChange, disabled }: ToggleProps) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={cn(
        "relative inline-flex h-6 w-11 shrink-0 items-center rounded-full transition-colors",
        checked ? "bg-foreground" : "bg-muted",
        disabled && "opacity-50",
      )}
    >
      <span
        className={cn(
          "bg-background inline-block size-5 rounded-full shadow transition-transform",
          checked ? "translate-x-[22px]" : "translate-x-0.5",
        )}
      />
    </button>
  )
}

export function PrimaryButton({
  className,
  loading,
  children,
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & { loading?: boolean }) {
  return (
    <button
      {...props}
      disabled={props.disabled || loading}
      className={cn(
        "bg-foreground text-background hover:bg-foreground/90 disabled:bg-muted disabled:text-muted-foreground inline-flex h-11 items-center justify-center gap-2 rounded-xl px-4 text-sm font-semibold tracking-tight transition-colors disabled:cursor-not-allowed",
        className,
      )}
    >
      {loading ? <Loader2 className="size-4 animate-spin" /> : children}
    </button>
  )
}

export function GhostButton({
  className,
  loading,
  children,
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & { loading?: boolean }) {
  return (
    <button
      {...props}
      disabled={props.disabled || loading}
      className={cn(
        "border-border bg-card text-foreground hover:bg-muted/60 inline-flex h-11 items-center justify-center gap-2 rounded-xl border px-4 text-sm font-semibold tracking-tight transition-colors disabled:cursor-not-allowed disabled:opacity-50",
        className,
      )}
    >
      {loading ? <Loader2 className="size-4 animate-spin" /> : children}
    </button>
  )
}

export function DestructiveButton({
  className,
  loading,
  children,
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & { loading?: boolean }) {
  return (
    <button
      {...props}
      disabled={props.disabled || loading}
      className={cn(
        "bg-rose-500/10 text-rose-600 hover:bg-rose-500/20 dark:text-rose-300 disabled:bg-muted disabled:text-muted-foreground inline-flex h-11 items-center justify-center gap-2 rounded-xl border border-rose-500/30 px-4 text-sm font-semibold tracking-tight transition-colors disabled:cursor-not-allowed",
        className,
      )}
    >
      {loading ? <Loader2 className="size-4 animate-spin" /> : children}
    </button>
  )
}

export function StatTile({
  icon: Icon,
  label,
  value,
  meta,
  loading,
}: {
  icon: ComponentType<SVGProps<SVGSVGElement>>
  label: string
  value: ReactNode
  meta?: ReactNode
  loading?: boolean
}) {
  return (
    <div className="border-border/60 bg-background relative rounded-xl border p-3.5">
      <div className="flex items-center gap-1.5">
        <Icon className="text-muted-foreground size-3.5" strokeWidth={2} />
        <p className="text-muted-foreground text-[10px] font-semibold tracking-[0.16em] uppercase">
          {label}
        </p>
      </div>
      <p className="text-foreground mt-1.5 text-xl font-semibold tabular-nums leading-tight">
        {loading ? <Loader2 className="size-4 animate-spin" /> : value}
      </p>
      {meta !== undefined && meta !== null && (
        <p className="text-muted-foreground mt-0.5 text-[11px] tabular-nums">
          {meta}
        </p>
      )}
    </div>
  )
}

export function MutationFeedback({
  state,
}: {
  state: { status: "idle" | "pending" | "success" | "error"; error?: unknown }
}) {
  const { t } = useLocale()
  if (state.status === "success") {
    return (
      <p className="mt-2 text-xs text-emerald-600 dark:text-emerald-400">
        {t.admin.common.saved}
      </p>
    )
  }
  if (state.status === "error") {
    const message =
      state.error instanceof Error ? state.error.message : t.admin.common.error
    return (
      <p className="mt-2 text-xs text-rose-500 dark:text-rose-400">{message}</p>
    )
  }
  return null
}
