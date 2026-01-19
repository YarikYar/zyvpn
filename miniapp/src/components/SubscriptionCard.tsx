import type { Subscription } from '../types'

interface SubscriptionCardProps {
  subscription: Subscription
  daysRemaining: number
  trafficUsed: number
  trafficLimit: number
  onViewKey: () => void
}

export default function SubscriptionCard({
  subscription,
  daysRemaining,
  trafficUsed,
  trafficLimit,
  onViewKey,
}: SubscriptionCardProps) {
  const isUnlimited = trafficLimit <= 0
  const trafficPercent = isUnlimited ? 0 : Math.min((trafficUsed / trafficLimit) * 100, 100)

  // Calculate days progress based on actual subscription duration
  const getTotalDays = () => {
    if (!subscription.started_at || !subscription.expires_at) return 30
    const start = new Date(subscription.started_at).getTime()
    const end = new Date(subscription.expires_at).getTime()
    return Math.max(1, Math.ceil((end - start) / (1000 * 60 * 60 * 24)))
  }
  const totalDays = getTotalDays()
  const daysUsed = totalDays - daysRemaining
  const daysPercent = Math.min(100, Math.max(0, (daysUsed / totalDays) * 100))

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return '-'
    return new Date(dateStr).toLocaleDateString('ru-RU', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
    })
  }

  return (
    <div className="card">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <span className="w-3 h-3 rounded-full bg-green-500"></span>
          <span className="font-semibold">Активна</span>
        </div>
        <button
          onClick={onViewKey}
          className="text-tg-link text-sm font-medium"
        >
          Показать ключ →
        </button>
      </div>

      {/* Days remaining */}
      <div className="mb-4">
        <div className="flex justify-between text-sm mb-1">
          <span className="text-hint">Срок действия</span>
          <span>{daysRemaining} дней</span>
        </div>
        <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
          <div
            className="h-full bg-tg-button rounded-full transition-all"
            style={{ width: `${100 - daysPercent}%` }}
          />
        </div>
        <p className="text-xs text-hint mt-1">До {formatDate(subscription.expires_at)} ({daysRemaining} из {totalDays} дней)</p>
      </div>

      {/* Traffic */}
      <div>
        <div className="flex justify-between text-sm mb-1">
          <span className="text-hint">Трафик</span>
          <span>
            {trafficUsed.toFixed(1)} {isUnlimited ? 'ГБ' : `/ ${trafficLimit} ГБ`}
          </span>
        </div>
        {!isUnlimited && (
          <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
            <div
              className="h-full bg-tg-button rounded-full transition-all"
              style={{ width: `${trafficPercent}%` }}
            />
          </div>
        )}
        {isUnlimited && (
          <p className="text-xs text-hint">Безлимитный трафик</p>
        )}
      </div>
    </div>
  )
}
