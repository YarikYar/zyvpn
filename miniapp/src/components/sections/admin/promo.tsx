import { useState } from "react"
import { Copy, Layers, Plus, Ticket } from "lucide-react"

import { useLocale } from "@/lib/i18n"
import {
  useAdminPromoBulk,
  useAdminPromoCreate,
  useAdminPromoDeactivate,
  useAdminPromos,
} from "@/lib/queries"
import type { AdminPromoType } from "@/lib/api"
import { cn } from "@/lib/utils"

import {
  AdminCard,
  AdminEmpty,
  AdminError,
  AdminLoader,
  DestructiveButton,
  FormField,
  GhostButton,
  PrimaryButton,
  TextInput,
} from "./shared"

const promoTypes: AdminPromoType[] = ["balance", "discount", "days"]

export function AdminPromoView() {
  const { t } = useLocale()
  const promos = useAdminPromos()
  const deactivate = useAdminPromoDeactivate()

  return (
    <div className="mt-6 space-y-6">
      <CreatePromoCard />
      <BulkPromoCard />

      <div>
        <h3 className="text-foreground inline-flex items-center gap-2 text-base font-semibold tracking-tight">
          <Ticket className="text-muted-foreground size-4" strokeWidth={2} />
          {t.admin.sections.promo.title}
        </h3>

        {promos.isPending && <AdminLoader />}
        {promos.error && <AdminError error={promos.error} />}
        {promos.data && promos.data.length === 0 && (
          <AdminEmpty>{t.admin.promo.empty}</AdminEmpty>
        )}

        {promos.data && promos.data.length > 0 && (
          <ul className="mt-3 space-y-2">
            {promos.data.map((p) => {
              const active = p.is_active !== false
              return (
                <li
                  key={p.code}
                  className="bg-card border-border flex items-start gap-3 rounded-2xl border p-4"
                >
                  <span
                    className={cn(
                      "inline-flex size-9 shrink-0 items-center justify-center rounded-xl",
                      active
                        ? "bg-violet-500/15 text-violet-600 dark:text-violet-300"
                        : "bg-muted/60 text-muted-foreground",
                    )}
                  >
                    <Ticket className="size-4" strokeWidth={2.25} />
                  </span>
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <p className="text-foreground truncate font-mono text-sm font-semibold tracking-tight">
                        {p.code}
                      </p>
                      <span
                        className={cn(
                          "inline-flex shrink-0 items-center rounded-full px-1.5 py-0.5 text-[9px] font-semibold tracking-[0.12em] uppercase",
                          active
                            ? "bg-emerald-500/15 text-emerald-600 dark:text-emerald-300"
                            : "bg-muted text-muted-foreground",
                        )}
                      >
                        {active ? t.admin.promo.active : t.admin.promo.inactive}
                      </span>
                    </div>
                    <p className="text-muted-foreground mt-0.5 text-xs tabular-nums">
                      {p.type ? typeLabel(p.type, t) : "—"}
                      {typeof p.amount === "number" && <> · {p.amount}</>}
                    </p>
                    {(typeof p.uses_left === "number" ||
                      p.uses_total !== undefined) && (
                      <p className="text-muted-foreground mt-1 text-[11px] tabular-nums">
                        {t.admin.promo.uses(
                          p.uses_left ?? 0,
                          p.uses_total ?? null,
                        )}
                      </p>
                    )}
                  </div>
                  {active && (
                    <DestructiveButton
                      onClick={() => deactivate.mutate({ code: p.code })}
                      loading={
                        deactivate.isPending &&
                        deactivate.variables?.code === p.code
                      }
                      className="h-9 shrink-0 px-3 text-xs"
                    >
                      {t.admin.promo.deactivate}
                    </DestructiveButton>
                  )}
                </li>
              )
            })}
          </ul>
        )}
      </div>
    </div>
  )
}

function typeLabel(type: AdminPromoType, t: ReturnType<typeof useLocale>["t"]) {
  if (type === "balance") return t.admin.promo.types.balance
  if (type === "discount") return t.admin.promo.types.discount
  if (type === "days") return t.admin.promo.types.days
  return type
}

