import type { LucideIcon } from "lucide-react"
import { useMemo, useState } from "react"
import { ArrowDown, ArrowUp, Gift, Globe2, History, Loader2, Star, Wallet } from "lucide-react"

import { TonIcon } from "@/components/icons/ton"
import { SectionHeading } from "@/components/sections/section-heading"
import { SegmentedToggle } from "@/components/ui/segmented-toggle"
import { useRegisterBack } from "@/lib/back-button"
import { useLocale, type Locale } from "@/lib/i18n"
import {
  useApplyPromo,
  useBalance,
  useBalanceTransactions,
  useRates,
  useTopupInit,
} from "@/lib/queries"
import type { BalanceTransaction, BalanceTransactionType } from "@/lib/api"
import { formatRubBalance, formatTonBalance, formatTonPurchase } from "@/lib/format"
import { cn } from "@/lib/utils"

type TopUpKind = "ton" | "stars"

const tonAmounts = [0.5, 1, 2, 5] as const
const starsAmounts = [50, 100, 250, 500] as const

type TxKnownKind = BalanceTransactionType
type TxLabelKey = "subscriptionPayment" | "regionSwitch" | "promo" | "topup" | "referralReward"

const txIcon: Record<TxKnownKind, LucideIcon> = {
  purchase: Wallet,
  switch_server: Globe2,
  promo: Gift,
  topup: ArrowDown,
  referral_reward: ArrowUp,
  refund: ArrowUp,
}

const txLabelKey: Record<TxKnownKind, TxLabelKey> = {
  purchase: "subscriptionPayment",
  switch_server: "regionSwitch",
  promo: "promo",
  topup: "topup",
  referral_reward: "referralReward",
  refund: "topup",
}

function txAmount(tx: BalanceTransaction): number {
  const a = tx.amount_ton ?? tx.amount
  return typeof a === "number" && Number.isFinite(a) ? a : 0
}

function txIconFor(tx: BalanceTransaction, amount: number): LucideIcon {
  if (tx.type && tx.type in txIcon) return txIcon[tx.type as TxKnownKind]
  return amount >= 0 ? ArrowDown : ArrowUp
}

function txLabelFor(tx: BalanceTransaction): TxLabelKey | null {
  if (tx.type && tx.type in txLabelKey) return txLabelKey[tx.type as TxKnownKind]
  return null
}

const localeMap: Record<Locale, string> = {
  en: "en-US",
  ru: "ru-RU",
}

