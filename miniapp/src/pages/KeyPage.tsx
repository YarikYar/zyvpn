import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { QRCodeSVG } from 'qrcode.react'
import { useStore } from '../store'
import { useTelegram } from '../hooks/useTelegram'

export default function KeyPage() {
  const navigate = useNavigate()
  const { webApp } = useTelegram()
  const { connectionKey, fetchConnectionKey, subscriptionStatus, fetchSubscriptionStatus } = useStore()
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    fetchConnectionKey()
    fetchSubscriptionStatus()
  }, [fetchConnectionKey, fetchSubscriptionStatus])

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

  const copyToClipboard = async () => {
    if (connectionKey) {
      try {
        await navigator.clipboard.writeText(connectionKey)
        setCopied(true)
        webApp?.HapticFeedback.notificationOccurred('success')
        setTimeout(() => setCopied(false), 2000)
      } catch {
        webApp?.showAlert('Не удалось скопировать ключ')
      }
    }
  }

  if (!subscriptionStatus?.active) {
    return (
      <div className="p-4">
        <div className="card text-center py-8">
          <span className="text-4xl mb-4 block">🔒</span>
          <h2 className="text-xl font-semibold mb-2">Нет активной подписки</h2>
          <p className="text-hint mb-4">
            Оформите подписку, чтобы получить ключ подключения
          </p>
          <button
            onClick={() => navigate('/')}
            className="btn-primary"
          >
            Выбрать тариф
          </button>
        </div>
      </div>
    )
  }

  if (!connectionKey) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-tg-button"></div>
      </div>
    )
  }

  return (
    <div className="p-4">
      <h1 className="text-2xl font-bold mb-6">Ваш ключ</h1>

      {/* QR Code */}
      <div className="card mb-4 flex justify-center py-6">
        <QRCodeSVG
          value={connectionKey}
          size={200}
          level="M"
          includeMargin
          bgColor="transparent"
          fgColor="currentColor"
        />
      </div>

      {/* Key Display */}
      <div className="card mb-4">
        <p className="text-xs text-hint mb-2">VLESS ключ:</p>
        <p className="text-sm font-mono break-all">{connectionKey}</p>
      </div>

      {/* Copy Button */}
      <button
        onClick={copyToClipboard}
        className="btn-primary w-full mb-4"
      >
        {copied ? '✓ Скопировано!' : 'Скопировать ключ'}
      </button>

      {/* Instructions */}
      <div className="card">
        <h3 className="font-semibold mb-2">Как подключиться:</h3>
        <ol className="text-sm text-hint space-y-2">
          <li>1. Установите приложение для VPN</li>
          <li className="pl-4 text-xs">
            iOS: Streisand, V2Box<br />
            Android: V2rayNG, NekoBox<br />
            Windows/Mac: Nekoray, V2rayN
          </li>
          <li>2. Скопируйте ключ выше</li>
          <li>3. В приложении выберите "Добавить из буфера"</li>
          <li>4. Подключитесь!</li>
        </ol>
      </div>
    </div>
  )
}
