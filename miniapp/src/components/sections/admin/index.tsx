import { useState } from "react"
import type { LucideIcon } from "lucide-react"
import {
  BarChart3,
  ChevronRight,
  FileText,
  Package,
  ServerCog,
  Settings as SettingsIcon,
  ShieldAlert,
  Shield,
  Ticket,
  Users,
} from "lucide-react"
import { AnimatePresence, motion } from "motion/react"

import { SectionHeading } from "@/components/sections/section-heading"
import { useRegisterBack } from "@/lib/back-button"
import { useLocale } from "@/lib/i18n"
import {
  scrollMainToTop,
  useScrollResetOnChange,
} from "@/lib/use-scroll-reset"
import { cn } from "@/lib/utils"

import { AdminBansView } from "./bans"
import { AdminLogsView } from "./logs"
import { AdminPlansView } from "./plans"
import { AdminPromoView } from "./promo"
import { AdminServersView } from "./servers"
import { AdminSettingsView } from "./settings"
import { AdminStatsView } from "./stats"
import { AdminUsersView } from "./users"

type AdminViewKey =
  | "stats"
  | "users"
  | "bans"
  | "promo"
  | "plans"
  | "servers"
  | "settings"
  | "logs"

type SectionMeta = {
  key: AdminViewKey
  icon: LucideIcon
  tone: string
}

const sections: SectionMeta[] = [
  { key: "stats", icon: BarChart3, tone: "bg-sky-500/15 text-sky-600 dark:text-sky-300" },
  { key: "users", icon: Users, tone: "bg-violet-500/15 text-violet-600 dark:text-violet-300" },
  { key: "bans", icon: ShieldAlert, tone: "bg-rose-500/15 text-rose-600 dark:text-rose-300" },
  { key: "promo", icon: Ticket, tone: "bg-fuchsia-500/15 text-fuchsia-600 dark:text-fuchsia-300" },
  { key: "plans", icon: Package, tone: "bg-amber-500/15 text-amber-600 dark:text-amber-300" },
  { key: "servers", icon: ServerCog, tone: "bg-teal-500/15 text-teal-600 dark:text-teal-300" },
  { key: "settings", icon: SettingsIcon, tone: "bg-foreground/10 text-foreground" },
  { key: "logs", icon: FileText, tone: "bg-slate-500/15 text-slate-600 dark:text-slate-300" },
]

const SLIDE_TRANSITION = {
  type: "spring",
  bounce: 0,
  duration: 0.4,
} as const

const slideVariants = {
  enter: (d: number) => ({ opacity: 0, x: d * 28 }),
  center: { opacity: 1, x: 0 },
  exit: (d: number) => ({ opacity: 0, x: d * -28 }),
}

export function AdminSection({ onClose }: { onClose: () => void }) {
  const [view, setView] = useState<AdminViewKey | null>(null)
  const [direction, setDirection] = useState(0)

  useRegisterBack(
    view === null
      ? onClose
      : () => {
          scrollMainToTop()
          setDirection(-1)
          setView(null)
        },
  )

  useScrollResetOnChange(view)

  const handlePick = (key: AdminViewKey) => {
    scrollMainToTop()
    setDirection(1)
    setView(key)
  }

  const { t } = useLocale()

  return (
    <section className="px-6 pb-16">
      <AnimatePresence mode="popLayout" initial={false} custom={direction}>
        {view === null ? (
          <motion.div
            key="admin-list"
            custom={direction}
            variants={slideVariants}
            initial="enter"
            animate="center"
            exit="exit"
            transition={SLIDE_TRANSITION}
          >
            <SectionHeading icon={Shield}>{t.admin.title}</SectionHeading>
            <p className="text-muted-foreground mt-3 max-w-md text-base sm:text-[17px]">
              {t.admin.intro}
            </p>

            <ul className="mt-8 space-y-2.5">
              {sections.map((s) => {
                const meta = t.admin.sections[s.key]
                const Icon = s.icon
                return (
                  <li key={s.key}>
                    <button
                      type="button"
                      onClick={() => handlePick(s.key)}
                      className="bg-card border-border hover:bg-muted/40 flex w-full items-center gap-3.5 rounded-2xl border p-4 text-left transition-colors"
                    >
                      <span
                        className={cn(
                          "inline-flex size-10 shrink-0 items-center justify-center rounded-xl",
                          s.tone,
                        )}
                      >
                        <Icon className="size-4" strokeWidth={2.25} />
                      </span>
                      <div className="min-w-0 flex-1">
                        <p className="text-foreground text-[15px] font-semibold tracking-tight">
                          {meta.title}
                        </p>
                        <p className="text-muted-foreground mt-0.5 truncate text-sm">
                          {meta.subtitle}
                        </p>
                      </div>
                      <ChevronRight
                        className="text-muted-foreground size-4 shrink-0"
                        strokeWidth={2}
                      />
                    </button>
                  </li>
                )
              })}
            </ul>
          </motion.div>
        ) : (
          <motion.div
            key={`admin-view-${view}`}
            custom={direction}
            variants={slideVariants}
            initial="enter"
            animate="center"
            exit="exit"
            transition={SLIDE_TRANSITION}
          >
            <AdminViewHeader viewKey={view} />
            <ViewRenderer view={view} />
          </motion.div>
        )}
      </AnimatePresence>
    </section>
  )
}

function AdminViewHeader({ viewKey }: { viewKey: AdminViewKey }) {
  const { t } = useLocale()
  const meta = sections.find((s) => s.key === viewKey)!
  const Icon = meta.icon
  return (
    <>
      <SectionHeading icon={Icon}>
        {t.admin.sections[viewKey].title}
      </SectionHeading>
      <p className="text-muted-foreground mt-3 max-w-md text-base sm:text-[17px]">
        {t.admin.sections[viewKey].subtitle}
      </p>
    </>
  )
}

function ViewRenderer({ view }: { view: AdminViewKey }) {
  switch (view) {
    case "stats":
      return <AdminStatsView />
    case "users":
      return <AdminUsersView />
    case "bans":
      return <AdminBansView />
    case "promo":
      return <AdminPromoView />
    case "plans":
      return <AdminPlansView />
    case "servers":
      return <AdminServersView />
    case "settings":
      return <AdminSettingsView />
    case "logs":
      return <AdminLogsView />
  }
}
