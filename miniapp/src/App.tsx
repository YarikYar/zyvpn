import { useEffect } from 'react'
import { Routes, Route } from 'react-router-dom'
import { useTelegram } from './hooks/useTelegram'
import { useStore } from './store'
import HomePage from './pages/HomePage'
import KeyPage from './pages/KeyPage'
import ReferralPage from './pages/ReferralPage'
import PaymentPage from './pages/PaymentPage'
import BalancePage from './pages/BalancePage'
import AdminPage from './pages/AdminPage'

function App() {
  const { webApp, user, initData } = useTelegram()
  const { fetchUser, fetchPlans, fetchRates } = useStore()

  useEffect(() => {
    if (webApp) {
      webApp.ready()
      webApp.expand()
    }
    // Fetch rates immediately (no auth needed)
    fetchRates()
  }, [webApp, fetchRates])

  useEffect(() => {
    if (initData) {
      fetchUser()
      fetchPlans()
    }
  }, [initData, fetchUser, fetchPlans])

  if (!user) {
    // Without Telegram init data we can't authenticate. This usually means
    // the user opened the URL in a regular browser or in Telegram's in-app
    // browser instead of opening the Mini App via the bot.
    const noInitData = !initData
    return (
      <div className="flex items-center justify-center min-h-screen p-6">
        <div className="text-center max-w-sm">
          {noInitData ? (
            <>
              <div className="text-4xl mb-4">🤖</div>
              <h2 className="text-lg font-semibold mb-2">Открой через бота</h2>
              <p className="text-hint mb-4">
                Эта страница работает только внутри Telegram Mini App. Открой бота
                и нажми «Открыть магазин».
              </p>
              <a
                href="https://t.me/zyvpn_bot"
                className="inline-block px-6 py-3 rounded-lg bg-tg-button text-tg-button-text font-medium"
              >
                Открыть @zyvpn_bot
              </a>
            </>
          ) : (
            <>
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-tg-button mx-auto mb-4"></div>
              <p className="text-hint">Loading...</p>
            </>
          )}
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen pb-20">
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/key" element={<KeyPage />} />
        <Route path="/referral" element={<ReferralPage />} />
        <Route path="/payment/:planId" element={<PaymentPage />} />
        <Route path="/balance" element={<BalancePage />} />
        <Route path="/admin" element={<AdminPage />} />
      </Routes>
    </div>
  )
}

export default App