function CreatePromoCard() {
  const { t } = useLocale()
  const create = useAdminPromoCreate()
  const [code, setCode] = useState("")
  const [type, setType] = useState<AdminPromoType>("balance")
  const [amount, setAmount] = useState("")
  const [uses, setUses] = useState("")
  const [expires, setExpires] = useState("")
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setSuccess(false)
    const amt = Number(amount.replace(",", "."))
    if (!Number.isFinite(amt)) {
      setError(t.admin.common.error)
      return
    }
    try {
      await create.mutateAsync({
        code: code.trim() || undefined,
        type,
        amount: amt,
        uses_total: uses ? Number(uses) : null,
        expires_at: expires.trim() || null,
      })
      setSuccess(true)
      setCode("")
      setAmount("")
      setUses("")
      setExpires("")
    } catch (err) {
      setError(err instanceof Error ? err.message : t.admin.common.error)
    }
  }

  return (
    <AdminCard>
      <p className="text-foreground inline-flex items-center gap-2 text-sm font-semibold tracking-tight">
        <Plus className="text-muted-foreground size-4" strokeWidth={2} />
        {t.admin.promo.createTitle}
      </p>
      <form onSubmit={handleSubmit} className="mt-3 space-y-2.5">
        <FormField label={t.admin.promo.codeLabel}>
          <TextInput
            value={code}
            onChange={(e) => setCode(e.target.value)}
            placeholder={t.admin.promo.codePlaceholder}
          />
        </FormField>
        <FormField label={t.admin.promo.typeLabel}>
          <div className="grid grid-cols-3 gap-2">
            {promoTypes.map((typ) => (
              <button
                key={typ}
                type="button"
                onClick={() => setType(typ)}
                aria-pressed={type === typ}
                className={cn(
                  "border-border/60 rounded-xl border px-2.5 py-2 text-xs font-semibold tracking-tight transition-colors",
                  type === typ
                    ? "bg-foreground text-background border-foreground"
                    : "bg-background text-foreground hover:bg-muted/40",
                )}
              >
                {typeLabel(typ, t)}
              </button>
            ))}
          </div>
        </FormField>
        <FormField label={t.admin.promo.amountLabel}>
          <TextInput
            inputMode="decimal"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            placeholder="0"
          />
        </FormField>
        <FormField label={t.admin.promo.usesLabel}>
          <TextInput
            inputMode="numeric"
            value={uses}
            onChange={(e) => setUses(e.target.value)}
            placeholder={t.admin.promo.usesPlaceholder}
          />
        </FormField>
        <FormField label={t.admin.promo.expiresLabel}>
          <TextInput
            value={expires}
            onChange={(e) => setExpires(e.target.value)}
            placeholder="2026-12-31"
          />
        </FormField>
        <PrimaryButton type="submit" loading={create.isPending} className="w-full">
          {t.admin.promo.createCta}
        </PrimaryButton>
        {error && (
          <p className="text-xs text-rose-500 dark:text-rose-400">{error}</p>
        )}
        {success && (
          <p className="text-xs text-emerald-600 dark:text-emerald-400">
            {t.admin.common.saved}
          </p>
        )}
      </form>
    </AdminCard>
  )
}

function BulkPromoCard() {
  const { t } = useLocale()
  const bulk = useAdminPromoBulk()
  const [type, setType] = useState<AdminPromoType>("balance")
  const [amount, setAmount] = useState("")
  const [count, setCount] = useState("10")
  const [prefix, setPrefix] = useState("")
  const [error, setError] = useState<string | null>(null)
  const [generated, setGenerated] = useState<string[]>([])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setGenerated([])
    const amt = Number(amount.replace(",", "."))
    const cnt = Number(count)
    if (!Number.isFinite(amt) || !Number.isFinite(cnt) || cnt <= 0) {
      setError(t.admin.common.error)
      return
    }
    try {
      const result = await bulk.mutateAsync({
        type,
        amount: amt,
        count: Math.round(cnt),
        prefix: prefix.trim() || undefined,
      })
      setGenerated(result.codes ?? [])
    } catch (err) {
      setError(err instanceof Error ? err.message : t.admin.common.error)
    }
  }

  const handleCopyAll = async () => {
    if (!generated.length) return
    try {
      await navigator.clipboard.writeText(generated.join("\n"))
    } catch {
      /* noop */
    }
  }

  return (
    <AdminCard>
      <p className="text-foreground inline-flex items-center gap-2 text-sm font-semibold tracking-tight">
        <Layers className="text-muted-foreground size-4" strokeWidth={2} />
        {t.admin.promo.bulkTitle}
      </p>
      <form onSubmit={handleSubmit} className="mt-3 space-y-2.5">
        <FormField label={t.admin.promo.typeLabel}>
          <div className="grid grid-cols-3 gap-2">
            {promoTypes.map((typ) => (
              <button
                key={typ}
                type="button"
                onClick={() => setType(typ)}
                aria-pressed={type === typ}
                className={cn(
                  "border-border/60 rounded-xl border px-2.5 py-2 text-xs font-semibold tracking-tight transition-colors",
                  type === typ
                    ? "bg-foreground text-background border-foreground"
                    : "bg-background text-foreground hover:bg-muted/40",
                )}
              >
                {typeLabel(typ, t)}
              </button>
            ))}
          </div>
        </FormField>
        <FormField label={t.admin.promo.amountLabel}>
          <TextInput
            inputMode="decimal"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            placeholder="0"
          />
        </FormField>
        <FormField label={t.admin.promo.bulkCountLabel}>
          <TextInput
            inputMode="numeric"
            value={count}
            onChange={(e) => setCount(e.target.value)}
            placeholder="10"
          />
        </FormField>
        <FormField label={t.admin.promo.bulkPrefixLabel}>
          <TextInput
            value={prefix}
            onChange={(e) => setPrefix(e.target.value)}
            placeholder="WELCOME"
          />
        </FormField>
        <PrimaryButton type="submit" loading={bulk.isPending} className="w-full">
          {t.admin.promo.bulkCta}
        </PrimaryButton>
        {error && (
          <p className="text-xs text-rose-500 dark:text-rose-400">{error}</p>
        )}
      </form>

      {generated.length > 0 && (
        <div className="border-border/60 bg-background mt-4 rounded-xl border p-3">
          <div className="flex items-center justify-between">
            <p className="text-muted-foreground text-[10px] font-semibold tracking-[0.18em] uppercase">
              {t.admin.promo.generatedTitle}
            </p>
            <GhostButton
              type="button"
              onClick={handleCopyAll}
              className="h-7 px-2 text-[11px]"
            >
              <Copy className="size-3" strokeWidth={2} />
              {t.admin.promo.copyAll}
            </GhostButton>
          </div>
          <pre className="text-foreground/85 mt-2 max-h-44 overflow-y-auto whitespace-pre-wrap font-mono text-[12px] leading-relaxed">
            {generated.join("\n")}
          </pre>
        </div>
      )}
    </AdminCard>
  )
}
