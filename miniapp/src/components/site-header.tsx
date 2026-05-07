import { AnimatePresence, motion } from "motion/react"

import { Dock } from "@/components/ui/dock"
import { useLocale } from "@/lib/i18n"
import { navItems, type NavKey } from "@/lib/nav"

const navLabelKeys: Record<NavKey, "main" | "region" | "subscription" | "referrals" | "settings"> = {
  Main: "main",
  Region: "region",
  Subscription: "subscription",
  Referrals: "referrals",
  Settings: "settings",
}

const SMOOTH = { type: "spring", stiffness: 360, damping: 32, mass: 0.9 } as const

export function SiteHeader({
  active,
  onChange,
}: {
  active: NavKey
  onChange: (key: NavKey) => void
}) {
  const { t } = useLocale()
  return (
    <div className="pointer-events-none fixed inset-x-0 bottom-0 z-50 px-3 pb-[calc(var(--tg-safe-area-inset-bottom,env(safe-area-inset-bottom))+0.75rem)] sm:px-4 sm:pb-[calc(var(--tg-safe-area-inset-bottom,env(safe-area-inset-bottom))+1rem)]">
      <Dock className="pointer-events-auto mt-0 h-16 w-full gap-1 p-1 sm:h-[72px] sm:gap-2 sm:p-1.5">
        <div className="flex flex-1 items-stretch">
          {navItems.map(({ icon: Icon, key }) => {
            const isActive = active === key
            const label = t.nav[navLabelKeys[key]]
            return (
              <button
                type="button"
                key={key}
                onClick={() => onChange(key)}
                aria-pressed={isActive}
                aria-label={label}
                className="relative flex flex-1 items-center justify-center self-stretch"
              >
                <motion.span
                  layout
                  transition={SMOOTH}
                  className="relative flex items-center rounded-full px-3 py-4 sm:px-4 sm:py-3.5"
                >
                  {isActive && (
                    <motion.span
                      layoutId="active-nav-pill"
                      transition={SMOOTH}
                      className="bg-muted absolute inset-0 rounded-full"
                    />
                  )}
                  <motion.span
                    layout
                    transition={SMOOTH}
                    className="relative flex shrink-0"
                  >
                    <Icon className="size-5 sm:size-[22px]" />
                  </motion.span>
                  <AnimatePresence initial={false}>
                    {isActive && (
                      <motion.span
                        key="label"
                        layout
                        initial={{ width: 0, opacity: 0, marginLeft: 0 }}
                        animate={{ width: "auto", opacity: 1, marginLeft: 8 }}
                        exit={{ width: 0, opacity: 0, marginLeft: 0 }}
                        transition={SMOOTH}
                        className="relative overflow-hidden text-sm font-medium tracking-tight whitespace-nowrap"
                      >
                        {label}
                      </motion.span>
                    )}
                  </AnimatePresence>
                </motion.span>
              </button>
            )
          })}
        </div>
      </Dock>
    </div>
  )
}
