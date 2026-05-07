import { Loader2, RotateCcw, Wrench } from "lucide-react"

import { useLocale } from "@/lib/i18n"

export function MaintenanceScreen({
  onRetry,
  retrying,
}: {
  onRetry?: () => void
  retrying?: boolean
}) {
  const { t } = useLocale()
  return (
    <div
      className="flex flex-col items-center justify-center px-6 text-center"
      style={{ minHeight: "100dvh" }}
    >
      <div className="bg-card border-border flex size-16 items-center justify-center rounded-2xl border shadow-lg">
        <Wrench className="text-muted-foreground size-8" strokeWidth={2} />
      </div>
      <h1 className="text-foreground mt-6 text-2xl font-semibold tracking-tight sm:text-3xl">
        {t.maintenance.title}
      </h1>
      <p className="text-muted-foreground mt-3 max-w-md text-base sm:text-[17px]">
        {t.maintenance.subtitle}
      </p>
      {onRetry && (
        <button
          type="button"
          onClick={onRetry}
          disabled={retrying}
          className="bg-foreground text-background hover:bg-foreground/90 disabled:bg-muted disabled:text-muted-foreground mt-8 inline-flex h-12 items-center justify-center gap-2 rounded-full px-6 text-base font-semibold tracking-tight transition-colors disabled:cursor-not-allowed"
        >
          {retrying ? (
            <Loader2 className="size-4 animate-spin" />
          ) : (
            <RotateCcw className="size-4" strokeWidth={2.25} />
          )}
          {t.maintenance.retry}
        </button>
      )}
    </div>
  )
}

export function SplashScreen() {
  return (
    <div
      className="flex items-center justify-center"
      style={{ minHeight: "100dvh" }}
    >
      <Loader2 className="text-muted-foreground size-7 animate-spin" />
    </div>
  )
}
