import type { ComponentType, ReactNode, SVGProps } from "react"
import { useEffect, useMemo, useState } from "react"
import {
  ArrowRight,
  BarChart3,
  Banknote,
  Calendar,
  Check,
  CheckCircle2,
  ChevronRight,
  Copy,
  CreditCard,
  KeyRound,
  Loader2,
  Smartphone,
  Sparkles,
  Star,
  Wallet,
} from "lucide-react"
import { AnimatePresence, motion } from "motion/react"
import { QRCodeSVG } from "qrcode.react"

import { TonIcon } from "@/components/icons/ton"
import { SectionHeading } from "@/components/sections/section-heading"
import { api, type Plan as ApiPlan } from "@/lib/api"
import { useRegisterBack } from "@/lib/back-button"
import { flagFor, regionTintFor } from "@/lib/flags"
import { formatRubPurchase, formatTonBalance, formatTonPurchase } from "@/lib/format"
import { useLocale } from "@/lib/i18n"
import type { Rates } from "@/lib/api"
import {
  planActiveShadow,
  planMetaFor,
  type PlanMeta,
} from "@/lib/plan-meta"
import {
  useBalance,
  useBuySubscription,
  usePayWithBalance,
  usePaymentStatus,
  usePlans,
  useRates,
  useServers,
  useSubscriptionKey,
  useSubscriptionStatus,
} from "@/lib/queries"
import { scrollMainToTop, useScrollResetOnChange } from "@/lib/use-scroll-reset"
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

function trafficLabel(plan: ApiPlan, gb: string) {
  return plan.traffic_gb === null ? "∞" : `${plan.traffic_gb} ${gb}`
}

export type SubscriptionView = "list" | "checkout" | "success"

type SuccessEntry = "payment" | "external"

export function SubscriptionSection({
  requestView,
  onConsumeRequest,
  onExternalBack,
}: {
  requestView?: SubscriptionView | null
  onConsumeRequest?: () => void
  onExternalBack?: () => void
} = {}) {
  const [view, setView] = useState<SubscriptionView>(() => requestView ?? "list")
  const [selectedPlanId, setSelectedPlanId] = useState<string | null>(null)
  const [direction, setDirection] = useState(0)
  const [successEntry, setSuccessEntry] = useState<SuccessEntry>(() =>
    requestView ? "external" : "payment",
  )

  useScrollResetOnChange(view)

  const plansQuery = usePlans()

  useEffect(() => {
    if (!requestView) return
    scrollMainToTop()
    setDirection(1)
    setSuccessEntry("external")
    setView(requestView)
    onConsumeRequest?.()
  }, [requestView, onConsumeRequest])

  const selectedPlan = plansQuery.data?.find((p) => p.id === selectedPlanId) ?? null

  const goCheckout = (id: string) => {
    scrollMainToTop()
    setSelectedPlanId(id)
    setDirection(1)
    setView("checkout")
  }

  const goSuccess = () => {
    scrollMainToTop()
    setDirection(1)
    setSuccessEntry("payment")
    setView("success")
  }

  const goBackToList = () => {
    scrollMainToTop()
    setDirection(-1)
    setView("list")
  }

  const handleSuccessBack = () => {
    if (successEntry === "external" && onExternalBack) {
      onExternalBack()
      return
    }
    goBackToList()
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
            <PlansListView onPick={goCheckout} />
          </motion.div>
        )}
        {view === "checkout" && selectedPlan && (
          <motion.div
            key="checkout"
            custom={direction}
            variants={slideVariants}
            initial="enter"
            animate="center"
            exit="exit"
            transition={SLIDE_TRANSITION}
          >
            <CheckoutView
              plan={selectedPlan}
              onPay={goSuccess}
              onBack={goBackToList}
            />
          </motion.div>
        )}
        {view === "success" && (
          <motion.div
            key="success"
            custom={direction}
            variants={slideVariants}
            initial="enter"
            animate="center"
            exit="exit"
            transition={SLIDE_TRANSITION}
          >
            <SuccessView
              onBack={handleSuccessBack}
              showPaymentBanner={successEntry === "payment"}
            />
          </motion.div>
        )}
      </AnimatePresence>
    </section>
  )
}

