import type { ComponentType, SVGProps } from "react"
import { useEffect, useMemo, useState } from "react"
import {
  ArrowRight,
  Banknote,
  CheckCircle2,
  Globe2,
  Loader2,
  Star,
  Wallet,
} from "lucide-react"
import { AnimatePresence, motion } from "motion/react"

import { TonIcon } from "@/components/icons/ton"
import { SectionHeading } from "@/components/sections/section-heading"
import { api, type Rates, type Server } from "@/lib/api"
import { useRegisterBack } from "@/lib/back-button"
import { flagFor, loadFromPercent, loadFromPing, pingTone, regionGlowFor, regionTintFor } from "@/lib/flags"
import { formatRubPurchase, formatTonBalance, formatTonPurchase } from "@/lib/format"
import { useLocale } from "@/lib/i18n"
import { scrollMainToTop, useScrollResetOnChange } from "@/lib/use-scroll-reset"
import {
  useBalance,
  usePaymentStatus,
  useRates,
  useServers,
  useSubscriptionStatus,
  useSwitchServer,
  useSwitchServerInfo,
} from "@/lib/queries"
import { cn } from "@/lib/utils"

const SELECT_SPRING = {
  type: "spring",
  stiffness: 420,
  damping: 28,
  mass: 0.8,
} as const

const HOVER_SPRING = {
  type: "spring",
  stiffness: 360,
  damping: 26,
  mass: 0.6,
} as const

const SLIDE_TRANSITION = {
  type: "spring",
  bounce: 0,
  duration: 0.45,
} as const

const slideVariants = {
  enter: (d: number) => ({ opacity: 0, x: d * 28 }),
  center: { opacity: 1, x: 0 },
  exit: (d: number) => ({ opacity: 0, x: d * -28 }),
}

function selectedShadow(code: string) {
  return `0 18px 44px -18px rgba(0,0,0,0.4), 0 0 36px ${regionGlowFor(code)}`
}

const loadDot: Record<"low" | "medium" | "high", string> = {
  low: "bg-emerald-400",
  medium: "bg-amber-300",
  high: "bg-red-300",
}

function loadOf(server: Server) {
  if (typeof server.load_percent === "number") return loadFromPercent(server.load_percent)
  if (typeof server.ping_ms === "number") return loadFromPing(server.ping_ms)
  return "low" as const
}

function codeOf(server: Server): string {
  return (server.country ?? "").toUpperCase()
}

function locationOf(server: Server): string {
  return server.city?.trim() || server.name?.trim() || ""
}

function isServerOffline(server: Server): boolean {
  return server.status === "offline"
}

type RegionView = "list" | "checkout"

export function RegionSection() {
  const [view, setView] = useState<RegionView>("list")
  const [candidate, setCandidate] = useState<Server | null>(null)
  const [direction, setDirection] = useState(0)

  useScrollResetOnChange(view)

  const handlePick = (server: Server) => {
    scrollMainToTop()
    setCandidate(server)
    setDirection(1)
    setView("checkout")
  }

  const handleBack = () => {
    scrollMainToTop()
    setDirection(-1)
    setView("list")
  }

  const handlePay = () => {
    scrollMainToTop()
    setDirection(-1)
    setView("list")
  }

  return (
    <section className="px-6 pb-16">
      <AnimatePresence mode="popLayout" initial={false} custom={direction}>
        {view === "list" && (
          <motion.div
            key="list"
            custom={direction}
            variants={slideVariants}
            initial="enter"
            animate="center"
            exit="exit"
            transition={SLIDE_TRANSITION}
          >
            <RegionListView onPick={handlePick} />
          </motion.div>
        )}
        {view === "checkout" && candidate && (
          <motion.div
            key="checkout"
            custom={direction}
            variants={slideVariants}
            initial="enter"
            animate="center"
            exit="exit"
            transition={SLIDE_TRANSITION}
          >
            <RegionCheckoutView
              server={candidate}
              onPay={handlePay}
              onBack={handleBack}
            />
          </motion.div>
        )}
      </AnimatePresence>
    </section>
  )
}

