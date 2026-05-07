import { useId } from "react"
import { motion } from "motion/react"
import type { LucideIcon } from "lucide-react"

import { cn } from "@/lib/utils"

const SPRING = {
  type: "spring",
  stiffness: 420,
  damping: 32,
  mass: 0.8,
} as const

type Option<T extends string> = {
  value: T
  label: string
  icon?: LucideIcon
}

export function SegmentedToggle<T extends string>({
  options,
  value,
  onChange,
  className,
}: {
  options: Option<T>[]
  value: T
  onChange: (next: T) => void
  className?: string
}) {
  const layoutId = useId()
  return (
    <div
      role="tablist"
      className={cn(
        "border-border/60 bg-muted/60 inline-flex shrink-0 items-center rounded-full border p-0.5",
        className,
      )}
    >
      {options.map((opt) => {
        const isActive = opt.value === value
        const Icon = opt.icon
        return (
          <button
            key={opt.value}
            type="button"
            role="tab"
            aria-selected={isActive}
            onClick={() => onChange(opt.value)}
            className={cn(
              "relative inline-flex items-center gap-1.5 rounded-full px-3 py-1.5 text-xs font-semibold tracking-tight transition-colors",
              isActive
                ? "text-background"
                : "text-muted-foreground hover:text-foreground",
            )}
          >
            {isActive && (
              <motion.span
                layoutId={`segment-${layoutId}`}
                transition={SPRING}
                className="bg-foreground absolute inset-0 rounded-full"
              />
            )}
            {Icon && <Icon className="relative size-3.5" strokeWidth={2.25} />}
            <span className="relative">{opt.label}</span>
          </button>
        )
      })}
    </div>
  )
}
