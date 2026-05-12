import { useState } from "react"
import { Banknote, Check, X } from "lucide-react"

import { useLocale } from "@/lib/i18n"
import {
  useAdminCashApprove,
  useAdminCashPending,
  useAdminCashReject,
} from "@/lib/queries"
import { formatTonBalance } from "@/lib/format"

import {
  AdminEmpty,
  AdminError,
  AdminLoader,
  DestructiveButton,
  PrimaryButton,
  formatAdminDate,
} from "./shared"

export function AdminCashView() {
  const { t, locale } = useLocale()
  const pending = useAdminCashPending()
  const approve = useAdminCashApprove()
  const reject = useAdminCashReject()
  const [busyId, setBusyId] = useState<string | null>(null)

  if (pending.isPending) return <AdminLoader />
  if (pending.error) return <AdminError error={pending.error} />
  if (!pending.data || pending.data.length === 0)
    return <AdminEmpty>{t.admin.cash.empty}</AdminEmpty>

  const handleApprove = async (id: string) => {
    setBusyId(id)
    try {
      await approve.mutateAsync(id)
    } finally {
      setBusyId(null)
    }
  }

  const handleReject = async (id: string) => {
    const reason = window.prompt(t.admin.cash.rejectReason) ?? undefined
    setBusyId(id)
    try {
      await reject.mutateAsync({ id, reason: reason || undefined })
    } finally {
      setBusyId(null)
    }
  }

  return (
    <ul className="mt-6 space-y-2.5">
      {pending.data.map((p) => {
        const busy = busyId === p.id
        const userLabel = p.username
          ? `@${p.username}`
          : p.first_name || `ID ${p.user_id}`
        return (
          <li
            key={p.id}
            className="bg-card border-border relative overflow-hidden rounded-2xl border bg-gradient-to-br from-emerald-500/10 via-emerald-500/3 to-transparent p-4 shadow-md"
          >
            <Banknote
              aria-hidden
              strokeWidth={1.2}
              className="pointer-events-none absolute -right-2 -bottom-3 size-24 text-emerald-500/20"
            />
            <div className="relative flex items-start justify-between gap-3">
              <div className="min-w-0 flex-1">
                <p className="text-muted-foreground text-[10px] font-semibold tracking-[0.18em] uppercase">
                  {t.admin.cash.plan}
                </p>
                <p className="text-foreground mt-0.5 truncate text-base font-semibold tracking-tight">
                  {p.plan_name || p.plan_id}
                </p>
                <p className="text-muted-foreground mt-2 truncate text-xs tabular-nums">
                  {userLabel} · ID {p.user_id}
                </p>
                {p.created_at && (
                  <p className="text-muted-foreground mt-0.5 text-[11px] tabular-nums">
                    {t.admin.cash.submitted}: {formatAdminDate(p.created_at, locale)}
                  </p>
                )}
                {p.note && (
                  <p className="text-muted-foreground mt-2 line-clamp-2 text-xs">
                    {p.note}
                  </p>
                )}
              </div>
              <div className="shrink-0 text-right">
                <p className="text-foreground text-base font-bold tabular-nums">
                  {typeof p.amount_rub === "number"
                    ? `${p.amount_rub} ₽`
                    : typeof p.amount_ton === "number"
                      ? `${formatTonBalance(p.amount_ton)} TON`
                      : "—"}
                </p>
              </div>
            </div>

            <div className="relative mt-4 grid grid-cols-2 gap-2">
              <DestructiveButton
                onClick={() => handleReject(p.id)}
                loading={busy && reject.isPending}
              >
                <X className="size-4" strokeWidth={2.25} />
                {t.admin.cash.reject}
              </DestructiveButton>
              <PrimaryButton
                onClick={() => handleApprove(p.id)}
                loading={busy && approve.isPending}
              >
                <Check className="size-4" strokeWidth={2.25} />
                {t.admin.cash.approve}
              </PrimaryButton>
            </div>
          </li>
        )
      })}
    </ul>
  )
}