function RegionListView({ onPick }: { onPick: (server: Server) => void }) {
  const { t } = useLocale()
  const serversQuery = useServers()
  const statusQuery = useSubscriptionStatus()
  const myServerId = statusQuery.data?.subscription?.server_id

  const sortedServers = useMemo(() => {
    const items = serversQuery.data ?? []
    const mine = items.filter((s) => myServerId && s.id === myServerId)
    const rest = items.filter((s) => !myServerId || s.id !== myServerId)
    const onlineRest = rest.filter((s) => !isServerOffline(s))
    const offlineRest = rest.filter((s) => isServerOffline(s))
    return [...mine, ...onlineRest, ...offlineRest]
  }, [serversQuery.data, myServerId])

  return (
    <>
      <SectionHeading icon={Globe2}>{t.region.title}</SectionHeading>

      {serversQuery.isPending && (
        <div className="mt-8 flex items-center justify-center py-12">
          <Loader2 className="text-muted-foreground size-6 animate-spin" />
        </div>
      )}
      {serversQuery.error && (
        <p className="mt-8 text-sm text-rose-500">
          {serversQuery.error instanceof Error
            ? serversQuery.error.message
            : "Failed to load servers"}
        </p>
      )}

      {sortedServers.length > 0 && (
        <ul className="mt-8 space-y-3">
          {sortedServers.map((s) => {
            const isActive = !!myServerId && s.id === myServerId
            const offline = isServerOffline(s)
            const interactive = !isActive && !offline
            const ping = s.ping_ms ?? 0
            const load = loadOf(s)
            const code = codeOf(s)
            const Flag = flagFor(code)
            const meta = t.region.countries[code as keyof typeof t.region.countries]
            const country = meta?.country ?? code
            const city = meta?.city ?? locationOf(s)
            return (
              <li key={s.id}>
                <motion.button
                  type="button"
                  onClick={() => interactive && onPick(s)}
                  aria-pressed={isActive}
                  aria-disabled={offline}
                  animate={{ scale: isActive ? 1.02 : 1 }}
                  whileHover={interactive ? { scale: 1.015, y: -2 } : undefined}
                  whileTap={interactive ? { scale: 0.985 } : undefined}
                  transition={HOVER_SPRING}
                  style={isActive ? { boxShadow: selectedShadow(code) } : undefined}
                  className={cn(
                    "group bg-card border-border relative w-full overflow-hidden rounded-2xl border bg-gradient-to-br to-transparent p-5 text-left shadow-lg select-none",
                    regionTintFor(code),
                    offline && "cursor-not-allowed opacity-60 saturate-50",
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
                      className="text-muted-foreground/20 pointer-events-none absolute -right-3 -bottom-3 size-20 sm:-right-4 sm:-bottom-4 sm:size-24"
                      strokeWidth={1.2}
                    />
                  )}

                  <div className="relative flex items-start justify-between gap-4">
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center">
                        <AnimatePresence initial={false}>
                          {isActive && (
                            <motion.span
                              key="active-chip"
                              initial={{ opacity: 0, scale: 0.6, width: 0, marginRight: 0 }}
                              animate={{ opacity: 1, scale: 1, width: "auto", marginRight: 8 }}
                              exit={{ opacity: 0, scale: 0.6, width: 0, marginRight: 0 }}
                              transition={SELECT_SPRING}
                              className="inline-flex shrink-0 items-center gap-1.5 overflow-hidden rounded-full border border-emerald-500/40 bg-emerald-500/15 px-2 py-0.5 text-[10px] font-semibold tracking-[0.12em] text-emerald-700 uppercase dark:text-emerald-300"
                            >
                              <span aria-hidden className="size-1.5 rounded-full bg-emerald-500" />
                              {t.common.active}
                            </motion.span>
                          )}
                        </AnimatePresence>
                        <p className="text-foreground text-xl font-semibold tracking-tight sm:text-2xl">
                          {country}
                        </p>
                      </div>
                      {city && (
                        <p className="text-muted-foreground mt-0.5 text-sm sm:text-[15px]">
                          {city}
                        </p>
                      )}
                    </div>
                    <div className="shrink-0 text-right">
                      {offline ? (
                        <p className="inline-flex items-center gap-1.5 text-sm font-semibold tracking-tight text-rose-500 dark:text-rose-400">
                          <span
                            aria-hidden
                            className="size-1.5 rounded-full bg-rose-500"
                          />
                          {t.region.offline}
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
                            {t.region.load[load]}
                          </p>
                        </>
                      )}
                    </div>
                  </div>
                </motion.button>
              </li>
            )
          })}
        </ul>
      )}
    </>
  )
}

