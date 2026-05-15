import { useEffect, useState } from "react"
import {
  Ban,
  Calendar,
  CalendarPlus,
  Check,
  ChevronRight,
  CircleSlash,
  Copy,
  KeyRound,
  Link2,
  Loader2,
  Plus,
  RotateCcw,
  Search,
  ShieldAlert,
  ShieldCheck,
  User,
  Wallet,
  X,
} from "lucide-react"
import { AnimatePresence, motion } from "motion/react"

import { ServerChipsRow } from "@/components/server-chip"
import { useLocale } from "@/lib/i18n"
import {
  useAdminUser,
  useAdminUserMutations,
  useAdminUserSubscription,
  useAdminUsers,
} from "@/lib/queries"
import { formatTonBalance } from "@/lib/format"
import { copyToClipboard, triggerHaptic } from "@/lib/telegram"
import { cn } from "@/lib/utils"
import type { AdminUser } from "@/lib/api"

import {
  AdminCard,
  AdminEmpty,
  AdminError,
  AdminLoader,
  DestructiveButton,
  GhostButton,
  PrimaryButton,
  formatAdminDate,
  formatAdminDateShort,
} from "./shared"

export function AdminUsersView() {
  const { t } = useLocale()
  const [search, setSearch] = useState("")
  const [debounced, setDebounced] = useState("")
  const [selected, setSelected] = useState<AdminUser | null>(null)

  useEffect(() => {
    const id = window.setTimeout(() => setDebounced(search), 320)
    return () => window.clearTimeout(id)
  }, [search])

  const listQuery = useAdminUsers({
    search: debounced || undefined,
    limit: 50,
  })

  return (
    <div className="mt-6">
      <label className="border-border bg-card relative flex items-center gap-2 rounded-2xl border px-4 py-3">
        <Search className="text-muted-foreground size-4" strokeWidth={2} />
        <input
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder={t.admin.users.searchPlaceholder}
          className="text-foreground placeholder:text-muted-foreground flex-1 bg-transparent text-sm outline-none"
        />
        {search && (
          <button
            type="button"
            onClick={() => setSearch("")}
            className="text-muted-foreground hover:text-foreground transition-colors"
          >
            <X className="size-4" strokeWidth={2.25} />
          </button>
        )}
      </label>

      {listQuery.isPending && <AdminLoader />}
      {listQuery.error && <AdminError error={listQuery.error} />}

      {listQuery.data && listQuery.data.length === 0 && (
        <AdminEmpty>{t.admin.users.empty}</AdminEmpty>
      )}

      {listQuery.data && listQuery.data.length > 0 && (
        <ul className="mt-3 space-y-2">
          {listQuery.data.map((u) => (
            <li key={u.id}>
              <button
                type="button"
                onClick={() => setSelected(u)}
                className="border-border bg-card hover:bg-muted/40 flex w-full items-center gap-3 rounded-2xl border p-4 text-left transition-colors"
              >
                <span
                  className={cn(
                    "inline-flex size-10 shrink-0 items-center justify-center rounded-xl",
                    u.is_banned
                      ? "bg-rose-500/15 text-rose-600 dark:text-rose-300"
                      : "bg-foreground/10 text-foreground",
                  )}
                >
                  <User className="size-4" strokeWidth={2} />
                </span>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <p className="text-foreground truncate text-sm font-semibold tracking-tight">
                      {displayName(u)}
                    </p>
                    {u.is_banned && (
                      <span className="inline-flex shrink-0 items-center gap-1 rounded-full bg-rose-500/15 px-1.5 py-0.5 text-[9px] font-semibold tracking-[0.12em] text-rose-600 uppercase dark:text-rose-300">
                        {t.admin.users.banned}
                      </span>
                    )}
                  </div>
                  <p className="text-muted-foreground mt-0.5 truncate text-xs tabular-nums">
                    ID {u.id}
                    {typeof u.balance_ton === "number" && (
                      <> · {formatTonBalance(u.balance_ton)} TON</>
                    )}
                  </p>
                </div>
                <ChevronRight className="text-muted-foreground size-4 shrink-0" strokeWidth={2} />
              </button>
            </li>
          ))}
        </ul>
      )}

      <AnimatePresence>
        {selected && (
          <UserSheet
            key={`user-${selected.id}`}
            user={selected}
            onClose={() => setSelected(null)}
          />
        )}
      </AnimatePresence>
    </div>
  )
}

