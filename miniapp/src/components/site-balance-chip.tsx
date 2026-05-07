import { Loader2 } from "lucide-react"
import { motion } from "motion/react"

import { TonIcon } from "@/components/icons/ton"
import { formatTonBalance } from "@/lib/format"
import { useBalance } from "@/lib/queries"
import { cn } from "@/lib/utils"

const SMOOTH = {
  type: "spring",
  stiffness: 360,
  damping: 32,
  mass: 0.9,
} as const

export function SiteBalanceChip({
  className,
  onClick,
}: {
  className?: string
  onClick?: () => void
}) {
  const balanceQuery = useBalance()
  const balance = balanceQuery.data?.balance

  return (
    <div
      className={cn(
        "pointer-events-none fixed inset-x-0 top-0 z-50 flex justify-end px-3 pt-[calc(var(--tg-safe-area-inset-top,env(safe-area-inset-top))+0.75rem)] sm:px-4 sm:pt-[calc(var(--tg-safe-area-inset-top,env(safe-area-inset-top))+1rem)]",
        className,
      )}
    >
      <motion.button
        type="button"
        onClick={onClick}
        layout
        transition={SMOOTH}
        className="pointer-events-auto flex h-11 items-center gap-2 rounded-full border bg-white/10 p-1 pl-3 backdrop-blur-md transition-colors hover:bg-white/20 sm:h-12 sm:gap-2.5 sm:pl-3.5 dark:bg-black/10 dark:hover:bg-black/20"
      >
        <span className="text-foreground text-sm font-medium tabular-nums tracking-tight whitespace-nowrap">
          {balance === undefined ? (
            balanceQuery.isPending ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              "—"
            )
          ) : (
            formatTonBalance(balance)
          )}
        </span>
        <div className="relative aspect-square h-full shrink-0">
          <TonIcon className="size-full" />
        </div>
      </motion.button>
    </div>
  )
}
