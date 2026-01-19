import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTonConnectUI, useTonAddress } from '@tonconnect/ui-react'
import { useStore } from '../store'
import { useTelegram } from '../hooks/useTelegram'
import { api } from '../api/client'
import type { BalanceTransaction } from '../types'

const TOP_UP_AMOUNTS = [0.5, 1, 2, 5]

export default function BalancePage() {
  const navigate = useNavigate()
  const { webApp } = useTelegram()
  const { user, fetchUser, rates } = useStore()
  const [tonConnectUI] = useTonConnectUI()
  const address = useTonAddress()

  const [transactions, setTransactions] = useState<BalanceTransaction[]>([])
  const [loading, setLoading] = useState(false)
  const [topUpAmount, setTopUpAmount] = useState<number | null>(null)
  const [paymentMethod, setPaymentMethod] = useState<'ton' | 'stars'>('ton')
  const [promoCode, setPromoCode] = useState('')
  const [promoLoading, setPromoLoading] = useState(false)
  const [promoError, setPromoError] = useState<string | null>(null)
  const [promoSuccess, setPromoSuccess] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [processingPaymentId, setProcessingPaymentId] = useState<string | null>(null)

  const balance = user?.balance ?? 0
  const tonRub = rates?.ton_rub ?? 475

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

  useEffect(() => {
    loadTransactions()
  }, [])

  // Poll for payment status when processing
  useEffect(() => {
    if (!processingPaymentId) return

    const pollStatus = async () => {
      try {
        const status = await api.getPaymentStatus(processingPaymentId)

        if (status.status === 'completed') {
          setProcessingPaymentId(null)
          setLoading(false)
          webApp?.HapticFeedback.notificationOccurred('success')
          webApp?.showAlert(`Баланс пополнен на ${topUpAmount} TON!`)
          fetchUser()
          loadTransactions()
          setTopUpAmount(null)
        } else if (status.status === 'failed') {
          setProcessingPaymentId(null)
          setLoading(false)
          setError('Транзакция не найдена или истекло время ожидания')
          webApp?.HapticFeedback.notificationOccurred('error')
        }
        // If still awaiting_tx or pending, continue polling
      } catch (err) {
        console.error('Failed to poll payment status:', err)
      }
    }

    const interval = setInterval(pollStatus, 3000) // Poll every 3 seconds
    pollStatus() // Initial poll

    return () => clearInterval(interval)
  }, [processingPaymentId, topUpAmount, webApp, fetchUser])

  const loadTransactions = async () => {
    try {
      const data = await api.getBalanceTransactions(10, 0)
      setTransactions(data.transactions || [])
    } catch (err) {
      console.error('Failed to load transactions:', err)
    }
  }

  const handleTopUp = async () => {
    if (!topUpAmount) return

    setLoading(true)
    setError(null)

    try {
      if (paymentMethod === 'ton') {
        // TON payment flow
        if (!address) {
          await tonConnectUI.openModal()
          setLoading(false)
          return
        }

        // Create top-up payment on backend
        const { payment_id } = await api.initTopUp(topUpAmount, 'ton')

        // Get payment info
        const tonInfo = await api.getTopUpTONInfo(payment_id)

        // Send TON transaction (without payload - simple transfer)
        const amountNano = (topUpAmount * 1e9).toString()

        const transaction = {
          validUntil: Math.floor(Date.now() / 1000) + 600,
          messages: [
            {
              address: tonInfo.wallet_address,
              amount: amountNano,
            },
          ],
        }

        const result = await tonConnectUI.sendTransaction(transaction)

        // Submit for verification (backend will verify via worker)
        await api.verifyTopUp(payment_id, result.boc)

        // Start polling for status
        setProcessingPaymentId(payment_id)
        // Keep loading=true, polling effect will handle completion
        return
      } else {
        // Stars payment - create invoice
        const { payment_id } = await api.initTopUp(topUpAmount, 'stars')

        // Get invoice link from backend
        const invoiceResponse = await api.initTopUpStars(payment_id)

        if (invoiceResponse.invoice_link && webApp) {
          // Open Telegram Stars payment
          webApp.openInvoice(invoiceResponse.invoice_link, async (status) => {
            if (status === 'paid') {
              webApp.HapticFeedback.notificationOccurred('success')
              webApp.showAlert(`Баланс пополнен на ${topUpAmount} TON!`)
              fetchUser()
              loadTransactions()
              setTopUpAmount(null)
            } else if (status === 'cancelled') {
              setError('Оплата отменена')
            } else if (status === 'failed') {
              setError('Ошибка оплаты')
            }
          })
        } else {
          setError('Не удалось создать счёт для оплаты')
        }
      }
    } catch (err) {
      setError((err as Error).message)
      webApp?.HapticFeedback.notificationOccurred('error')
    } finally {
      setLoading(false)
    }
  }

  const handleApplyPromo = async () => {
    if (!promoCode.trim()) return

    setPromoLoading(true)
    setPromoError(null)
    setPromoSuccess(null)

    try {
      const result = await api.applyPromoCode(promoCode.trim())
      setPromoSuccess(result.message)
      setPromoCode('')
      webApp?.HapticFeedback.notificationOccurred('success')

      // Refresh data
      fetchUser()
      loadTransactions()
    } catch (err) {
      setPromoError((err as Error).message)
      webApp?.HapticFeedback.notificationOccurred('error')
    } finally {
      setPromoLoading(false)
    }
  }

  const getTransactionTypeLabel = (type: string) => {
    switch (type) {
      case 'referral_bonus': return 'Реферальный бонус'
      case 'giveaway': return 'Розыгрыш'
      case 'subscription_payment': return 'Оплата подписки'
      case 'refund': return 'Возврат'
      case 'top_up': return 'Пополнение'
      case 'promo_code': return 'Промокод'
      default: return type
    }
  }

  return (
    <div className="p-4">
      <h1 className="text-2xl font-bold mb-2">Баланс</h1>

      {/* Balance Card */}
      <div className="card mb-6 text-center py-6">
        <p className="text-4xl font-bold mb-1">💎 {balance.toFixed(4)}</p>
        <p className="text-hint">TON ≈ {Math.round(balance * tonRub)} ₽</p>
      </div>

      {/* Promo Code Section */}
      <div className="card mb-6">
        <h2 className="text-lg font-semibold mb-3">Промокод</h2>
        <div className="flex gap-2">
          <input
            type="text"
            value={promoCode}
            onChange={(e) => setPromoCode(e.target.value.toUpperCase())}
            placeholder="Введите промокод"
            className="flex-1 input"
            disabled={promoLoading}
          />
          <button
            onClick={handleApplyPromo}
            disabled={promoLoading || !promoCode.trim()}
            className="btn-primary px-4"
          >
            {promoLoading ? '...' : 'OK'}
          </button>
        </div>
        {promoError && (
          <p className="text-red-500 text-sm mt-2">{promoError}</p>
        )}
        {promoSuccess && (
          <p className="text-green-500 text-sm mt-2">{promoSuccess}</p>
        )}
      </div>

      {/* Top Up Section */}
      <div className="mb-6">
        <h2 className="text-lg font-semibold mb-3">Пополнить баланс</h2>

        {/* Payment method selector */}
        <div className="flex gap-2 mb-3">
          <button
            onClick={() => setPaymentMethod('ton')}
            className={`flex-1 py-2 rounded-xl text-sm transition-all ${
              paymentMethod === 'ton'
                ? 'bg-tg-button text-white'
                : 'bg-tg-secondary-bg'
            }`}
          >
            💎 TON
          </button>
          <button
            onClick={() => setPaymentMethod('stars')}
            className={`flex-1 py-2 rounded-xl text-sm transition-all ${
              paymentMethod === 'stars'
                ? 'bg-tg-button text-white'
                : 'bg-tg-secondary-bg'
            }`}
          >
            ⭐ Stars
          </button>
        </div>

        <div className="grid grid-cols-4 gap-2 mb-3">
          {TOP_UP_AMOUNTS.map((amount) => (
            <button
              key={amount}
              onClick={() => setTopUpAmount(amount)}
              className={`card py-3 text-center transition-all ${
                topUpAmount === amount ? 'ring-2 ring-tg-button' : ''
              }`}
            >
              <p className="font-semibold">{amount}</p>
              <p className="text-xs text-hint">TON</p>
            </button>
          ))}
        </div>

        {topUpAmount && (
          <div className="space-y-3">
            <div className="card">
              <div className="flex justify-between">
                <span className="text-hint">Сумма:</span>
                <span className="font-semibold">{topUpAmount} TON</span>
              </div>
              <div className="flex justify-between mt-1">
                <span className="text-hint">≈</span>
                <span>{Math.round(topUpAmount * tonRub)} ₽</span>
              </div>
              {paymentMethod === 'stars' && (
                <div className="flex justify-between mt-1">
                  <span className="text-hint">В Stars:</span>
                  <span>~{Math.round(topUpAmount * 100)} XTR</span>
                </div>
              )}
            </div>

            {error && (
              <div className="bg-red-100 text-red-700 p-3 rounded-xl text-sm">
                {error}
              </div>
            )}

            <button
              onClick={handleTopUp}
              disabled={loading}
              className="btn-primary w-full"
            >
              {loading ? (
                <span className="flex items-center justify-center gap-2">
                  <span className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></span>
                  {processingPaymentId ? 'Ожидание подтверждения...' : 'Обработка...'}
                </span>
              ) : paymentMethod === 'ton' ? (
                address ? `Пополнить ${topUpAmount} TON` : 'Подключить кошелёк'
              ) : (
                `Пополнить через Stars`
              )}
            </button>

            {paymentMethod === 'ton' && address && (
              <p className="text-center text-hint text-xs">
                Кошелек: {address.slice(0, 6)}...{address.slice(-4)}
              </p>
            )}
          </div>
        )}
      </div>

      {/* Transactions History */}
      <div>
        <h2 className="text-lg font-semibold mb-3">История операций</h2>
        {transactions.length === 0 ? (
          <div className="card text-center py-6">
            <p className="text-hint">Нет операций</p>
          </div>
        ) : (
          <div className="space-y-2">
            {transactions.map((tx) => (
              <div key={tx.id} className="card flex justify-between items-center">
                <div>
                  <p className="font-medium">{getTransactionTypeLabel(tx.type)}</p>
                  <p className="text-xs text-hint">
                    {new Date(tx.created_at).toLocaleDateString('ru-RU', {
                      day: '2-digit',
                      month: '2-digit',
                      year: '2-digit',
                      hour: '2-digit',
                      minute: '2-digit',
                    })}
                  </p>
                </div>
                <p className={`font-semibold ${tx.amount > 0 ? 'text-green-600' : 'text-red-600'}`}>
                  {tx.amount > 0 ? '+' : ''}{tx.amount.toFixed(4)} TON
                </p>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
