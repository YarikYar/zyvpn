import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useTonConnectUI, useTonAddress } from '@tonconnect/ui-react'
import { useStore } from '../store'
import { useTelegram } from '../hooks/useTelegram'
import { api } from '../api/client'
import type { TONPaymentInfo } from '../types'

export default function PaymentPage() {
  const { planId } = useParams<{ planId: string }>()
  const navigate = useNavigate()
  const { webApp } = useTelegram()
  const { plans, rates, user, fetchUser } = useStore()
  const [tonConnectUI] = useTonConnectUI()
  const address = useTonAddress()

  const [selectedPayment, setSelectedPayment] = useState<'ton' | 'stars' | 'balance'>('ton')
  const [loading, setLoading] = useState(false)
  const [paymentInfo, setPaymentInfo] = useState<TONPaymentInfo | null>(null)
  const [error, setError] = useState<string | null>(null)

  const plan = plans.find(p => p.id === planId)
  const userBalance = user?.balance ?? 0
  const canPayFromBalance = userBalance >= (plan?.price_ton ?? 0)

  // Exchange rates with fallbacks
  const tonRub = rates?.ton_rub ?? 475
  const usdRub = rates?.usd_rub ?? 95
  // Stars: 1 star ≈ $0.02 (50 stars = $1)
  const starRub = usdRub / 50

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

  if (!plan) {
    return (
      <div className="p-4">
        <div className="card text-center py-8">
          <p className="text-hint">Тариф не найден</p>
          <button onClick={() => navigate('/')} className="btn-primary mt-4">
            Назад
          </button>
        </div>
      </div>
    )
  }

  const handlePayment = async () => {
    if (!planId) return

    setLoading(true)
    setError(null)

    try {
      if (selectedPayment === 'balance') {
        // Pay from balance
        console.log('Paying from balance...')
        const result = await api.payFromBalance(planId)
        console.log('Balance payment result:', result)

        if (result.success) {
          webApp?.HapticFeedback.notificationOccurred('success')
          webApp?.showAlert('Оплата успешна! Ваша подписка активирована.')
          fetchUser() // Refresh balance
          navigate('/key')
        }
        return
      }

      if (selectedPayment === 'ton') {
        // Create payment
        console.log('Creating TON payment...')
        const result = await api.buySubscription(planId, 'ton')
        console.log('Payment result:', result)
        const tonInfo = result.ton_info as TONPaymentInfo
        console.log('TON info:', tonInfo)

        if (!tonInfo) {
          setError('Не удалось получить данные для оплаты TON')
          return
        }

        if (!address) {
          // Connect wallet first
          console.log('No wallet connected, opening modal...')
          await tonConnectUI.openModal()
          setPaymentInfo(tonInfo)
          return
        }

        // Send transaction
        console.log('Sending TON transaction...')
        await sendTONTransaction(tonInfo)
      } else {
        // Stars payment via Telegram
        console.log('Creating Stars payment...')
        const result = await api.buySubscription(planId, 'stars')
        console.log('Stars payment result:', result)
        const paymentId = result.payment.id

        // Request invoice link from backend
        console.log('Requesting invoice link for payment:', paymentId)
        const invoiceResponse = await api.initStarsPayment(paymentId)
        console.log('Invoice response:', invoiceResponse)

        if (invoiceResponse.invoice_link && webApp) {
          // Open Telegram payment
          console.log('Opening invoice:', invoiceResponse.invoice_link)
          webApp.openInvoice(invoiceResponse.invoice_link, (status) => {
            if (status === 'paid') {
              webApp.HapticFeedback.notificationOccurred('success')
              webApp.showAlert('Оплата успешна! Ваша подписка активирована.')
              navigate('/key')
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

  const sendTONTransaction = async (info: TONPaymentInfo) => {
    try {
      console.log('Preparing TON transaction with info:', info)
      const amountNano = (parseFloat(info.amount) * 1e9).toString()
      console.log('Amount in nanotons:', amountNano)

      // Simple transfer without payload - payment identified by amount and sender
      const transaction = {
        validUntil: Math.floor(Date.now() / 1000) + 600, // 10 minutes
        messages: [
          {
            address: info.wallet_address,
            amount: amountNano,
          },
        ],
      }

      console.log('Sending transaction:', JSON.stringify(transaction))
      const result = await tonConnectUI.sendTransaction(transaction)
      console.log('Transaction result:', result)

      // Verify payment - passing boc which contains sender address
      const verification = await api.verifyTONPayment(info.payment_id, result.boc)
      console.log('Verification result:', verification)

      if (verification.success) {
        webApp?.HapticFeedback.notificationOccurred('success')
        webApp?.showAlert('Оплата успешна! Ваша подписка активирована.')
        navigate('/key')
      }
    } catch (err) {
      console.error('TON transaction error:', err)
      setError('Транзакция отменена или не удалась: ' + (err as Error).message)
    }
  }

  // If wallet connected and we have pending payment, try to complete
  useEffect(() => {
    if (address && paymentInfo) {
      sendTONTransaction(paymentInfo)
    }
  }, [address, paymentInfo])

  return (
    <div className="p-4">
      <h1 className="text-2xl font-bold mb-2">Оплата</h1>
      <p className="text-hint mb-6">Выберите способ оплаты</p>

      {/* Plan Summary */}
      <div className="card mb-6">
        <div className="flex justify-between items-center">
          <div>
            <h3 className="font-semibold">{plan.name}</h3>
            <p className="text-sm text-hint">{plan.description}</p>
          </div>
          <div className="text-right">
            <p className="font-bold">≈{Math.round(plan.price_usd * usdRub)} ₽</p>
            <p className="text-xs text-hint">${plan.price_usd}</p>
          </div>
        </div>
      </div>

      {/* Payment Methods */}
      <div className="space-y-3 mb-6">
        {/* Balance payment option (if user has enough) */}
        {userBalance > 0 && (
          <PaymentMethodCard
            icon="💰"
            title="Баланс"
            subtitle={canPayFromBalance
              ? `${userBalance.toFixed(4)} TON (хватает!)`
              : `${userBalance.toFixed(4)} TON (нужно ${plan.price_ton} TON)`}
            selected={selectedPayment === 'balance'}
            onSelect={() => canPayFromBalance && setSelectedPayment('balance')}
            disabled={!canPayFromBalance}
          />
        )}
        <PaymentMethodCard
          icon="💎"
          title="TON"
          subtitle={`${plan.price_ton} TON ≈ ${Math.round(plan.price_ton * tonRub)} ₽`}
          selected={selectedPayment === 'ton'}
          onSelect={() => setSelectedPayment('ton')}
        />
        <PaymentMethodCard
          icon="⭐"
          title="Telegram Stars"
          subtitle={`${plan.price_stars} ⭐ ≈ ${Math.round(plan.price_stars * starRub)} ₽`}
          selected={selectedPayment === 'stars'}
          onSelect={() => setSelectedPayment('stars')}
        />
      </div>

      {/* Error */}
      {error && (
        <div className="bg-red-100 text-red-700 p-3 rounded-xl mb-4 text-sm">
          {error}
        </div>
      )}

      {/* Pay Button */}
      <button
        onClick={handlePayment}
        disabled={loading}
        className="btn-primary w-full"
      >
        {loading ? (
          <span className="flex items-center justify-center gap-2">
            <span className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></span>
            Обработка...
          </span>
        ) : selectedPayment === 'balance' ? (
          `Оплатить с баланса (${plan.price_ton} TON)`
        ) : selectedPayment === 'ton' ? (
          `Оплатить ${plan.price_ton} TON`
        ) : (
          `Оплатить ${plan.price_stars} Stars`
        )}
      </button>

      {/* TON Connect Status */}
      {selectedPayment === 'ton' && address && (
        <p className="text-center text-hint text-sm mt-4">
          Кошелек подключен: {address.slice(0, 6)}...{address.slice(-4)}
        </p>
      )}
    </div>
  )
}

interface PaymentMethodCardProps {
  icon: string
  title: string
  subtitle: string
  selected: boolean
  onSelect: () => void
  disabled?: boolean
}

function PaymentMethodCard({ icon, title, subtitle, selected, onSelect, disabled }: PaymentMethodCardProps) {
  return (
    <button
      onClick={onSelect}
      disabled={disabled}
      className={`card w-full flex items-center gap-4 transition-all ${
        selected ? 'ring-2 ring-tg-button' : ''
      } ${disabled ? 'opacity-50 cursor-not-allowed' : ''}`}
    >
      <span className="text-2xl">{icon}</span>
      <div className="flex-1 text-left">
        <p className="font-medium">{title}</p>
        <p className="text-sm text-hint">{subtitle}</p>
      </div>
      <div className={`w-5 h-5 rounded-full border-2 ${
        selected ? 'bg-tg-button border-tg-button' : 'border-gray-300'
      }`}>
        {selected && (
          <svg className="w-full h-full text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
          </svg>
        )}
      </div>
    </button>
  )
}
