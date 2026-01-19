import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useStore } from '../store'
import { useTelegram } from '../hooks/useTelegram'

export default function ReferralPage() {
  const navigate = useNavigate()
  const { webApp } = useTelegram()
  const { referralStats, referralLink, fetchReferralStats, fetchReferralLink } = useStore()
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    fetchReferralStats()
    fetchReferralLink()
  }, [fetchReferralStats, fetchReferralLink])

  useEffect(() => {
    if (webApp) {
      webApp.BackButton.show()
      webApp.BackButton.onClick(() => navigate('/'))
      return () => {
        webApp.BackButton.hide()
        webApp.BackButton.offClick(() => navigate('/'))
      }
    }
  }, [webApp, navigate])

  const copyLink = async () => {
    if (referralLink) {
      try {
        await navigator.clipboard.writeText(referralLink)
        setCopied(true)
        webApp?.HapticFeedback.notificationOccurred('success')
        setTimeout(() => setCopied(false), 2000)
      } catch {
        webApp?.showAlert('Не удалось скопировать ссылку')
      }
    }
  }

  const shareLink = () => {
    if (referralLink && webApp) {
      const text = 'Присоединяйся к ZyVPN! Быстрый и безопасный VPN.'
      webApp.openTelegramLink(`https://t.me/share/url?url=${encodeURIComponent(referralLink)}&text=${encodeURIComponent(text)}`)
    }
  }

  return (
    <div className="p-4">
      <h1 className="text-2xl font-bold mb-2">Реферальная программа</h1>
      <p className="text-hint mb-6">
        Приглашай друзей и получай % от их платежей!
      </p>

      {/* How it works */}
      <div className="card mb-4">
        <h3 className="font-semibold mb-3">Как это работает:</h3>
        <div className="space-y-3">
          <div className="flex items-start gap-3">
            <span className="text-xl">1️⃣</span>
            <p className="text-sm">Поделись своей ссылкой с другом</p>
          </div>
          <div className="flex items-start gap-3">
            <span className="text-xl">2️⃣</span>
            <p className="text-sm">Друг регистрируется и оплачивает подписку</p>
          </div>
          <div className="flex items-start gap-3">
            <span className="text-xl">3️⃣</span>
            <p className="text-sm">Ты получаешь % от каждого платежа друга на баланс!</p>
          </div>
        </div>
      </div>

      {/* Stats */}
      {referralStats && (
        <div className="grid grid-cols-3 gap-3 mb-6">
          <div className="card text-center">
            <p className="text-2xl font-bold">{referralStats.total_referrals}</p>
            <p className="text-xs text-hint">Приглашено</p>
          </div>
          <div className="card text-center">
            <p className="text-2xl font-bold">{referralStats.pending_referrals}</p>
            <p className="text-xs text-hint">Ожидают</p>
          </div>
          <div className="card text-center">
            <p className="text-2xl font-bold">+{referralStats.credited_bonus_ton.toFixed(2)}</p>
            <p className="text-xs text-hint">TON</p>
          </div>
        </div>
      )}

      {/* Referral Link */}
      {referralLink && (
        <div className="card mb-4">
          <p className="text-xs text-hint mb-2">Ваша ссылка:</p>
          <p className="text-sm font-mono break-all mb-3">{referralLink}</p>
          <div className="grid grid-cols-2 gap-2">
            <button
              onClick={copyLink}
              className="btn-primary text-sm py-2"
            >
              {copied ? '✓ Скопировано' : 'Копировать'}
            </button>
            <button
              onClick={shareLink}
              className="btn-primary text-sm py-2"
            >
              Поделиться
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
