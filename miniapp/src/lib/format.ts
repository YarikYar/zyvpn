export function formatTonPurchase(value: number | undefined | null): string {
  if (typeof value !== "number" || !Number.isFinite(value)) return "0.00"
  return (Math.ceil(value * 100) / 100).toFixed(2)
}

export function formatTonBalance(value: number | undefined | null): string {
  if (typeof value !== "number" || !Number.isFinite(value)) return "0.00"
  return (Math.floor(value * 100) / 100).toFixed(2)
}

type RatesLike = { ton_rub?: number } | null | undefined

function rubFromTon(
  ton: number | undefined | null,
  rates: RatesLike,
  rounder: (n: number) => number,
): number | null {
  if (typeof ton !== "number" || !Number.isFinite(ton)) return null
  const rate = rates?.ton_rub
  if (typeof rate !== "number" || !Number.isFinite(rate) || rate <= 0) return null
  return rounder(ton * rate)
}

export function formatRubPurchase(ton: number | undefined | null, rates: RatesLike): string | null {
  const value = rubFromTon(ton, rates, Math.ceil)
  return value === null ? null : String(value)
}

export function formatRubBalance(ton: number | undefined | null, rates: RatesLike): string | null {
  const value = rubFromTon(ton, rates, Math.floor)
  return value === null ? null : String(value)
}
