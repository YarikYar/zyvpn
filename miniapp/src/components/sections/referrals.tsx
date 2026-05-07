import { useState } from "react"
import {
  Check,
  Copy,
  CreditCard,
  Gift,
  Loader2,
  Send,
  Share2,
  UserPlus,
  Users,
  Wallet,
  type LucideIcon,
} from "lucide-react"
import { motion } from "motion/react"

import { TonIcon } from "@/components/icons/ton"
import { SectionHeading } from "@/components/sections/section-heading"
import { formatTonBalance } from "@/lib/format"
import { useLocale } from "@/lib/i18n"
import {
  useReferralLink,
  useReferralStats,
} from "@/lib/queries"
import { cn } from "@/lib/utils"

const REFERRAL_TINT = "from-violet-500/12 via-fuchsia-500/8"
const REWARD_PERCENT = 20

type StepKey = "share" | "friend" | "reward"

const stepIcons: Record<StepKey, LucideIcon> = {
  share: Send,
  friend: UserPlus,
  reward: Wallet,
}

const stepKeys: StepKey[] = ["share", "friend", "reward"]

export function ReferralsSection() {
  const { t } = useLocale()
  const linkQuery = useReferralLink()
  const statsQuery = useReferralStats()

  const referralLink = linkQuery.data?.link ?? ""

  const [copied, setCopied] = useState(false)

  const handleCopy = async () => {
    if (!referralLink) return
    try {
      await navigator.clipboard.writeText(referralLink)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 1600)
    } catch {
      /* noop */
    }
  }

  const handleShare = async () => {
    if (!referralLink) return
    const tg = (window as { Telegram?: { WebApp?: { openTelegramLink?: (url: string) => void } } }).Telegram?.WebApp
    if (tg?.openTelegramLink) {
      const shareUrl = `https://t.me/share/url?url=${encodeURIComponent(referralLink)}&text=${encodeURIComponent(t.referrals.shareText)}`
      tg.openTelegramLink(shareUrl)
      return
    }
    if (navigator.share) {
      try {
        await navigator.share({
          title: "ZYVPN",
          text: t.referrals.shareText,
          url: referralLink,
        })
      } catch {
        /* user cancelled */
      }
      return
    }
    handleCopy()
  }

  return (
    <section className="px-6 pb-16">
      <SectionHeading icon={Gift}>{t.referrals.title}</SectionHeading>
      <p className="text-muted-foreground mt-3 max-w-md text-base sm:text-[17px]">
        {t.referrals.intro(REWARD_PERCENT)}
      </p>

      {/* Hero balance tile */}
      <div
        className={cn(
          "border-border bg-card relative mt-8 overflow-hidden rounded-2xl border bg-gradient-to-br to-transparent p-5 shadow-lg",
          REFERRAL_TINT,
        )}
      >
        <Gift
          aria-hidden
          strokeWidth={1.2}
          className="pointer-events-none absolute -right-3 -bottom-8 size-36 text-violet-500/20 sm:-right-4 sm:-bottom-10 sm:size-44"
        />
        <p className="text-muted-foreground text-xs font-semibold tracking-[0.18em] uppercase">
          {t.referrals.earnedSoFar}
        </p>
        <div className="mt-2 flex items-end gap-2">
          {statsQuery.isPending ? (
            <Loader2 className="text-muted-foreground my-2 size-8 animate-spin" />
          ) : (
            <>
              <p className="text-foreground text-4xl font-bold tabular-nums sm:text-5xl">
                +{formatTonBalance(statsQuery.data?.ton_earned ?? 0)}
              </p>
              <span className="mb-1 inline-flex size-7 items-center justify-center rounded-full">
                <TonIcon className="size-7" />
              </span>
            </>
          )}
        </div>
        {statsQuery.error && (
          <p className="text-rose-500 mt-2 text-xs">
            {statsQuery.error instanceof Error
              ? statsQuery.error.message
              : "Failed to load stats"}
          </p>
        )}
        <div className="mt-4 grid grid-cols-2 gap-3">
          <Stat
            icon={Users}
            label={t.referrals.invited}
            value={statsQuery.data?.invited ?? 0}
            loading={statsQuery.isPending}
          />
          <Stat
            icon={CreditCard}
            label={t.referrals.pending}
            value={
              (statsQuery.data?.invited ?? 0) -
              (statsQuery.data?.paid ?? 0)
            }
            loading={statsQuery.isPending}
          />
        </div>
      </div>

      {/* How it works */}
      <h3 className="text-foreground mt-10 text-lg font-semibold tracking-tight">
        {t.referrals.howItWorks}
      </h3>
      <ol className="mt-4 space-y-2.5">
        {stepKeys.map((key, i) => {
          const Icon = stepIcons[key]
          const meta = t.referrals.steps[key]
          const description =
            key === "reward"
              ? t.referrals.steps.reward.description(REWARD_PERCENT)
              : (meta as { description: string }).description
          return (
            <li
              key={key}
              className="border-border bg-card flex items-start gap-3.5 rounded-2xl border p-4"
            >
              <span className="border-border bg-muted/50 text-foreground/85 inline-flex size-9 shrink-0 items-center justify-center rounded-xl border">
                <Icon className="size-4" strokeWidth={2} />
              </span>
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="text-muted-foreground text-[10px] font-semibold tracking-[0.18em] tabular-nums uppercase">
                    {t.referrals.stepLabel(i + 1)}
                  </span>
                </div>
                <p className="text-foreground mt-0.5 text-[15px] font-semibold tracking-tight">
                  {meta.title}
                </p>
                <p className="text-muted-foreground mt-0.5 text-sm leading-snug">
                  {description}
                </p>
              </div>
            </li>
          )
        })}
      </ol>

      {/* Invite link card */}
      <div className="border-border bg-card mt-6 rounded-2xl border p-4">
        <p className="text-muted-foreground text-xs font-semibold tracking-[0.18em] uppercase">
          {t.referrals.yourInviteLink}
        </p>
        <div className="border-border/60 bg-muted/40 mt-2 overflow-hidden rounded-xl border">
          {linkQuery.isPending ? (
            <p className="text-muted-foreground inline-flex items-center gap-2 px-3.5 py-3 text-sm">
              <Loader2 className="size-4 animate-spin" /> …
            </p>
          ) : linkQuery.error ? (
            <p className="text-rose-500 px-3.5 py-3 text-sm">
              {linkQuery.error instanceof Error
                ? linkQuery.error.message
                : "Failed to load link"}
            </p>
          ) : (
            <p
              className="text-foreground/90 px-3.5 py-3 font-mono text-[13px] break-all sm:text-sm"
              style={{ fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace" }}
            >
              {referralLink}
            </p>
          )}
        </div>
        <div className="mt-3 grid grid-cols-2 gap-2">
          <button
            type="button"
            onClick={handleCopy}
            disabled={!referralLink}
            className={cn(
              "bg-foreground text-background hover:bg-foreground/90 disabled:bg-muted disabled:text-muted-foreground inline-flex h-11 items-center justify-center gap-2 rounded-xl text-sm font-semibold tracking-tight transition-colors disabled:cursor-not-allowed",
            )}
          >
            <motion.span
              key={copied ? "check" : "copy"}
              initial={{ opacity: 0, scale: 0.7 }}
              animate={{ opacity: 1, scale: 1 }}
              className="inline-flex items-center gap-2"
            >
              {copied ? (
                <>
                  <Check className="size-4" strokeWidth={2.5} />
                  {t.common.copied}
                </>
              ) : (
                <>
                  <Copy className="size-4" strokeWidth={2} />
                  {t.common.copy}
                </>
              )}
            </motion.span>
          </button>
          <button
            type="button"
            onClick={handleShare}
            disabled={!referralLink}
            className="border-border bg-card hover:bg-muted/60 text-foreground disabled:opacity-60 inline-flex h-11 items-center justify-center gap-2 rounded-xl border text-sm font-semibold tracking-tight transition-colors disabled:cursor-not-allowed"
          >
            <Share2 className="size-4" strokeWidth={2} />
            {t.common.share}
          </button>
        </div>
      </div>
    </section>
  )
}

function Stat({
  icon: Icon,
  label,
  value,
  loading,
}: {
  icon: LucideIcon
  label: string
  value: number
  loading?: boolean
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
        {loading ? <Loader2 className="size-4 animate-spin" /> : value}
      </p>
    </div>
  )
}