type SwitchMethodId = "balance" | "ton" | "stars" | "cash"

type SwitchMethod = {
  id: SwitchMethodId
  name: string
  subtitle: string
  icon: ComponentType<SVGProps<SVGSVGElement>>
  iconBg: string
  iconColor: string
  available: boolean
  ctaLabel: string
}

function TonInline(props: SVGProps<SVGSVGElement>) {
  return <TonIcon {...props} />
}

function buildSwitchMethods(
  priceTon: number,
  balance: number,
  rates: Rates | undefined,
  t: ReturnType<typeof useLocale>["t"],
): SwitchMethod[] {
  const balanceEnough = balance >= priceTon
  const starsCount = Math.max(1, Math.round(priceTon * 60))
  const rubApprox = formatRubPurchase(priceTon, rates)
  const rubSuffix = rubApprox ? ` ≈ ${rubApprox} ₽` : ""
  const m = t.subscription.checkout.methods
  return [
    {
      id: "balance",
      name: m.balance.name,
      subtitle: `${formatTonBalance(balance)} TON ${
        balanceEnough ? m.balance.enough : m.balance.insufficient
      }`,
      icon: Wallet,
      iconBg: "bg-amber-500/15",
      iconColor: "text-amber-500",
      available: balanceEnough,
      ctaLabel: t.subscription.checkout.payWithBalance(priceTon),
    },
    {
      id: "ton",
      name: m.ton.name,
      subtitle: `${formatTonPurchase(priceTon)} TON${rubSuffix}`,
      icon: TonInline,
      iconBg: "bg-sky-500/15",
      iconColor: "text-sky-500",
      available: true,
      ctaLabel: t.region.checkout.payCta(priceTon),
    },
    {
      id: "stars",
      name: m.stars.name,
      subtitle: `${starsCount} Stars${rubSuffix}`,
      icon: Star,
      iconBg: "bg-yellow-400/15",
      iconColor: "text-yellow-500 dark:text-yellow-300",
      available: true,
      ctaLabel: t.subscription.checkout.payStars(starsCount),
    },
    {
      id: "cash",
      name: m.cash.name,
      subtitle: rubApprox ? m.cash.subtitle(Number(rubApprox)) : m.cash.name,
      icon: Banknote,
      iconBg: "bg-emerald-500/15",
      iconColor: "text-emerald-500",
      available: !!rubApprox,
      ctaLabel: t.subscription.checkout.submitCash,
    },
  ]
}

