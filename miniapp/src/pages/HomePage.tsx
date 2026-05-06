import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useStore } from '../store'
import { useTelegram } from '../hooks/useTelegram'
import { api, ServerPublic, Incident, DailyUptime } from '../api/client'
import PlanCard from '../components/PlanCard'
import SubscriptionCard from '../components/SubscriptionCard'
import UptimeBars from '../components/UptimeBars'

const ONBOARDING_KEY = 'zyvpn_onboarding_seen'

export default function HomePage() {
  const navigate = useNavigate()
  const { user } = useTelegram()
  const { plans, subscriptionStatus, fetchSubscriptionStatus, fetchPlans, user: storeUser, selectedServerId, setSelectedServerId } = useStore()
  const [isAdmin, setIsAdmin] = useState(false)
  const [showOnboarding, setShowOnboarding] = useState(false)
  const [servers, setServers] = useState<ServerPublic[]>([])
  const [incidents, setIncidents] = useState<Incident[]>([])
  const [dailyUptime, setDailyUptime] = useState<DailyUptime[]>([])

  useEffect(() => {
    fetchSubscriptionStatus()
    fetchPlans() // Refresh plans on every visit
    // Check if user is admin
    api.admin.checkAccess().then(setIsAdmin)

    // Load servers
    api.getServers().then(data => {
      setServers(data.servers || [])
      // Auto-select first online server if none selected
      if (!selectedServerId) {
        const onlineServer = data.servers?.find(s => s.status === 'online')
        if (onlineServer) {
          setSelectedServerId(onlineServer.id)
        }
      }
    }).catch(() => {})

    api.getIncidents(168, 5).then(data => setIncidents(data.incidents || [])).catch(() => {})

    // Show onboarding only on first visit
    if (!localStorage.getItem(ONBOARDING_KEY)) {
      setShowOnboarding(true)
    }
  }, [fetchSubscriptionStatus, fetchPlans, selectedServerId, setSelectedServerId])

  // Daily uptime chart for the currently selected server.
  useEffect(() => {
    if (!selectedServerId) {
      setDailyUptime([])
      return
    }
    api.getDailyUptime(selectedServerId, 30)
      .then(data => setDailyUptime(data.days || []))
      .catch(() => setDailyUptime([]))
  }, [selectedServerId])

  const handleCloseOnboarding = () => {
    localStorage.setItem(ONBOARDING_KEY, 'true')
    setShowOnboarding(false)
  }

  const handleGoToBalance = () => {
    localStorage.setItem(ONBOARDING_KEY, 'true')
    setShowOnboarding(false)
    navigate('/balance')
  }

  const balance = storeUser?.balance ?? 0

  return (
    <div className="p-4">
      {/* Header */}
      <div className="mb-6">
        <div className="flex justify-between items-start">
          <div>
            <h1 className="text-2xl font-bold">ZyVPN</h1>
            <p className="text-hint mt-1">
              {user?.first_name ? `Привет, ${user.first_name}!` : 'Быстрый и безопасный VPN'}
            </p>
          </div>
          <button
            onClick={() => navigate('/balance')}
            className="bg-tg-secondary-bg px-3 py-1.5 rounded-xl text-right hover:opacity-80 transition-opacity"
          >
            <p className="text-xs text-hint">Баланс</p>
            <p className="font-semibold">💎 {balance.toFixed(2)} TON</p>
          </button>
        </div>
      </div>

      {/* Active Subscription */}
      {subscriptionStatus?.active && subscriptionStatus.subscription && (
        <div className="mb-6">
          <h2 className="text-lg font-semibold mb-3">Ваша подписка</h2>
          <SubscriptionCard
            subscription={subscriptionStatus.subscription}
            daysRemaining={subscriptionStatus.days_remaining}
            trafficUsed={subscriptionStatus.traffic_gb.used}
            trafficLimit={subscriptionStatus.traffic_gb.limit}
            onViewKey={() => navigate('/key')}
          />
        </div>
      )}

      {/* Server Selection */}
      {servers.length > 0 && (
        <div className="mb-6">
          <h2 className="text-lg font-semibold mb-3">Регион</h2>
          <div className="space-y-2">
            {servers.map((server) => (
              <button
                key={server.id}
                onClick={() => setSelectedServerId(server.id)}
                className={`w-full card flex items-center justify-between p-3 transition-all ${
                  selectedServerId === server.id
                    ? 'border-tg-button ring-1 ring-tg-button'
                    : ''
                } ${server.status !== 'online' ? 'opacity-50' : ''}`}
                disabled={server.status !== 'online'}
              >
                <div className="flex items-center gap-3">
                  <span className="text-2xl">{server.flag_emoji}</span>
                  <div className="text-left">
                    <p className="font-medium">{server.name}</p>
                    <p className="text-xs text-hint">{server.country}{server.city ? `, ${server.city}` : ''}</p>
                    {server.uptime_24h !== undefined && (
                      <p className={`text-[10px] ${
                        server.uptime_24h >= 0.99 ? 'text-green-500' :
                        server.uptime_24h >= 0.95 ? 'text-yellow-500' :
                        'text-red-500'
                      }`}>
                        🟢 {(server.uptime_24h * 100).toFixed(1)}% / 24ч
                      </p>
                    )}
                  </div>
                </div>
                <div className="text-right">
                  {server.status === 'online' ? (
                    <>
                      <p className={`font-medium ${
                        server.ping_ms && server.ping_ms < 100 ? 'text-green-500' :
                        server.ping_ms && server.ping_ms < 200 ? 'text-yellow-500' :
                        'text-red-500'
                      }`}>
                        {server.ping_ms ? `${server.ping_ms} ms` : '...'}
                      </p>
                      <p className="text-xs text-hint">
                        {server.load_percent > 80 ? 'Загружен' : server.load_percent > 50 ? 'Средняя' : 'Низкая'} нагрузка
                      </p>
                    </>
                  ) : (
                    <>
                      <p className="text-xs text-red-500">Оффлайн</p>
                      {server.status_since && (
                        <p className="text-[10px] text-hint">
                          {humanDuration(Date.now() - new Date(server.status_since).getTime())}
                        </p>
                      )}
                    </>
                  )}
                </div>
              </button>
            ))}
          </div>
          {dailyUptime.length > 0 && (
            <div className="mt-3 card p-3">
              <div className="flex justify-between items-baseline mb-2">
                <p className="text-xs text-hint">Аптайм за 30 дней</p>
                <p className="text-[10px] text-hint">сегодня →</p>
              </div>
              <UptimeBars days={dailyUptime} />
            </div>
          )}
        </div>
      )}

      {/* Recent incidents */}
      {incidents.length > 0 && (
        <div className="mb-6">
          <h2 className="text-lg font-semibold mb-3">Последние сбои</h2>
          <div className="space-y-2">
            {incidents.map((inc, i) => (
              <div key={i} className="card p-3 flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <span className="text-xl">{inc.flag_emoji}</span>
                  <div>
                    <p className="text-sm">{inc.server_name}</p>
                    <p className="text-[10px] text-hint">
                      {new Date(inc.started_at).toLocaleString('ru-RU')}
                    </p>
                  </div>
                </div>
                <div className="text-right">
                  {inc.ended_at ? (
                    <p className="text-xs text-hint">
                      {humanDuration(inc.duration_seconds * 1000)}
                    </p>
                  ) : (
                    <p className="text-xs text-red-500">сейчас</p>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Plans */}
      <div className="mb-6">
        <h2 className="text-lg font-semibold mb-3">
          {subscriptionStatus?.active ? 'Продлить подписку' : 'Выберите тариф'}
        </h2>
        <div className="space-y-3">
          {plans.map((plan) => (
            <PlanCard
              key={plan.id}
              plan={plan}
              onSelect={() => navigate(`/payment/${plan.id}`)}
            />
          ))}
        </div>
      </div>

      {/* Quick Actions */}
      <div className="grid grid-cols-2 gap-3">
        <button
          onClick={() => navigate('/key')}
          className="card flex flex-col items-center justify-center py-6"
        >
          <span className="text-2xl mb-2">🔑</span>
          <span className="font-medium">Ключ</span>
        </button>
        <button
          onClick={() => navigate('/referral')}
          className="card flex flex-col items-center justify-center py-6"
        >
          <span className="text-2xl mb-2">🎁</span>
          <span className="font-medium">Рефералы</span>
        </button>
      </div>

      {/* Admin Panel Link */}
      {isAdmin && (
        <button
          onClick={() => navigate('/admin')}
          className="mt-4 w-full card flex items-center justify-center gap-2 py-3 bg-red-500/10 border border-red-500/30"
        >
          <span className="text-lg">⚙️</span>
          <span className="font-medium text-red-500">Admin Panel</span>
        </button>
      )}

      {/* Onboarding Modal */}
      {showOnboarding && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4">
          <div className="bg-tg-bg rounded-2xl p-6 max-w-sm w-full shadow-xl">
            <div className="text-center mb-4">
              <span className="text-5xl">🎁</span>
            </div>
            <h2 className="text-xl font-bold text-center mb-2">
              Добро пожаловать в ZyVPN!
            </h2>
            <p className="text-hint text-center mb-4">
              У тебя есть промокод? Активируй его на странице баланса и получи бонус!
            </p>
            <div className="bg-tg-secondary-bg rounded-xl p-3 mb-4">
              <div className="flex items-center gap-3">
                <span className="text-2xl">💎</span>
                <div>
                  <p className="font-medium">Баланс → Промокод</p>
                  <p className="text-xs text-hint">Введи код и нажми OK</p>
                </div>
              </div>
            </div>
            <div className="flex gap-2">
              <button
                onClick={handleCloseOnboarding}
                className="flex-1 py-3 rounded-xl bg-tg-secondary-bg font-medium"
              >
                Позже
              </button>
              <button
                onClick={handleGoToBalance}
                className="flex-1 btn-primary"
              >
                Ввести промокод
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

// humanDuration formats milliseconds as a short Russian string:
// "5 мин", "2 ч", "3 д". For very small durations falls back to seconds.
function humanDuration(ms: number): string {
  if (ms < 0) ms = 0
  const sec = Math.round(ms / 1000)
  if (sec < 60) return `${sec} с`
  const min = Math.round(sec / 60)
  if (min < 60) return `${min} мин`
  const hr = Math.round(min / 60)
  if (hr < 24) return `${hr} ч`
  const days = Math.round(hr / 24)
  return `${days} д`
}
