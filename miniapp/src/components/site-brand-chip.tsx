import { ArrowLeft, Shield } from "lucide-react"
import { AnimatePresence, motion } from "motion/react"

import { useBackButton } from "@/lib/back-button"
import { useLocale } from "@/lib/i18n"
import { cn } from "@/lib/utils"

const SMOOTH = {
  type: "spring",
  stiffness: 360,
  damping: 32,
  mass: 0.9,
} as const

export function SiteBrandChip({ className }: { className?: string }) {
  const { back } = useBackButton()
  const { t } = useLocale()
  const isBack = !!back

  return (
    <div
      className={cn(
        "pointer-events-none fixed inset-x-0 top-0 z-50 flex justify-start px-3 pt-[calc(var(--tg-safe-area-inset-top,env(safe-area-inset-top))+0.75rem)] sm:px-4 sm:pt-[calc(var(--tg-safe-area-inset-top,env(safe-area-inset-top))+1rem)]",
        className,
      )}
    >
      <motion.button
        type="button"
        onClick={isBack ? back : undefined}
        disabled={!isBack}
        layout
        transition={SMOOTH}
        className={cn(
          "pointer-events-auto flex h-11 items-center gap-2 rounded-full border bg-white/10 p-1 pr-3 backdrop-blur-md sm:h-12 sm:gap-2.5 sm:pr-3.5 dark:bg-black/10",
          isBack && "hover:bg-white/20 dark:hover:bg-black/20 transition-colors",
          !isBack && "cursor-default",
        )}
      >
        <div className="bg-foreground text-background relative flex aspect-square h-full shrink-0 items-center justify-center overflow-hidden rounded-full">
          <AnimatePresence initial={false} mode="wait">
            {isBack ? (
              <motion.span
                key="back"
                initial={{ opacity: 0, x: -6, rotate: -90 }}
                animate={{ opacity: 1, x: 0, rotate: 0 }}
                exit={{ opacity: 0, x: 6, rotate: 90 }}
                transition={{ duration: 0.18, ease: "easeOut" }}
                className="inline-flex"
              >
                <ArrowLeft className="size-4 sm:size-[18px]" strokeWidth={2.4} />
              </motion.span>
            ) : (
              <motion.span
                key="brand"
                initial={{ opacity: 0, x: 6, rotate: 90 }}
                animate={{ opacity: 1, x: 0, rotate: 0 }}
                exit={{ opacity: 0, x: -6, rotate: -90 }}
                transition={{ duration: 0.18, ease: "easeOut" }}
                className="inline-flex"
              >
                <Shield className="size-4 sm:size-[18px]" strokeWidth={2.4} />
              </motion.span>
            )}
          </AnimatePresence>
        </div>
        <AnimatePresence initial={false} mode="wait">
          <motion.span
            key={isBack ? "back-label" : "brand-label"}
            initial={{ opacity: 0, x: -4 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: 4 }}
            transition={{ duration: 0.18, ease: "easeOut" }}
            className="text-foreground text-sm font-semibold tracking-tight whitespace-nowrap"
          >
            {isBack ? t.common.back : "ZYVPN"}
          </motion.span>
        </AnimatePresence>
      </motion.button>
    </div>
  )
}
