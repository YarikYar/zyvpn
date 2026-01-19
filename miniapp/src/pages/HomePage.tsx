import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useStore } from '../store'
import { useTelegram } from '../hooks/useTelegram'
import { api } from '../api/client'
import PlanCard from '../components/PlanCard'
import SubscriptionCard from '../components/SubscriptionCard'

const ONBOARDING_KEY = 'zyvpn_onboarding_seen'

export default function HomePage() {
  const navigate = useNavigate()
  const { user } = useTelegram()
  const { plans, subscriptionStatus, fetchSubscriptionStatus, user: storeUser } = useStore()
  const [isAdmin, setIsAdmin] = useState(false)
  const [showOnboarding, setShowOnboarding] = useState(false)

  useEffect(() => {
    fetchSubscriptionStatus()
    // Check if user is admin
    api.admin.checkAccess().then(setIsAdmin)

    // Show onboarding only on first visit
    if (!localStorage.getItem(ONBOARDING_KEY)) {
      setShowOnboarding(true)
    }
  }, [fetchSubscriptionStatus])

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