function PlansListView({ onPick }: { onPick: (id: string) => void }) {
  const { t } = useLocale()
  const plansQuery = usePlans()
  const statusQuery = useSubscriptionStatus()
  const ratesQuery = useRates()

  const activeId = statusQuery.data?.active
    ? statusQuery.data.subscription?.plan_id ?? null
    : null

  const sortedPlans = useMemo(() => {
    const items = plansQuery.data ?? []
    if (!activeId) return items
    return [
      ...items.filter((p) => p.id === activeId),
      ...items.filter((p) => p.id !== activeId),
    ]
  }, [plansQuery.data, activeId])

  return (
    <>
      <SectionHeading icon={CreditCard}>{t.subscription.title}</SectionHeading>
      <p className="text-muted-foreground mt-3 max-w-md text-base sm:text-[17px]">
        {t.subscription.intro}
      </p>

      {plansQuery.isPending && (
        <div className="mt-8 flex items-center justify-center py-12">
          <Loader2 className="text-muted-foreground size-6 animate-spin" />
        </div>
      )}
      {plansQuery.error && (
        <p className="mt-8 text-sm text-rose-500">
          {plansQuery.error instanceof Error
            ? plansQuery.error.message
            : "Failed to load plans"}
        </p>
      )}

      {sortedPlans.length > 0 && (
        <ul className="mt-8 space-y-3">
          {sortedPlans.map((p) => {
            const meta = planMetaFor(p)
            const Icon = meta.Icon
            const isActive = activeId === p.id
            const tierLabel = t.subscription.plans[meta.variant].tier
            return (
              <li key={p.id}>
                <motion.button
                  type="button"
                  onClick={() => onPick(p.id)}
                  aria-pressed={isActive}
                  animate={{ scale: isActive ? 1.02 : 1 }}
                  whileHover={isActive ? undefined : { scale: 1.015, y: -2 }}
                  whileTap={isActive ? undefined : { scale: 0.985 }}
                  transition={HOVER_SPRING}
                  style={isActive ? { boxShadow: planActiveShadow(meta.variant) } : undefined}
                  className={cn(
                    "group bg-card border-border relative w-full overflow-hidden rounded-2xl border bg-gradient-to-br to-transparent p-5 text-left shadow-lg select-none",
                    meta.tint,
                  )}
                >
                  <Icon
                    aria-hidden
                    strokeWidth={1.2}
                    className={cn(
                      "pointer-events-none absolute -right-3 -bottom-8 size-36 transition-transform duration-500 ease-out group-hover:scale-105 group-hover:rotate-3 sm:-right-4 sm:-bottom-10 sm:size-44",
                      meta.iconColor,
                    )}
                  />

                  <div className="relative flex items-center justify-between gap-3">
                    <p
                      className={cn(
                        "inline-flex items-center gap-2 text-xs font-semibold tracking-[0.18em] uppercase",
                        meta.kicker,
                      )}
                    >
                      <AnimatePresence initial={false}>
                        {isActive && (
                          <motion.span
                            key="active-chip"
                            aria-hidden
                            initial={{ opacity: 0, scale: 0.6, width: 0, marginRight: 0 }}
                            animate={{ opacity: 1, scale: 1, width: "auto", marginRight: 6 }}
                            exit={{ opacity: 0, scale: 0.6, width: 0, marginRight: 0 }}
                            transition={SELECT_SPRING}
                            className="inline-flex shrink-0 items-center gap-1.5 overflow-hidden"
                          >
                            <span aria-hidden className="relative inline-flex size-2">
                              <span className="absolute inset-0 rounded-full bg-emerald-500" />
                              <span className="absolute inset-0 animate-ping rounded-full bg-emerald-500/60" />
                            </span>
                          </motion.span>
                        )}
                      </AnimatePresence>
                      {isActive ? `${t.common.active} · ` : ""}
                      {tierLabel}
                    </p>
                    {p.popular && (
                      <span className="border-foreground/15 bg-foreground/8 text-foreground inline-flex shrink-0 items-center gap-1 rounded-full border px-2 py-0.5 text-[10px] font-semibold tracking-[0.12em] uppercase">
                        <Sparkles className="size-2.5" strokeWidth={2.5} />
                        {t.common.popular}
                      </span>
                    )}
                  </div>

                  <h3 className="text-foreground relative mt-2 text-2xl font-bold tracking-tight sm:text-3xl">
                    {p.name}
                  </h3>
                  <p className="text-muted-foreground relative mt-1 max-w-[80%] text-sm leading-snug sm:text-[15px]">
                    {t.subscription.composeDescription(p.duration_days, p.traffic_gb)}
                  </p>

                  <div className="relative mt-4 grid grid-cols-3 gap-2">
                    <PlanStat
                      icon={Calendar}
                      label={t.subscription.fields.term}
                      value={t.subscription.daysLabel(p.duration_days)}
                    />
                    <PlanStat
                      icon={BarChart3}
                      label={t.subscription.fields.traffic}
                      value={trafficLabel(p, t.common.gb)}
                    />
                    <PlanStat
                      icon={Smartphone}
                      label={t.subscription.fields.devices}
                      value="3"
                    />
                  </div>

                  <div className="relative mt-4 flex items-baseline justify-between gap-3">
                    <p className="text-muted-foreground text-xs font-semibold tracking-[0.16em] uppercase">
                      {t.subscription.fields.price}
                    </p>
                    <div className="flex items-baseline gap-2">
                      <span className="text-muted-foreground text-sm font-medium tabular-nums">
                        {formatTonPurchase(p.price_ton)} TON
                      </span>
                      {(() => {
                        const rub = formatRubPurchase(p.price_ton, ratesQuery.data)
                        if (!rub) return null
                        return (
                          <span className="text-foreground text-2xl font-bold tabular-nums sm:text-[26px]">
                            ≈{rub}
                            <span className="text-muted-foreground ml-0.5 text-base font-medium">
                              ₽
                            </span>
                          </span>
                        )
                      })()}
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

function PlanStat({
  icon: Icon,
  label,
  value,
}: {
  icon: ComponentType<SVGProps<SVGSVGElement>>
  label: string
  value: string
}) {
  return (
    <div className="border-border/60 bg-background relative rounded-xl border p-3">
      <div className="flex items-center gap-1.5">
        <Icon className="text-muted-foreground size-3.5" strokeWidth={2} />
        <p className="text-muted-foreground text-[10px] font-semibold tracking-[0.16em] uppercase">
          {label}
        </p>
      </div>
      <p className="text-foreground mt-1 text-base font-semibold tabular-nums">
        {value}
      </p>
    </div>
  )
}

type PaymentMethodId = "balance" | "ton" | "stars" | "cash"

type PaymentMethod = {
  id: PaymentMethodId
  name: string
  subtitle: ReactNode
  icon: ComponentType<SVGProps<SVGSVGElement>>
  iconBg: string
  iconColor: string
  available: boolean
  ctaLabel: ReactNode
}

function TonInline(props: SVGProps<SVGSVGElement>) {
  return <TonIcon {...props} />
}

function buildMethods(
  plan: ApiPlan,
  balance: number,
  rates: Rates | undefined,
  t: ReturnType<typeof useLocale>["t"],
): PaymentMethod[] {
  const balanceEnough = balance >= plan.price_ton
  const m = t.subscription.checkout.methods
  const rubApprox = formatRubPurchase(plan.price_ton, rates)
  const rubSuffix = rubApprox ? ` ≈ ${rubApprox} ₽` : ""
  const rubForCash = rubApprox ? Number(rubApprox) : 0
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
      ctaLabel: t.subscription.checkout.payWithBalance(plan.price_ton),
    },
    {
      id: "ton",
      name: m.ton.name,
      subtitle: `${formatTonPurchase(plan.price_ton)} TON${rubSuffix}`,
      icon: TonInline,
      iconBg: "bg-sky-500/15",
      iconColor: "text-sky-500",
      available: true,
      ctaLabel: t.subscription.checkout.payTon(plan.price_ton),
    },
    {
      id: "stars",
      name: m.stars.name,
      subtitle: `${plan.price_stars} Stars${rubSuffix}`,
      icon: Star,
      iconBg: "bg-yellow-400/15",
      iconColor: "text-yellow-500 dark:text-yellow-300",
      available: true,
      ctaLabel: t.subscription.checkout.payStars(plan.price_stars),
    },
    {
      id: "cash",
      name: m.cash.name,
      subtitle: rubApprox ? m.cash.subtitle(rubForCash) : m.cash.name,
      icon: Banknote,
      iconBg: "bg-emerald-500/15",
      iconColor: "text-emerald-500",
      available: !!rubApprox,
      ctaLabel: t.subscription.checkout.submitCash,
    },
  ]
}

function CheckoutView({
  plan,
  onPay,
  onBack,
}: {
  plan: ApiPlan
  onPay: () => void
  onBack: () => void
}) {
  useRegisterBack(onBack)
  const { t } = useLocale()
  const meta = planMetaFor(plan)
  const tierLabel = t.subscription.plans[meta.variant].tier
  const balanceQuery = useBalance()
  const balance = balanceQuery.data?.balance ?? 0
  const ratesQuery = useRates()

  const buyMutation = useBuySubscription()
  const balancePay = usePayWithBalance()

  const methods = buildMethods(plan, balance, ratesQuery.data, t)
  const firstAvailable =
    methods.find((m) => m.available)?.id ?? methods[0].id
  const [methodId, setMethodId] = useState<PaymentMethodId>(firstAvailable)
  const method = methods.find((m) => m.id === methodId) ?? methods[0]

  const [paymentId, setPaymentId] = useState<string | null>(null)
  const [payError, setPayError] = useState<string | null>(null)
  const statusQuery = usePaymentStatus(paymentId)

  useEffect(() => {
    if (statusQuery.data?.status === "succeeded") {
      setPaymentId(null)
      onPay()
    }
    if (statusQuery.data?.status === "failed" || statusQuery.data?.status === "expired") {
      setPaymentId(null)
      setPayError("Payment did not succeed")
    }
  }, [statusQuery.data, onPay])

  const isPending =
    buyMutation.isPending || balancePay.isPending || !!paymentId

  const handlePay = async () => {
    setPayError(null)
    try {
      if (method.id === "balance") {
        await balancePay.mutateAsync({ plan_id: plan.id })
        onPay()
        return
      }
      const result = await buyMutation.mutateAsync({
        plan_id: plan.id,
        provider: method.id,
      })
      const paymentIdReturned = result.payment_id
      if (method.id === "cash") {
        onPay()
        return
      }
      if (!paymentIdReturned) {
        setPayError("Backend did not return payment_id")
        return
      }
      if (method.id === "ton") {
        const init = await api.paymentTonInit(paymentIdReturned)
        if (init.deep_link) {
          openExternalLink(init.deep_link)
        }
        setPaymentId(paymentIdReturned)
        return
      }
      if (method.id === "stars") {
        const init = await api.paymentStarsInit(paymentIdReturned)
        if (init.invoice_url) {
          openInvoice(init.invoice_url, (status) => {
            if (status === "paid") onPay()
            else if (status === "failed" || status === "cancelled") {
              setPayError("Payment cancelled")
            }
          })
        }
        return
      }
    } catch (err) {
      setPayError(err instanceof Error ? err.message : "Payment failed")
    }
  }

  return (
    <>
      <SectionHeading icon={CreditCard}>
        {t.subscription.checkout.title}
      </SectionHeading>

      <div
        className={cn(
          "bg-card border-border relative mt-6 overflow-hidden rounded-2xl border bg-gradient-to-br to-transparent p-5 shadow-lg",
          meta.tint,
        )}
      >
        <meta.Icon
          aria-hidden
          strokeWidth={1.2}
          className={cn(
            "pointer-events-none absolute -right-3 -bottom-8 size-32 sm:-right-4 sm:-bottom-10 sm:size-40",
            meta.iconColor,
          )}
        />
        <p
          className={cn(
            "relative inline-flex items-center gap-2 text-xs font-semibold tracking-[0.18em] uppercase",
            meta.kicker,
          )}
        >
          {tierLabel}
        </p>
        <div className="relative mt-2 flex items-start justify-between gap-3">
          <div className="min-w-0 flex-1">
            <h3 className="text-foreground text-xl font-bold tracking-tight sm:text-2xl">
              {plan.name}
            </h3>
            <p className="text-muted-foreground mt-1 text-sm leading-snug sm:text-[15px]">
              {t.subscription.composeDescription(plan.duration_days, plan.traffic_gb)}
            </p>
            <div className="text-muted-foreground mt-3 flex flex-wrap gap-x-3 gap-y-1 text-xs">
              <span className="inline-flex items-center gap-1 tabular-nums">
                <Calendar className="size-3" strokeWidth={2} />
                {t.subscription.daysLabel(plan.duration_days)}
              </span>
              <span className="inline-flex items-center gap-1 tabular-nums">
                <BarChart3 className="size-3" strokeWidth={2} />
                {trafficLabel(plan, t.common.gb)}
              </span>
            </div>
          </div>
          <div className="shrink-0 text-right leading-none">
            {(() => {
              const rub = formatRubPurchase(plan.price_ton, ratesQuery.data)
              if (!rub) {
                return (
                  <p className="text-foreground text-xl font-bold tabular-nums sm:text-2xl">
                    {formatTonPurchase(plan.price_ton)}
                    <span className="text-muted-foreground ml-0.5 text-sm font-medium">
                      TON
                    </span>
                  </p>
                )
              }
              return (
                <>
                  <p className="text-foreground text-xl font-bold tabular-nums sm:text-2xl">
                    ≈{rub}
                    <span className="text-muted-foreground ml-0.5 text-sm font-medium">
                      ₽
                    </span>
                  </p>
                  <p className="text-muted-foreground mt-1.5 text-xs font-medium tabular-nums">
                    {formatTonPurchase(plan.price_ton)} TON
                  </p>
                </>
              )
            })()}
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

      {payError && (
        <p className="mt-4 text-sm text-rose-500">{payError}</p>
      )}
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

function SuccessView({
  onBack,
  showPaymentBanner,
}: {
  onBack: () => void
  showPaymentBanner: boolean
}) {
  useRegisterBack(onBack)
  const { t } = useLocale()
  const keyQuery = useSubscriptionKey()
  const serversQuery = useServers()
  const statusQuery = useSubscriptionStatus()
  const myServerId = statusQuery.data?.subscription?.server_id

  const activeServer =
    (myServerId
      ? serversQuery.data?.find((s) => s.id === myServerId)
      : undefined) ?? serversQuery.data?.[0]
  const serverCode = activeServer?.country?.toUpperCase() ?? ""
  const Flag = serverCode ? flagFor(serverCode) : null
  const regionTint = regionTintFor(serverCode)
  const regionMeta = serverCode
    ? t.region.countries[serverCode as keyof typeof t.region.countries]
    : null
  const serverLocation =
    activeServer?.city?.trim() || activeServer?.name?.trim() || ""

  const connectApps = [
    { platform: t.subscription.success.platformIos, apps: "Streisand, V2Box" },
    { platform: t.subscription.success.platformAndroid, apps: "V2rayNG, NekoBox" },
    { platform: t.subscription.success.platformDesktop, apps: "Nekoray, V2rayN" },
  ]

  const [copied, setCopied] = useState(false)

  const vlessKey = keyQuery.data?.key ?? ""

  const handleCopy = async () => {
    if (!vlessKey) return
    try {
      await navigator.clipboard.writeText(vlessKey)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 1600)
    } catch {
      /* noop */
    }
  }

  return (
    <>
      <SectionHeading icon={KeyRound}>{t.subscription.success.title}</SectionHeading>

      {showPaymentBanner && (
        <motion.div
          initial={{ opacity: 0, y: -8, scale: 0.96 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          transition={{ duration: 0.4, ease: [0.32, 0.72, 0, 1] }}
          className="mt-6 flex items-center gap-3 rounded-2xl border border-emerald-500/40 bg-emerald-500/10 p-4 text-emerald-700 dark:text-emerald-300"
        >
          <span className="inline-flex size-9 shrink-0 items-center justify-center rounded-full border border-emerald-500/40 bg-emerald-500/15">
            <CheckCircle2 className="size-5" strokeWidth={2.5} />
          </span>
          <div className="min-w-0 flex-1">
            <p className="text-foreground text-sm font-semibold">
              {t.subscription.success.banner.title}
            </p>
            <p className="mt-0.5 text-xs leading-snug">
              {t.subscription.success.banner.subtitle}
            </p>
          </div>
        </motion.div>
      )}

      {activeServer && (
        <div
          className={cn(
            "bg-card border-border relative overflow-hidden rounded-2xl border bg-gradient-to-br to-transparent p-5 shadow-lg",
            showPaymentBanner ? "mt-4" : "mt-6",
            regionTint,
          )}
        >
          {Flag && (
            <Flag
              aria-hidden
              className="pointer-events-none absolute -right-3 -bottom-3 h-20 aspect-[3/2] rounded-xl opacity-25 sm:-right-4 sm:-bottom-4 sm:h-24"
            />
          )}
          <div className="relative flex items-center gap-3">
            <div className="min-w-0 flex-1">
              <p className="text-foreground text-xl font-semibold tracking-tight sm:text-2xl">
                {regionMeta?.country ?? serverCode}
              </p>
              {(regionMeta?.city || serverLocation) && (
                <p className="text-muted-foreground mt-0.5 text-sm">
                  {regionMeta?.city ?? serverLocation}
                </p>
              )}
            </div>
            <button
              type="button"
              className="border-border bg-background hover:bg-muted/60 text-foreground inline-flex shrink-0 items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-semibold tracking-tight transition-colors sm:text-sm"
            >
              {t.subscription.success.switch}
              <span className="text-muted-foreground tabular-nums">
                {t.subscription.success.switchPrice}
              </span>
              <ChevronRight
                className="text-muted-foreground size-4"
                strokeWidth={2.4}
              />
            </button>
          </div>
        </div>
      )}

      <div className="bg-card border-border mt-3 flex aspect-square items-center justify-center rounded-2xl border p-5">
        <div className="flex aspect-square h-full w-full items-center justify-center rounded-xl bg-white">
          {keyQuery.isPending && (
            <Loader2 className="text-muted-foreground size-8 animate-spin" />
          )}
          {keyQuery.error && (
            <p className="px-4 text-center text-xs text-rose-500">
              {keyQuery.error instanceof Error
                ? keyQuery.error.message
                : "Failed to load key"}
            </p>
          )}
          {vlessKey && (
            <QRCodeSVG
              value={vlessKey}
              size={Math.round(220)}
              level="M"
              bgColor="#ffffff"
              fgColor="#000000"
              className="h-[88%] w-[88%]"
            />
          )}
        </div>
      </div>

      <div className="bg-card border-border mt-3 rounded-2xl border p-4">
        <p className="text-muted-foreground text-xs font-semibold tracking-[0.18em] uppercase">
          {t.subscription.success.vlessKey}
        </p>
        <p
          className="text-foreground/90 mt-2 font-mono text-[12px] leading-relaxed break-all sm:text-[13px]"
          style={{ fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace" }}
        >
          {keyQuery.isPending ? "…" : vlessKey || "—"}
        </p>
      </div>

      <button
        type="button"
        onClick={handleCopy}
        disabled={!vlessKey}
        className="bg-foreground text-background hover:bg-foreground/90 disabled:bg-muted disabled:text-muted-foreground mt-3 inline-flex h-14 w-full items-center justify-center gap-2 rounded-2xl text-base font-semibold tracking-tight transition-colors disabled:cursor-not-allowed"
      >
        <motion.span
          key={copied ? "check" : "copy"}
          initial={{ opacity: 0, scale: 0.7 }}
          animate={{ opacity: 1, scale: 1 }}
          className="inline-flex items-center gap-2"
        >
          {copied ? (
            <>
              <Check className="size-5" strokeWidth={2.5} />
              {t.common.copied}
            </>
          ) : (
            <>
              <Copy className="size-5" strokeWidth={2} />
              {t.subscription.success.copyKey}
            </>
          )}
        </motion.span>
      </button>

      <h3 className="text-foreground mt-8 text-lg font-semibold tracking-tight">
        {t.subscription.success.howToConnect}
      </h3>
      <ol className="mt-4 space-y-3">
        <li className="bg-card border-border rounded-2xl border p-4">
          <div className="flex items-start gap-3">
            <span className="bg-muted text-foreground inline-flex size-7 shrink-0 items-center justify-center rounded-lg text-xs font-semibold tabular-nums">
              1
            </span>
            <div className="min-w-0 flex-1">
              <p className="text-foreground font-semibold tracking-tight">
                {t.subscription.success.stepTitleInstall}
              </p>
              <ul className="text-muted-foreground mt-2 space-y-1 text-sm">
                {connectApps.map((row) => (
                  <li key={row.platform} className="flex gap-2">
                    <span className="text-foreground/85 w-28 shrink-0 font-medium">
                      {row.platform}
                    </span>
                    <span>{row.apps}</span>
                  </li>
                ))}
              </ul>
            </div>
          </div>
        </li>
        {[
          t.subscription.success.stepCopyKey,
          t.subscription.success.stepAddFromClipboard,
          t.subscription.success.stepConnect,
        ].map((step, i) => (
          <li
            key={step}
            className="bg-card border-border flex items-center gap-3 rounded-2xl border p-4"
          >
            <span className="bg-muted text-foreground inline-flex size-7 shrink-0 items-center justify-center rounded-lg text-xs font-semibold tabular-nums">
              {i + 2}
            </span>
            <p className="text-foreground/90 text-sm">{step}</p>
          </li>
        ))}
      </ol>
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

export type { PlanMeta }
