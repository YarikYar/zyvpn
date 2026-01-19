import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { QRCodeSVG } from 'qrcode.react'
import { useStore } from '../store'
import { useTelegram } from '../hooks/useTelegram'
import { api, ServerPublic } from '../api/client'

export default function KeyPage() {
  const navigate = useNavigate()
  const { webApp } = useTelegram()
  const { connectionKey, fetchConnectionKey, subscriptionStatus, fetchSubscriptionStatus } = useStore()
  const [copied, setCopied] = useState(false)
  const [servers, setServers] = useState<ServerPublic[]>([])
  const [switching, setSwitching] = useState(false)
  const [showServerPicker, setShowServerPicker] = useState(false)

  useEffect(() => {
    fetchConnectionKey()
    fetchSubscriptionStatus()
    // Load servers for switching
    api.getServers().then(data => {
      setServers(data.servers || [])
    }).catch(() => {})
  }, [fetchConnectionKey, fetchSubscriptionStatus])

  const handleSwitchServer = async (serverId: string) => {
    setSwitching(true)
    try {
      const result = await api.switchServer(serverId)
      if (result.success) {
        webApp?.HapticFeedback.notificationOccurred('success')
        // Refresh subscription status and key
        await fetchSubscriptionStatus()
        await fetchConnectionKey()
        setShowServerPicker(false)
      }
    } catch (err) {
      webApp?.HapticFeedback.notificationOccurred('error')
      webApp?.showAlert((err as Error).message || 'Не удалось сменить сервер')
    } finally {
      setSwitching(false)
    }
  }

  // Get current server from subscription
  const currentServerId = subscriptionStatus?.subscription?.server_id
  const currentServer = servers.find(s => s.id === currentServerId)

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

      {/* Current Server & Region Switch */}
      {servers.length > 0 && (
        <div className="card mb-4">
          <div className="flex justify-between items-center">
            <div className="flex items-center gap-3">
              <span className="text-2xl">{currentServer?.flag_emoji || '🌍'}</span>
              <div>
                <p className="font-medium">{currentServer?.name || 'Сервер'}</p>
                <p className="text-xs text-hint">
                  {currentServer?.country}{currentServer?.city ? `, ${currentServer.city}` : ''}
                </p>
              </div>
            </div>
            <button
              onClick={() => setShowServerPicker(true)}
              className="bg-tg-secondary-bg px-3 py-2 rounded-xl text-sm font-medium hover:opacity-80 transition-opacity"
              disabled={switching}
            >
              {switching ? 'Переключение...' : 'Сменить регион'}
            </button>
          </div>
        </div>
      )}

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

      {/* Server Picker Modal */}
      {showServerPicker && (
        <div className="fixed inset-0 bg-black/60 flex items-end justify-center z-50">
          <div className="bg-tg-bg rounded-t-2xl p-4 w-full max-w-lg max-h-[80vh] overflow-y-auto animate-slide-up">
            <div className="flex justify-between items-center mb-4">
              <h2 className="text-lg font-bold">Выберите регион</h2>
              <button
                onClick={() => setShowServerPicker(false)}
                className="text-hint text-2xl leading-none"
              >
                ×
              </button>
            </div>
            <p className="text-hint text-sm mb-4">
              После смены региона ваш ключ подключения изменится. Не забудьте обновить его в VPN приложении.
            </p>
            <div className="space-y-2">
              {servers.filter(s => s.status === 'online').map((server) => (
                <button
                  key={server.id}
                  onClick={() => handleSwitchServer(server.id)}
                  disabled={switching || server.id === currentServerId}
                  className={`w-full card flex items-center justify-between p-3 transition-all ${
                    server.id === currentServerId
                      ? 'border-tg-button ring-1 ring-tg-button'
                      : ''
                  }`}
                >
                  <div className="flex items-center gap-3">
                    <span className="text-2xl">{server.flag_emoji}</span>
                    <div className="text-left">
                      <p className="font-medium">{server.name}</p>
                      <p className="text-xs text-hint">
                        {server.country}{server.city ? `, ${server.city}` : ''}
                      </p>
                    </div>
                  </div>
                  <div className="text-right">
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
                  </div>
                </button>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
