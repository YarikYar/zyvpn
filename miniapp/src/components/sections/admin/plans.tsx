import { useState } from "react"
import { Pencil, Plus, Star, Trash2 } from "lucide-react"
import { AnimatePresence, motion } from "motion/react"

import { useLocale } from "@/lib/i18n"
import {
  useAdminPlanCreate,
  useAdminPlanDelete,
  useAdminPlanUpdate,
  useAdminPlans,
} from "@/lib/queries"
import type { AdminPlanInput, Plan } from "@/lib/api"
import { formatTonBalance } from "@/lib/format"
import { cn } from "@/lib/utils"

import {
  AdminEmpty,
  AdminError,
  AdminLoader,
  DestructiveButton,
  FormField,
  GhostButton,
  MutationFeedback,
  PrimaryButton,
  TextArea,
  TextInput,
  Toggle,
} from "./shared"

type Mode = { kind: "list" } | { kind: "form"; plan: Plan | null }

export function AdminPlansView() {
  const { t } = useLocale()
  const plans = useAdminPlans()
  const del = useAdminPlanDelete()
  const [mode, setMode] = useState<Mode>({ kind: "list" })

  return (
    <div className="mt-6">
      {mode.kind === "list" && (
        <>
          <PrimaryButton
            onClick={() => setMode({ kind: "form", plan: null })}
            className="w-full"
          >
            <Plus className="size-4" strokeWidth={2.25} />
            {t.admin.plans.createCta}
          </PrimaryButton>

          {plans.isPending && <AdminLoader />}
          {plans.error && <AdminError error={plans.error} />}
          {plans.data && plans.data.length === 0 && (
            <AdminEmpty>{t.admin.plans.empty}</AdminEmpty>
          )}

          {plans.data && plans.data.length > 0 && (
            <ul className="mt-4 space-y-2">
              {plans.data.map((p) => {
                const active = p.is_active !== false
                return (
                  <li
                    key={p.id}
                    className="bg-card border-border rounded-2xl border p-4"
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0 flex-1">
                        <div className="flex flex-wrap items-center gap-1.5">
                          <p className="text-foreground truncate text-base font-semibold tracking-tight">
                            {p.name}
                          </p>
                          {p.popular && (
                            <span className="inline-flex items-center gap-0.5 rounded-full bg-amber-500/15 px-1.5 py-0.5 text-[9px] font-semibold tracking-[0.12em] text-amber-700 uppercase dark:text-amber-300">
                              <Star className="size-2.5" strokeWidth={2.5} />
                              {t.admin.plans.popularLabel}
                            </span>
                          )}
                          <span
                            className={cn(
                              "inline-flex items-center rounded-full px-1.5 py-0.5 text-[9px] font-semibold tracking-[0.12em] uppercase",
                              active
                                ? "bg-emerald-500/15 text-emerald-600 dark:text-emerald-300"
                                : "bg-muted text-muted-foreground",
                            )}
                          >
                            {active ? t.admin.plans.active : t.admin.plans.inactive}
                          </span>
                        </div>
                        <p className="text-muted-foreground mt-0.5 truncate text-xs tabular-nums">
                          {p.id} · {p.duration_days}d ·{" "}
                          {p.traffic_gb === null
                            ? "∞"
                            : `${p.traffic_gb} GB`}{" "}
                          · {formatTonBalance(p.price_ton)} TON
                        </p>
                        {p.description && (
                          <p className="text-muted-foreground mt-2 line-clamp-2 text-xs">
                            {p.description}
                          </p>
                        )}
                      </div>
                    </div>
                    <div className="mt-3 grid grid-cols-2 gap-2">
                      <GhostButton
                        className="h-9 text-xs"
                        onClick={() => setMode({ kind: "form", plan: p })}
                      >
                        <Pencil className="size-3.5" strokeWidth={2} />
                        {t.admin.plans.editCta}
                      </GhostButton>
                      <DestructiveButton
                        className="h-9 text-xs"
                        onClick={() => {
                          if (window.confirm(t.admin.plans.confirmDelete)) {
                            del.mutate(p.id)
                          }
                        }}
                        loading={del.isPending && del.variables === p.id}
                      >
                        <Trash2 className="size-3.5" strokeWidth={2} />
                        {t.admin.plans.deleteCta}
                      </DestructiveButton>
                    </div>
                  </li>
                )
              })}
            </ul>
          )}
        </>
      )}

      <AnimatePresence>
        {mode.kind === "form" && (
          <motion.div
            key="plan-form"
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 8 }}
            transition={{ duration: 0.2 }}
          >
            <PlanForm
              plan={mode.plan}
              onCancel={() => setMode({ kind: "list" })}
              onSaved={() => setMode({ kind: "list" })}
            />
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}

function PlanForm({
  plan,
  onCancel,
  onSaved,
}: {
  plan: Plan | null
  onCancel: () => void
  onSaved: () => void
}) {
  const { t } = useLocale()
  const create = useAdminPlanCreate()
  const update = useAdminPlanUpdate()

  const [id, setId] = useState(plan?.id ?? "")
  const [name, setName] = useState(plan?.name ?? "")
  const [durationDays, setDurationDays] = useState(
    String(plan?.duration_days ?? 30),
  )
  const [trafficGb, setTrafficGb] = useState(
    plan?.traffic_gb === null ? "" : String(plan?.traffic_gb ?? ""),
  )
  const [unlimited, setUnlimited] = useState(plan?.traffic_gb === null)
  const [priceTon, setPriceTon] = useState(String(plan?.price_ton ?? ""))
  const [priceStars, setPriceStars] = useState(String(plan?.price_stars ?? ""))
  const [maxDevices, setMaxDevices] = useState(String(plan?.max_devices ?? ""))
  const [description, setDescription] = useState(plan?.description ?? "")
  const [isActive, setIsActive] = useState(plan?.is_active !== false)
  const [popular, setPopular] = useState(!!plan?.popular)
  const [referrerId, setReferrerId] = useState(
    plan?.visible_to_referrer_id
      ? String(plan.visible_to_referrer_id)
      : "",
  )

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const input: AdminPlanInput = {
      id: id.trim() || undefined,
      name: name.trim(),
      duration_days: Number(durationDays),
      traffic_gb: unlimited ? null : Number(trafficGb),
      price_ton: Number(priceTon.replace(",", ".")),
      price_stars: priceStars ? Number(priceStars) : undefined,
      max_devices: maxDevices ? Number(maxDevices) : undefined,
      description: description.trim() || undefined,
      is_active: isActive,
      popular,
      visible_to_referrer_id: referrerId ? Number(referrerId) : null,
    }
    try {
      if (plan) {
        await update.mutateAsync({ id: plan.id, input })
      } else {
        await create.mutateAsync(input)
      }
      onSaved()
    } catch {
      /* MutationFeedback will show error */
    }
  }

  const busy = create.isPending || update.isPending
  const mutationState = plan ? update : create

  return (
    <form
      onSubmit={handleSubmit}
      className="bg-card border-border space-y-3 rounded-2xl border p-5"
    >
      <h3 className="text-foreground text-base font-semibold tracking-tight">
        {plan ? t.admin.plans.updateTitle : t.admin.plans.createTitle}
      </h3>

      {!plan && (
        <FormField label={t.admin.plans.fields.id}>
          <TextInput
            value={id}
            onChange={(e) => setId(e.target.value)}
            placeholder="basic"
          />
        </FormField>
      )}
      <FormField label={t.admin.plans.fields.name}>
        <TextInput
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
        />
      </FormField>
      <div className="grid grid-cols-2 gap-3">
        <FormField label={t.admin.plans.fields.durationDays}>
          <TextInput
            inputMode="numeric"
            value={durationDays}
            onChange={(e) => setDurationDays(e.target.value)}
            required
          />
        </FormField>
        <FormField label={t.admin.plans.fields.priceTon}>
          <TextInput
            inputMode="decimal"
            value={priceTon}
            onChange={(e) => setPriceTon(e.target.value)}
            required
          />
        </FormField>
      </div>

      <FormField label={t.admin.plans.fields.trafficGb}>
        <div className="flex items-center gap-3">
          <TextInput
            inputMode="numeric"
            value={trafficGb}
            disabled={unlimited}
            onChange={(e) => setTrafficGb(e.target.value)}
            placeholder="100"
            className="flex-1"
          />
          <label className="inline-flex shrink-0 items-center gap-2 text-xs">
            <Toggle checked={unlimited} onChange={setUnlimited} />
            <span className="text-muted-foreground">
              {t.admin.plans.fields.unlimitedTraffic}
            </span>
          </label>
        </div>
      </FormField>

      <div className="grid grid-cols-2 gap-3">
        <FormField label={t.admin.plans.fields.priceStars}>
          <TextInput
            inputMode="numeric"
            value={priceStars}
            onChange={(e) => setPriceStars(e.target.value)}
          />
        </FormField>
        <FormField label={t.admin.plans.fields.maxDevices}>
          <TextInput
            inputMode="numeric"
            value={maxDevices}
            onChange={(e) => setMaxDevices(e.target.value)}
          />
        </FormField>
      </div>

      <FormField label={t.admin.plans.fields.description}>
        <TextArea
          rows={3}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
      </FormField>

      <FormField label={t.admin.plans.fields.visibleToReferrerId}>
        <TextInput
          inputMode="numeric"
          value={referrerId}
          onChange={(e) => setReferrerId(e.target.value)}
          placeholder="0"
        />
      </FormField>

      <div className="border-border/60 bg-background flex items-center justify-between rounded-xl border px-3.5 py-3">
        <span className="text-foreground text-sm font-medium">
          {t.admin.plans.fields.isActive}
        </span>
        <Toggle checked={isActive} onChange={setIsActive} />
      </div>
      <div className="border-border/60 bg-background flex items-center justify-between rounded-xl border px-3.5 py-3">
        <span className="text-foreground text-sm font-medium">
          {t.admin.plans.fields.popular}
        </span>
        <Toggle checked={popular} onChange={setPopular} />
      </div>

      <MutationFeedback state={mutationState} />

      <div className="grid grid-cols-2 gap-2 pt-2">
        <GhostButton type="button" onClick={onCancel} disabled={busy}>
          {t.admin.plans.cancelCta}
        </GhostButton>
        <PrimaryButton type="submit" loading={busy}>
          {t.admin.plans.saveCta}
        </PrimaryButton>
      </div>
    </form>
  )
}