function RegionCheckoutView({
  server,
  onPay,
  onBack,
}: {
  server: Server
  onPay: () => void
  onBack: () => void
}) {
  useRegisterBack(onBack)
  const { t } = useLocale()
  const code = codeOf(server)
  const meta = t.region.countries[code as keyof typeof t.region.countries]
  const country = meta?.country ?? code
  const city = meta?.city ?? locationOf(server)
  const Flag = flagFor(code)
  const ping = server.ping_ms ?? 0
  const load = loadOf(server)

  const switchInfo = useSwitchServerInfo()
  const balanceQuery = useBalance()
  const ratesQuery = useRates()
  const balance = balanceQuery.data?.balance ?? 0
  const priceTon = switchInfo.data?.price_ton ?? 0.1

  const switchMutation = useSwitchServer()
  const [paymentId, setPaymentId] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const statusQuery = usePaymentStatus(paymentId)

  useEffect(() => {
    if (statusQuery.data?.status === "succeeded") {
      setPaymentId(null)
      onPay()
    }
    if (statusQuery.data?.status === "failed" || statusQuery.data?.status === "expired") {
      setPaymentId(null)
      setError("Payment did not succeed")
    }
  }, [statusQuery.data, onPay])

  const methods = buildSwitchMethods(priceTon, balance, ratesQuery.data, t)
  const firstAvailable = methods.find((m) => m.available)?.id ?? methods[0].id
  const [methodId, setMethodId] = useState<SwitchMethodId>(firstAvailable)
  const method = methods.find((m) => m.id === methodId) ?? methods[0]

  const isPending = switchMutation.isPending || !!paymentId

  const handlePay = async () => {
    setError(null)
    try {
      const result = await switchMutation.mutateAsync({
        server_id: server.id,
        provider: method.id === "cash" ? undefined : (method.id as "balance" | "ton" | "stars"),
      })
      if (method.id === "balance" || !result.payment_id) {
        onPay()
        return
      }
      const id = result.payment_id
      if (method.id === "ton") {
        const init = await api.paymentTonInit(id)
        if (init.deep_link) openExternalLink(init.deep_link)
        setPaymentId(id)
        return
      }
      if (method.id === "stars") {
        const init = await api.paymentStarsInit(id)
        if (init.invoice_url) {
          openInvoice(init.invoice_url, (status) => {
            if (status === "paid") onPay()
            else if (status === "failed" || status === "cancelled") {
              setError("Payment cancelled")
            }
          })
        }
        return
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Switch failed")
    }
  }

  return (
    <>
      <SectionHeading icon={Globe2}>{t.region.checkout.title}</SectionHeading>
      <p className="text-muted-foreground mt-3 max-w-md text-base sm:text-[17px]">
        {t.region.checkout.intro}
      </p>

      <div
        className={cn(
          "bg-card border-border relative mt-6 overflow-hidden rounded-2xl border bg-gradient-to-br to-transparent p-5 shadow-lg",
          regionTintFor(code),
        )}
      >
        {Flag && (
          <Flag
            aria-hidden
            className="pointer-events-none absolute -right-3 -bottom-3 h-20 aspect-[3/2] rounded-xl opacity-25 sm:-right-4 sm:-bottom-4 sm:h-24"
          />
        )}
        <p className="text-muted-foreground relative text-xs font-semibold tracking-[0.18em] uppercase">
          {t.region.checkout.newRegion}
        </p>
        <div className="relative mt-2 flex items-start justify-between gap-3">
          <div className="min-w-0 flex-1">
            <h3 className="text-foreground text-xl font-bold tracking-tight sm:text-2xl">
              {country}
            </h3>
            {city && (
              <p className="text-muted-foreground mt-1 text-sm">
                {city}
              </p>
            )}
            <div className="mt-3 inline-flex items-center gap-1.5 text-xs">
              <span className={cn("font-semibold tabular-nums", pingTone(ping))}>
                {ping} {t.common.msUnit}
              </span>
              <span className="text-muted-foreground">·</span>
              <span className="text-muted-foreground">{t.region.load[load]}</span>
            </div>
          </div>
          <div className="shrink-0 text-right leading-none">
            <p className="text-foreground text-xl font-bold tabular-nums sm:text-2xl">
              {formatTonPurchase(priceTon)} TON
            </p>
            {switchInfo.data && switchInfo.data.free_switches_remaining > 0 && (
              <p className="text-emerald-600 mt-1.5 text-xs font-medium dark:text-emerald-400">
                {switchInfo.data.free_switches_remaining} free
              </p>
            )}
          </div>
        </div>
      </div>

      <h3 className="text-foreground mt-8 text-lg font-semibold tracking-tight">
        {t.subscription.checkout.paymentMethod}
      </h3>
      <ul className="mt-4 space-y-2.5">
        {methods.map((m) => {
          const Icon = m.icon
          const isPicked = methodId === m.id
          return (
            <li key={m.id}>
              <motion.button
                type="button"
                onClick={() => m.available && setMethodId(m.id)}
                aria-pressed={isPicked}
                disabled={!m.available || isPending}
                animate={{ scale: isPicked ? 1.02 : 1 }}
                whileHover={isPicked || !m.available ? undefined : { scale: 1.01 }}
                whileTap={!m.available ? undefined : { scale: 0.985 }}
                transition={HOVER_SPRING}
                className={cn(
                  "border-border bg-card hover:bg-muted/40 relative flex w-full items-center gap-3 rounded-2xl border p-4 text-left transition-colors select-none disabled:cursor-not-allowed disabled:opacity-50",
                  isPicked &&
                    "border-foreground/80 hover:bg-card shadow-foreground/10 shadow-xl",
                )}
              >
                <span
                  className={cn(
                    "flex size-10 shrink-0 items-center justify-center rounded-xl",
                    m.iconBg,
                  )}
                >
                  <Icon className={cn("size-5", m.iconColor)} strokeWidth={2} />
                </span>
                <div className="min-w-0 flex-1">
                  <p className="text-foreground font-semibold tracking-tight">
                    {m.name}
                  </p>
                  <p className="text-muted-foreground mt-0.5 text-sm">
                    {m.subtitle}
                  </p>
                </div>
                <AnimatePresence initial={false}>
                  {isPicked && (
                    <motion.span
                      key="picked"
                      initial={{ opacity: 0, scale: 0.6 }}
                      animate={{ opacity: 1, scale: 1 }}
                      exit={{ opacity: 0, scale: 0.6 }}
                      transition={SELECT_SPRING}
                      className="text-foreground inline-flex shrink-0 items-center justify-center"
                    >
                      <CheckCircle2 className="size-5" strokeWidth={2.5} />
                    </motion.span>
                  )}
                </AnimatePresence>
              </motion.button>
            </li>
          )
        })}
      </ul>

      {error && <p className="mt-4 text-sm text-rose-500">{error}</p>}
      {paymentId && (
        <p className="text-muted-foreground mt-4 inline-flex items-center gap-2 text-sm">
          <Loader2 className="size-4 animate-spin" />
          Waiting for payment confirmation…
        </p>
      )}

      <button
        type="button"
        onClick={handlePay}
        disabled={!method.available || isPending}
        className="bg-foreground text-background hover:bg-foreground/90 disabled:bg-muted disabled:text-muted-foreground mt-8 inline-flex h-14 w-full items-center justify-center gap-2 rounded-2xl text-base font-semibold tracking-tight transition-colors disabled:cursor-not-allowed"
      >
        {isPending ? (
          <Loader2 className="size-5 animate-spin" />
        ) : (
          <>
            {method.ctaLabel}
            <ArrowRight className="size-4" strokeWidth={2.5} />
          </>
        )}
      </button>
    </>
  )
}

function openExternalLink(url: string) {
  const tg = (window as { Telegram?: { WebApp?: { openLink?: (url: string) => void } } }).Telegram?.WebApp
  if (tg?.openLink) {
    tg.openLink(url)
    return
  }
  window.open(url, "_blank", "noopener,noreferrer")
}

type InvoiceCallback = (status: "paid" | "failed" | "cancelled" | "pending") => void

function openInvoice(url: string, callback: InvoiceCallback) {
  const tg = (window as {
    Telegram?: {
      WebApp?: {
        openInvoice?: (url: string, callback: InvoiceCallback) => void
      }
    }
  }).Telegram?.WebApp
  if (tg?.openInvoice) {
    tg.openInvoice(url, callback)
    return
  }
  openExternalLink(url)
}
