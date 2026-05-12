import { useEffect, useState } from "react"
import {
  ArrowLeftRight,
  Gift,
  TrendingUp,
  Wallet,
} from "lucide-react"
import type { LucideIcon } from "lucide-react"
import type { UseMutationResult } from "@tanstack/react-query"

import { useLocale } from "@/lib/i18n"
import {
  useAdminSettingMutations,
  useAdminSettings,
} from "@/lib/queries"

import {
  AdminCard,
  AdminError,
  AdminLoader,
  MutationFeedback,
  PrimaryButton,
  TextInput,
} from "./shared"

export function AdminSettingsView() {
  const { t } = useLocale()
  const settings = useAdminSettings()
  const m = useAdminSettingMutations()

  if (settings.isPending) return <AdminLoader />
  if (settings.error) return <AdminError error={settings.error} />

  const data = settings.data

  return (
    <div className="mt-6 space-y-3">
      <SettingRow
        icon={Wallet}
        title={t.admin.settingsTab.topupBonus}
        help={t.admin.settingsTab.topupBonusHelp}
        initial={data?.topup_bonus_percent ?? 0}
        unit={t.admin.settingsTab.percentUnit}
        inputMode="decimal"
        mutation={m.topupBonus}
        toBody={(value) => ({ percent: value })}
      />
      <SettingRow
        icon={Gift}
        title={t.admin.settingsTab.referralBonus}
        help={t.admin.settingsTab.referralBonusHelp}
        initial={data?.referral_bonus_percent ?? 0}
        unit={t.admin.settingsTab.percentUnit}
        inputMode="decimal"
        mutation={m.referralBonus}
        toBody={(value) => ({ percent: value })}
      />
      <SettingRow
        icon={TrendingUp}
        title={t.admin.settingsTab.referralBonusDays}
        help={t.admin.settingsTab.referralBonusDaysHelp}
        initial={data?.referral_bonus_days ?? 0}
        unit={t.admin.settingsTab.daysUnit}
        inputMode="numeric"
        mutation={m.referralBonusDays}
        toBody={(value) => ({ days: Math.round(value) })}
      />
      <SettingRow
        icon={ArrowLeftRight}
        title={t.admin.settingsTab.regionSwitchPrice}
        help={t.admin.settingsTab.regionSwitchPriceHelp}
        initial={data?.region_switch_price_ton ?? 0}
        unit={t.admin.settingsTab.tonUnit}
        inputMode="decimal"
        mutation={m.regionSwitchPrice}
        toBody={(value) => ({ price_ton: value })}
      />
    </div>
  )
}

function SettingRow<TBody>({
  icon: Icon,
  title,
  help,
  initial,
  unit,
  inputMode,
  mutation,
  toBody,
}: {
  icon: LucideIcon
  title: string
  help: string
  initial: number
  unit: string
  inputMode: "decimal" | "numeric"
  mutation: UseMutationResult<{ ok: boolean }, Error, TBody>
  toBody: (value: number) => TBody
}) {
  const { t } = useLocale()
  const [value, setValue] = useState(String(initial))

  useEffect(() => {
    setValue(String(initial))
  }, [initial])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const numeric = Number(value.replace(",", "."))
    if (!Number.isFinite(numeric)) return
    await mutation.mutateAsync(toBody(numeric))
  }

  return (
    <AdminCard>
      <div className="flex items-start gap-3">
        <span className="bg-foreground/10 text-foreground inline-flex size-9 shrink-0 items-center justify-center rounded-xl">
          <Icon className="size-4" strokeWidth={2} />
        </span>
        <div className="min-w-0 flex-1">
          <p className="text-foreground text-sm font-semibold tracking-tight">
            {title}
          </p>
          <p className="text-muted-foreground mt-0.5 text-xs leading-snug">
            {help}
          </p>
        </div>
      </div>
      <form
        onSubmit={handleSubmit}
        className="mt-3 flex items-center gap-2"
      >
        <div className="border-border/60 bg-background flex flex-1 items-center overflow-hidden rounded-xl border">
          <TextInput
            inputMode={inputMode}
            value={value}
            onChange={(e) => setValue(e.target.value)}
            className="rounded-none border-0 bg-transparent px-3.5 py-2.5"
          />
          <span className="text-muted-foreground shrink-0 px-3 text-xs font-semibold tabular-nums">
            {unit}
          </span>
        </div>
        <PrimaryButton type="submit" loading={mutation.isPending} className="h-11 shrink-0 px-4">
          {t.admin.settingsTab.save}
        </PrimaryButton>
      </form>
      <MutationFeedback state={mutation} />
    </AdminCard>
  )
}