function displayName(u: AdminUser): string {
  if (u.username) return `@${u.username}`
  const name = [u.first_name, u.last_name].filter(Boolean).join(" ").trim()
  return name || `ID ${u.id}`
}

function UserSheet({
  user,
  onClose,
}: {
  user: AdminUser
  onClose: () => void
}) {
  const { t, locale } = useLocale()
  const fullQuery = useAdminUser(user.id)
  const u = fullQuery.data ?? user
  const m = useAdminUserMutations(user.id)
  const [pendingLabel, setPendingLabel] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const isBusy =
    m.balanceAdd.isPending ||
    m.balanceSet.isPending ||
    m.subscriptionCancel.isPending ||
    m.subscriptionExtend.isPending ||
    m.rotateSubscriptionToken.isPending ||
    m.ban.isPending ||
    m.unban.isPending

  const wrap = async (label: string, fn: () => Promise<unknown>) => {
    setError(null)
    setPendingLabel(label)
    try {
      await fn()
    } catch (err) {
      setError(err instanceof Error ? err.message : t.admin.common.error)
    } finally {
      setPendingLabel(null)
    }
  }

  const askNumber = (label: string): number | null => {
    const raw = window.prompt(label)
    if (raw === null) return null
    const v = Number(raw.replace(",", "."))
    return Number.isFinite(v) ? v : null
  }

  const askText = (label: string): string | null => window.prompt(label)

  const handleAddBalance = () =>
    wrap(t.admin.users.actions.addBalance, async () => {
      const amount = askNumber(t.admin.users.promptAmount)
      if (amount === null) return
      const reason = askText(t.admin.users.promptReason) ?? undefined
      await m.balanceAdd.mutateAsync({ amount, reason: reason || undefined })
    })

  const handleSetBalance = () =>
    wrap(t.admin.users.actions.setBalance, async () => {
      const amount = askNumber(t.admin.users.promptAmount)
      if (amount === null) return
      const reason = askText(t.admin.users.promptReason) ?? undefined
      await m.balanceSet.mutateAsync({ amount, reason: reason || undefined })
    })

  const handleExtend = () =>
    wrap(t.admin.users.actions.extendDays, async () => {
      const days = askNumber(t.admin.users.promptDays)
      if (days === null) return
      await m.subscriptionExtend.mutateAsync({ days: Math.round(days) })
    })

  const handleCancelSub = () =>
    wrap(t.admin.users.actions.cancelSubscription, async () => {
      if (!window.confirm(t.admin.users.confirmCancel)) return
      await m.subscriptionCancel.mutateAsync()
    })

  const handleRotateToken = () =>
    wrap(t.admin.users.actions.rotateToken, async () => {
      if (!window.confirm(t.admin.users.confirmRotate)) return
      await m.rotateSubscriptionToken.mutateAsync()
    })

  const handleBan = () =>
    wrap(t.admin.users.actions.ban, async () => {
      if (!window.confirm(t.admin.users.confirmBan)) return
      const reason = askText(t.admin.users.promptReason) ?? undefined
      await m.ban.mutateAsync({ reason: reason || undefined })
    })

  const handleUnban = () =>
    wrap(t.admin.users.actions.unban, async () => {
      if (!window.confirm(t.admin.users.confirmUnban)) return
      await m.unban.mutateAsync()
    })

  return (
    <motion.div
      key="user-overlay"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={{ duration: 0.18 }}
      className="fixed inset-0 z-[60] flex items-end justify-center bg-black/40 backdrop-blur-sm sm:items-center"
      onClick={onClose}
    >
      <motion.div
        initial={{ y: 36, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        exit={{ y: 36, opacity: 0 }}
        transition={{ type: "spring", stiffness: 320, damping: 32 }}
        className="bg-background border-border max-h-[88dvh] w-full overflow-y-auto rounded-t-3xl border-t p-5 shadow-xl sm:max-w-md sm:rounded-3xl sm:border"
        onClick={(e) => e.stopPropagation()}
        style={{ scrollbarWidth: "none" }}
      >
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0 flex-1">
            <p className="text-muted-foreground text-[10px] font-semibold tracking-[0.18em] uppercase">
              ID {u.id}
            </p>
            <h3 className="text-foreground mt-0.5 text-xl font-semibold tracking-tight">
              {displayName(u)}
            </h3>
            {u.is_banned && (
              <span className="mt-2 inline-flex items-center gap-1 rounded-full bg-rose-500/15 px-2 py-0.5 text-[10px] font-semibold tracking-[0.12em] text-rose-600 uppercase dark:text-rose-300">
                <ShieldAlert className="size-3" strokeWidth={2.5} />
                {t.admin.users.banned}
              </span>
            )}
          </div>
          <button
            type="button"
            onClick={onClose}
            className="text-muted-foreground hover:text-foreground -mr-1 -mt-1 inline-flex size-9 items-center justify-center rounded-full transition-colors"
          >
            <X className="size-4" strokeWidth={2.25} />
          </button>
        </div>

        <div className="mt-5 grid grid-cols-2 gap-3">
          <UserDetail
            icon={Wallet}
            label={t.admin.users.balance}
            value={
              typeof u.balance_ton === "number"
                ? `${formatTonBalance(u.balance_ton)} TON`
                : "—"
            }
          />
          <UserDetail
            icon={Calendar}
            label={t.admin.users.subscription}
            value={
              u.subscription_plan
                ? u.subscription_plan
                : t.admin.users.noSubscription
            }
            meta={
              u.subscription_ends_at
                ? formatAdminDateShort(u.subscription_ends_at, locale)
                : undefined
            }
          />
          <UserDetail
            icon={CalendarPlus}
            label={t.admin.users.joined}
            value={formatAdminDate(u.created_at, locale) || "—"}
          />
          <UserDetail
            icon={User}
            label={t.admin.users.lastSeen}
            value={formatAdminDate(u.last_seen_at, locale) || "—"}
          />
        </div>

        {error && (
          <p className="mt-3 text-xs text-rose-500 dark:text-rose-400">
            {error}
          </p>
        )}
        {pendingLabel && (
          <p className="text-muted-foreground mt-3 inline-flex items-center gap-1.5 text-xs">
            <Loader2 className="size-3.5 animate-spin" />
            {pendingLabel}
          </p>
        )}

        <UserSubscriptionCard userId={user.id} />

        <div className="mt-5 grid grid-cols-2 gap-2.5">
          <PrimaryButton onClick={handleAddBalance} disabled={isBusy}>
            <Plus className="size-4" strokeWidth={2.25} />
            {t.admin.users.actions.addBalance}
          </PrimaryButton>
          <GhostButton onClick={handleSetBalance} disabled={isBusy}>
            <Wallet className="size-4" strokeWidth={2} />
            {t.admin.users.actions.setBalance}
          </GhostButton>
          <PrimaryButton onClick={handleExtend} disabled={isBusy}>
            <CalendarPlus className="size-4" strokeWidth={2.25} />
            {t.admin.users.actions.extendDays}
          </PrimaryButton>
          <GhostButton onClick={handleCancelSub} disabled={isBusy}>
            <CircleSlash className="size-4" strokeWidth={2} />
            {t.admin.users.actions.cancelSubscription}
          </GhostButton>
          <GhostButton
            className="col-span-2"
            onClick={handleRotateToken}
            disabled={isBusy}
          >
            <RotateCcw className="size-4" strokeWidth={2} />
            {t.admin.users.actions.rotateToken}
          </GhostButton>
          {u.is_banned ? (
            <PrimaryButton
              className="col-span-2"
              onClick={handleUnban}
              disabled={isBusy}
            >
              <ShieldCheck className="size-4" strokeWidth={2.25} />
              {t.admin.users.actions.unban}
            </PrimaryButton>
          ) : (
            <DestructiveButton
              className="col-span-2"
              onClick={handleBan}
              disabled={isBusy}
            >
              <Ban className="size-4" strokeWidth={2.25} />
              {t.admin.users.actions.ban}
            </DestructiveButton>
          )}
        </div>
      </motion.div>
    </motion.div>
  )
}

function UserSubscriptionCard({ userId }: { userId: number }) {
  const { t } = useLocale()
  const subQuery = useAdminUserSubscription(userId)
  const [copied, setCopied] = useState(false)
  const sub = subQuery.data ?? null
  const url = sub?.subscription_url ?? ""
  const servers = sub?.servers ?? []

  const handleCopy = async () => {
    if (!url) return
    const ok = await copyToClipboard(url)
    if (!ok) return
    triggerHaptic({ type: "notification", status: "success" })
    setCopied(true)
    window.setTimeout(() => setCopied(false), 1600)
  }

  return (
    <div className="border-border bg-card mt-5 rounded-2xl border p-4">
      <div className="flex items-center justify-between gap-2">
        <p className="text-muted-foreground inline-flex items-center gap-1.5 text-[10px] font-semibold tracking-[0.18em] uppercase">
          <Link2 className="size-3" strokeWidth={2.5} />
          {t.admin.users.subscriptionCard.title}
        </p>
        {subQuery.isPending && (
          <Loader2 className="text-muted-foreground size-3.5 animate-spin" />
        )}
      </div>
      {subQuery.isError && (
        <p className="text-muted-foreground mt-2 text-xs">
          {t.admin.users.subscriptionCard.empty}
        </p>
      )}
      {sub && !url && (
        <p className="text-muted-foreground mt-2 text-xs">
          {t.admin.users.subscriptionCard.empty}
        </p>
      )}
      {url && (
        <>
          <p
            className="text-foreground/90 mt-2 font-mono text-[11px] leading-relaxed break-all sm:text-[12px]"
            style={{ fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace" }}
          >
            {url}
          </p>
          <p className="text-muted-foreground mt-2 text-[11px] leading-snug">
            {t.admin.users.subscriptionCard.helper}
          </p>
          <div className="mt-3 flex items-center justify-between gap-2">
            {servers.length > 0 && (
              <p className="text-muted-foreground inline-flex items-center gap-1.5 text-[11px] tabular-nums">
                <KeyRound className="size-3" strokeWidth={2.25} />
                {t.admin.users.subscriptionCard.serversLabel(servers.length)}
              </p>
            )}
            <button
              type="button"
              onClick={handleCopy}
              className="border-border bg-background hover:bg-muted/60 text-foreground inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-[11px] font-semibold tracking-tight transition-colors"
            >
              {copied ? (
                <>
                  <Check className="size-3" strokeWidth={2.5} />
                  {t.admin.users.subscriptionCard.copied}
                </>
              ) : (
                <>
                  <Copy className="size-3" strokeWidth={2} />
                  {t.admin.users.subscriptionCard.copy}
                </>
              )}
            </button>
          </div>
          {servers.length > 0 && (
            <ServerChipsRow servers={servers} className="mt-3" max={6} />
          )}
        </>
      )}
    </div>
  )
}

function UserDetail({
  icon: Icon,
  label,
  value,
  meta,
}: {
  icon: React.ComponentType<React.SVGProps<SVGSVGElement>>
  label: string
  value: string
  meta?: string
}) {
  return (
    <AdminCard className="border-border/60 bg-background p-3.5">
      <div className="flex items-center gap-1.5">
        <Icon className="text-muted-foreground size-3.5" strokeWidth={2} />
        <p className="text-muted-foreground text-[10px] font-semibold tracking-[0.16em] uppercase">
          {label}
        </p>
      </div>
      <p className="text-foreground mt-1 truncate text-sm font-semibold tabular-nums">
        {value}
      </p>
      {meta && (
        <p className="text-muted-foreground mt-0.5 truncate text-[11px] tabular-nums">
          {meta}
        </p>
      )}
    </AdminCard>
  )
}