function formatTxDate(date: Date, locale: Locale) {
  return new Intl.DateTimeFormat(localeMap[locale], {
    day: "2-digit",
    month: "2-digit",
    year: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date)
}

export function WalletSection({ onClose }: { onClose: () => void }) {
  useRegisterBack(onClose)
  const { t, locale } = useLocale()

  const balanceQuery = useBalance()
  const txQuery = useBalanceTransactions({ limit: 50 })
  const ratesQuery = useRates()
  const applyPromo = useApplyPromo()
  const topupInit = useTopupInit()

  const [promo, setPromo] = useState("")
  const [promoMessage, setPromoMessage] = useState<{ kind: "ok" | "err"; text: string } | null>(null)
  const [topUpKind, setTopUpKind] = useState<TopUpKind>("ton")

  const balance = balanceQuery.data?.balance ?? 0
  const rubApprox = useMemo(
    () => formatRubBalance(balance, ratesQuery.data),
    [balance, ratesQuery.data],
  )

  const presets = topUpKind === "ton" ? tonAmounts : starsAmounts
  const presetUnit = topUpKind === "ton" ? "TON" : t.wallet.topUpVia.stars

  const handleApplyPromo = async () => {
    const code = promo.trim()
    if (!code) return
    setPromoMessage(null)
    try {
      const result = await applyPromo.mutateAsync({ code })
      if (result.ok) {
        setPromoMessage({
          kind: "ok",
          text: result.message ?? t.common.copied,
        })
        setPromo("")
      } else {
        setPromoMessage({ kind: "err", text: result.message ?? "—" })
      }
    } catch (err) {
      setPromoMessage({
        kind: "err",
        text: err instanceof Error ? err.message : "Error",
      })
    }
  }

  const handleTopupClick = async (amount: number) => {
    try {
      const { payment_id } = await topupInit.mutateAsync({
        amount,
        provider: topUpKind,
      })
      console.info("[wallet] topup initiated", { amount, topUpKind, payment_id })
    } catch (err) {
      console.error("[wallet] topup failed", err)
    }
  }

  return (
    <section className="px-6 pb-16">
      <SectionHeading icon={Wallet}>{t.wallet.title}</SectionHeading>

      {/* Hero balance card */}
      <div className="bg-card border-border relative mt-6 overflow-hidden rounded-2xl border bg-gradient-to-br from-sky-500/12 via-blue-500/8 to-transparent p-6 shadow-lg">
        <TonIcon
          aria-hidden
          className="pointer-events-none absolute -right-3 -bottom-6 size-32 opacity-20 sm:-right-4 sm:-bottom-8 sm:size-40"
        />
        <div className="relative flex items-baseline gap-3">
          {balanceQuery.isPending ? (
            <Loader2 className="text-muted-foreground my-2 size-8 animate-spin" />
          ) : (
            <>
              <p className="text-foreground text-5xl font-bold tabular-nums sm:text-6xl">
                {formatTonBalance(balance)}
              </p>
              <span className="inline-flex size-8 shrink-0 items-center justify-center">
                <TonIcon className="size-8" />
              </span>
            </>
          )}
        </div>
        {(ratesQuery.isPending || rubApprox) && (
          <p className="text-muted-foreground relative mt-2 text-sm font-medium tabular-nums">
            {ratesQuery.isPending ? "…" : t.wallet.heroApprox(Number(rubApprox))}
          </p>
        )}
        {balanceQuery.error && (
          <p className="text-rose-500 relative mt-2 text-xs">
            {balanceQuery.error instanceof Error
              ? balanceQuery.error.message
              : "Failed to load balance"}
          </p>
        )}
      </div>

      {/* Promo code */}
      <div className="bg-card border-border mt-3 rounded-2xl border p-5">
        <p className="text-foreground inline-flex items-center gap-2 text-base font-semibold tracking-tight">
          <Gift className="text-muted-foreground size-4" strokeWidth={2} />
          {t.wallet.promoTitle}
        </p>
        <div className="border-border/60 bg-background mt-3 flex items-stretch overflow-hidden rounded-xl border">
          <input
            type="text"
            value={promo}
            onChange={(e) => setPromo(e.target.value)}
            placeholder={t.wallet.promoPlaceholder}
            className="text-foreground placeholder:text-muted-foreground flex-1 bg-transparent px-4 py-3 text-sm outline-none"
            disabled={applyPromo.isPending}
          />
          <button
            type="button"
            onClick={handleApplyPromo}
            disabled={!promo.trim() || applyPromo.isPending}
            className="bg-foreground text-background hover:bg-foreground/90 disabled:bg-muted disabled:text-muted-foreground inline-flex min-w-14 items-center justify-center px-5 text-sm font-semibold transition-colors disabled:cursor-not-allowed"
          >
            {applyPromo.isPending ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              t.wallet.promoSubmit
            )}
          </button>
        </div>
        {promoMessage && (
          <p
            className={cn(
              "mt-2 text-xs",
              promoMessage.kind === "ok"
                ? "text-emerald-600 dark:text-emerald-400"
                : "text-rose-600 dark:text-rose-400",
            )}
          >
            {promoMessage.text}
          </p>
        )}
      </div>

      {/* Top up */}
      <h3 className="text-foreground mt-10 text-lg font-semibold tracking-tight">
        {t.wallet.topUp}
      </h3>
      <div className="mt-3">
        <SegmentedToggle
          value={topUpKind}
          onChange={(v) => setTopUpKind(v as TopUpKind)}
          options={[
            { value: "ton", label: t.wallet.topUpVia.ton, icon: TonIconLucide },
            { value: "stars", label: t.wallet.topUpVia.stars, icon: Star },
          ]}
        />
      </div>
      <div className="mt-3 grid grid-cols-4 gap-2">
        {presets.map((value) => (
          <button
            key={value}
            type="button"
            onClick={() => handleTopupClick(value)}
            disabled={topupInit.isPending}
            className="bg-card border-border hover:bg-muted/40 flex flex-col items-center justify-center rounded-xl border p-3 text-center transition-colors disabled:cursor-not-allowed disabled:opacity-60"
          >
            <span className="text-foreground text-lg font-bold tabular-nums">
              {topUpKind === "ton" ? formatTonPurchase(value) : value}
            </span>
            <span className="text-muted-foreground mt-0.5 text-[10px] font-semibold tracking-[0.18em] uppercase">
              {presetUnit}
            </span>
          </button>
        ))}
      </div>

      {/* History */}
      <h3 className="text-foreground mt-10 inline-flex items-center gap-2 text-lg font-semibold tracking-tight">
        <History className="text-muted-foreground size-4" strokeWidth={2} />
        {t.wallet.history}
      </h3>
      {txQuery.isPending && (
        <div className="mt-3 flex items-center justify-center py-8">
          <Loader2 className="text-muted-foreground size-5 animate-spin" />
        </div>
      )}
      {txQuery.error && (
        <p className="mt-3 text-sm text-rose-500">
          {txQuery.error instanceof Error
            ? txQuery.error.message
            : "Failed to load transactions"}
        </p>
      )}
      {txQuery.data && txQuery.data.length === 0 && (
        <p className="text-muted-foreground mt-3 text-sm">—</p>
      )}
      {txQuery.data && txQuery.data.length > 0 && (
        <ul className="mt-3 space-y-2">
          {txQuery.data.map((tx, i) => {
            const amount = txAmount(tx)
            const positive = amount > 0
            const Icon = txIconFor(tx, amount)
            const labelKey = txLabelFor(tx)
            const fallbackLabel = t.wallet.tx.topup
            const description = tx.description ?? (labelKey ? t.wallet.tx[labelKey] : fallbackLabel)
            const createdAt = tx.created_at ? new Date(tx.created_at) : null
            const dateLabel =
              createdAt && !Number.isNaN(createdAt.getTime())
                ? formatTxDate(createdAt, locale)
                : ""
            return (
              <li
                key={tx.id ?? `${tx.created_at ?? i}`}
                className="bg-card border-border flex items-center gap-3 rounded-2xl border p-4"
              >
                <span
                  className={cn(
                    "inline-flex size-9 shrink-0 items-center justify-center rounded-xl",
                    positive
                      ? "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400"
                      : "bg-rose-500/15 text-rose-600 dark:text-rose-400",
                  )}
                >
                  <Icon className="size-4" strokeWidth={2.25} />
                </span>
                <div className="min-w-0 flex-1">
                  <p className="text-foreground text-sm font-semibold tracking-tight">
                    {description}
                  </p>
                  {dateLabel && (
                    <p className="text-muted-foreground mt-0.5 text-xs tabular-nums">
                      {dateLabel}
                    </p>
                  )}
                </div>
                <p
                  className={cn(
                    "shrink-0 text-sm font-bold tabular-nums",
                    positive
                      ? "text-emerald-600 dark:text-emerald-400"
                      : "text-rose-600 dark:text-rose-400",
                  )}
                >
                  {positive ? "+" : ""}
                  {formatTonBalance(amount)} TON
                </p>
              </li>
            )
          })}
        </ul>
      )}
    </section>
  )
}

const TonIconLucide = TonIcon as unknown as LucideIcon
