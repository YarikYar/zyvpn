import { Check, ChevronDown, Smartphone } from "lucide-react"
import { AnimatePresence, motion } from "motion/react"

import { useLocale } from "@/lib/i18n"
import { cn } from "@/lib/utils"

const SPRING = {
  type: "spring",
  bounce: 0,
  duration: 0.45,
} as const

const platformApps: Array<{
  key: "ios" | "android" | "desktop"
  apps: string
}> = [
  { key: "ios", apps: "Streisand · V2Box · Shadowrocket" },
  { key: "android", apps: "v2rayNG · Hiddify · NekoBox" },
  { key: "desktop", apps: "Hiddify · NekoRay · v2rayN" },
]

export function ConnectInstructions({
  open,
  onToggle,
  className,
}: {
  open: boolean
  onToggle: () => void
  className?: string
}) {
  const { t } = useLocale()

  return (
    <div
      className={cn(
        "bg-card border-border overflow-hidden rounded-2xl border",
        className,
      )}
    >
      <button
        type="button"
        onClick={onToggle}
        aria-expanded={open}
        className="hover:bg-muted/40 flex w-full items-center gap-3 px-4 py-3.5 text-left transition-colors"
      >
        <span className="bg-muted text-foreground inline-flex size-9 shrink-0 items-center justify-center rounded-xl">
          <Smartphone className="size-4" strokeWidth={2.25} />
        </span>
        <div className="min-w-0 flex-1">
          <p className="text-foreground text-sm font-semibold tracking-tight">
            {t.subscription.url.openInstructions}
          </p>
          <p className="text-muted-foreground mt-0.5 truncate text-xs">
            {open
              ? t.subscription.url.hideInstructions
              : t.subscription.success.stepTitleInstall}
          </p>
        </div>
        <motion.span
          animate={{ rotate: open ? 180 : 0 }}
          transition={SPRING}
          className="text-muted-foreground shrink-0"
        >
          <ChevronDown className="size-4" strokeWidth={2} />
        </motion.span>
      </button>

      <AnimatePresence initial={false}>
        {open && (
          <motion.div
            key="content"
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={SPRING}
            className="overflow-hidden"
          >
            <div className="border-border/60 border-t px-4 pt-4 pb-5">
              <ol className="space-y-3">
                <li className="bg-background border-border/60 rounded-2xl border p-3.5">
                  <div className="flex items-start gap-3">
                    <Step n={1} />
                    <div className="min-w-0 flex-1">
                      <p className="text-foreground text-sm font-semibold tracking-tight">
                        {t.subscription.success.stepTitleInstall}
                      </p>
                      <ul className="text-muted-foreground mt-2 space-y-1 text-xs">
                        {platformApps.map((row) => {
                          const label =
                            row.key === "ios"
                              ? t.subscription.success.platformIos
                              : row.key === "android"
                                ? t.subscription.success.platformAndroid
                                : t.subscription.success.platformDesktop
                          return (
                            <li key={row.key} className="flex gap-2">
                              <span className="text-foreground/85 w-24 shrink-0 font-medium">
                                {label}
                              </span>
                              <span>{row.apps}</span>
                            </li>
                          )
                        })}
                      </ul>
                    </div>
                  </div>
                </li>

                <StepRow
                  n={2}
                  text={t.subscription.success.stepPasteUrl}
                />
                <StepRow
                  n={3}
                  text={t.subscription.success.stepPickServer}
                />
                <li className="bg-emerald-500/10 border-emerald-500/30 flex items-center gap-3 rounded-2xl border p-3.5">
                  <span className="inline-flex size-7 shrink-0 items-center justify-center rounded-lg bg-emerald-500/25 text-emerald-700 dark:text-emerald-300">
                    <Check className="size-3.5" strokeWidth={2.5} />
                  </span>
                  <p className="text-foreground/90 text-sm">
                    {t.subscription.success.stepConnect}
                  </p>
                </li>
              </ol>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}

function Step({ n }: { n: number }) {
  return (
    <span className="bg-muted text-foreground inline-flex size-7 shrink-0 items-center justify-center rounded-lg text-xs font-semibold tabular-nums">
      {n}
    </span>
  )
}

function StepRow({ n, text }: { n: number; text: string }) {
  return (
    <li className="bg-background border-border/60 flex items-center gap-3 rounded-2xl border p-3.5">
      <Step n={n} />
      <p className="text-foreground/90 text-sm">{text}</p>
    </li>
  )
}
