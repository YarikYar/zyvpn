import type { Plan } from '../types'
import { useStore } from '../store'

interface PlanCardProps {
  plan: Plan
  onSelect: () => void
}

export default function PlanCard({ plan, onSelect }: PlanCardProps) {
  const { rates } = useStore()
  const isPopular = plan.name === 'Pro'

  const usdRub = rates?.usd_rub ?? 95

  return (
    <button
      onClick={onSelect}
      className={`card w-full text-left relative ${isPopular ? 'ring-2 ring-tg-button' : ''}`}
    >
      {isPopular && (
        <span className="absolute -top-2 left-4 bg-tg-button text-tg-button-text text-xs px-2 py-0.5 rounded-full">
          Популярный
        </span>
      )}
      <div className="flex justify-between items-start">
        <div className="flex-1">
          <h3 className="font-semibold text-lg">{plan.name}</h3>
          <p className="text-sm text-hint mt-1">{plan.description}</p>
          <div className="flex gap-3 mt-2 text-xs text-hint">
            <span>📅 {plan.duration_days} дн</span>
            <span>📊 {plan.traffic_gb > 0 ? `${plan.traffic_gb} ГБ` : '∞'}</span>
            <span>📱 {plan.max_devices || 3} устр</span>
          </div>
        </div>
        <div className="text-right ml-4">
          <p className="text-xl font-bold">≈{Math.round(plan.price_usd * usdRub)} ₽</p>
          <p className="text-xs text-hint">${plan.price_usd}</p>
        </div>
      </div>
    </button>
  )
}
