import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api/client'

type Tab = 'stats' | 'users' | 'bans' | 'promo' | 'cash' | 'plans' | 'servers' | 'settings'

interface Stats {
  total_users: number
  active_subscriptions: number
  banned_users: number
  active_promo_codes: number
}

interface UserItem {
  id: number
  username?: string
  first_name?: string
  balance: number
  created_at: string
  subscription?: {
    status: string
    expires_at?: string
  }
}

interface BanItem {
  id: string
  user_id?: number
  ip_address?: string
  reason?: string
  banned_at: string
  expires_at?: string
}

interface PromoItem {
  id: string
  code: string
  type: string
  value: number
  max_uses?: number
  used_count: number
  is_active: boolean
  expires_at?: string
}

interface PlanItem {
  id: string
  name: string
  description: string
  duration_days: number
  traffic_gb: number
  max_devices: number
  price_ton: number
  price_stars: number
  price_usd: number
  is_active: boolean
  sort_order: number
}

interface ServerItem {
  id: string
  name: string
  country: string
  city?: string
  flag_emoji: string
  xui_base_url: string
  xui_username: string
  xui_password: string
  xui_inbound_id: number
  server_address: string
  server_port: number
  public_key: string
  short_id: string
  server_name: string
  is_active: boolean
  sort_order: number
  capacity: number
  current_load: number
  status: string
  ping_ms?: number
}

export default function AdminPage() {
  const navigate = useNavigate()
  const [tab, setTab] = useState<Tab>('stats')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [stats, setStats] = useState<Stats | null>(null)
  const [users, setUsers] = useState<UserItem[]>([])
  const [usersTotal, setUsersTotal] = useState(0)
  const [bans, setBans] = useState<BanItem[]>([])
  const [promos, setPromos] = useState<PromoItem[]>([])
  const [plans, setPlans] = useState<PlanItem[]>([])
  const [servers, setServers] = useState<ServerItem[]>([])
  const [topupBonus, setTopupBonus] = useState(0)
  const [referralBonus, setReferralBonus] = useState(5)
  const [referralBonusDays, setReferralBonusDays] = useState(0)
  const [regionSwitchPrice, setRegionSwitchPrice] = useState(0.1)
  const [search, setSearch] = useState('')
  const [selectedUser, setSelectedUser] = useState<UserItem | null>(null)
  const [selectedPlan, setSelectedPlan] = useState<PlanItem | null>(null)
  const [selectedServer, setSelectedServer] = useState<ServerItem | null>(null)

  // Modal states
  const [showBalanceModal, setShowBalanceModal] = useState(false)
  const [showBanModal, setShowBanModal] = useState(false)
  const [showExtendModal, setShowExtendModal] = useState(false)
  const [showPromoModal, setShowPromoModal] = useState(false)
  const [balanceAmount, setBalanceAmount] = useState('')
  const [banReason, setBanReason] = useState('')
  const [extendDays, setExtendDays] = useState('')
  const [promoType, setPromoType] = useState<'balance' | 'days' | 'region_switch' | 'cash_plan'>('balance')
  const [promoValue, setPromoValue] = useState('')
  const [promoCount, setPromoCount] = useState('1')
  const [promoMaxUses, setPromoMaxUses] = useState('1')
  const [promoPlanId, setPromoPlanId] = useState('')
  const [promoCashAmount, setPromoCashAmount] = useState('')

  // Pending cash payments (for the new cash provider flow)
  const [cashPayments, setCashPayments] = useState<any[]>([])

  // Plan modal states
  const [showPlanModal, setShowPlanModal] = useState(false)
  const [planName, setPlanName] = useState('')
  const [planDescription, setPlanDescription] = useState('')
  const [planDuration, setPlanDuration] = useState('')
  const [planTraffic, setPlanTraffic] = useState('')
  const [planDevices, setPlanDevices] = useState('3')
  const [planPriceTon, setPlanPriceTon] = useState('')
  const [planPriceStars, setPlanPriceStars] = useState('')
  const [planPriceUsd, setPlanPriceUsd] = useState('')
  const [planSortOrder, setPlanSortOrder] = useState('0')
  const [planIsActive, setPlanIsActive] = useState(true)

  // Server modal states
  const [showServerModal, setShowServerModal] = useState(false)
  const [serverName, setServerName] = useState('')
  const [serverCountry, setServerCountry] = useState('')
  const [serverCity, setServerCity] = useState('')
  const [serverFlagEmoji, setServerFlagEmoji] = useState('')
  const [serverXuiBaseUrl, setServerXuiBaseUrl] = useState('')
  const [serverXuiUsername, setServerXuiUsername] = useState('')
  const [serverXuiPassword, setServerXuiPassword] = useState('')
  const [serverXuiInboundId, setServerXuiInboundId] = useState('1')
  const [serverAddress, setServerAddress] = useState('')
  const [serverSortOrder, setServerSortOrder] = useState('0')
  const [serverCapacity, setServerCapacity] = useState('100')
  const [serverIsActive, setServerIsActive] = useState(true)
  const [serverSaving, setServerSaving] = useState(false)

  useEffect(() => {
    loadData()
  }, [tab])

  const loadData = async () => {
    setLoading(true)
    setError(null)
    try {
      if (tab === 'stats') {
        const data = await api.admin.getStats()
        setStats(data)
      } else if (tab === 'users') {
        const data = await api.admin.listUsers(50, 0, search)
        setUsers(data.users)
        setUsersTotal(data.total)
      } else if (tab === 'bans') {
        const data = await api.admin.listBans()
        setBans(data.bans || [])
      } else if (tab === 'promo') {
        const data = await api.admin.listPromoCodes()
        setPromos(data.promo_codes || [])
        // Plans needed for the cash_plan promo creation form.
        const plansData = await api.admin.listPlans()
        setPlans(plansData.plans || [])
      } else if (tab === 'cash') {
        const data = await api.admin.listPendingCashPayments()
        setCashPayments(data.payments || [])
      } else if (tab === 'plans') {
        const data = await api.admin.listPlans()
        setPlans(data.plans || [])
      } else if (tab === 'servers') {
        const data = await api.admin.listServers()
        setServers(data.servers || [])
      } else if (tab === 'settings') {
        const [topupData, referralData, referralDaysData, regionSwitchData] = await Promise.all([
          api.admin.getTopupBonus(),
          api.admin.getReferralBonus(),
          api.admin.getReferralBonusDays(),
          api.admin.getRegionSwitchPrice()
        ])
        setTopupBonus(topupData.topup_bonus_percent || 0)
        setReferralBonus(referralData.referral_bonus_percent || 5)
        setReferralBonusDays(referralDaysData.referral_bonus_days || 0)
        setRegionSwitchPrice(regionSwitchData.region_switch_price || 0.1)
      }
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setLoading(false)
    }
  }

  const searchUsers = async () => {
    setLoading(true)
    try {
      const data = await api.admin.listUsers(50, 0, search)
      setUsers(data.users)
      setUsersTotal(data.total)
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setLoading(false)
    }
  }

  const handleAddBalance = async () => {
    if (!selectedUser || !balanceAmount) return
    try {
      await api.admin.addBalance(selectedUser.id, parseFloat(balanceAmount))
      setShowBalanceModal(false)
      setBalanceAmount('')
      loadData()
    } catch (e) {
      alert((e as Error).message)
    }
  }

  const handleBanUser = async () => {
    if (!selectedUser) return
    try {
      await api.admin.banUser(selectedUser.id, banReason || 'Banned by admin')
      setShowBanModal(false)
      setBanReason('')
      loadData()
    } catch (e) {
      alert((e as Error).message)
    }
  }

  const handleUnbanUser = async (userId: number) => {
    try {
      await api.admin.unbanUser(userId)
      loadData()
    } catch (e) {
      alert((e as Error).message)
    }
  }

  const handleExtendSub = async () => {
    if (!selectedUser || !extendDays) return
    try {
      await api.admin.extendSubscription(selectedUser.id, parseInt(extendDays))
      setShowExtendModal(false)
      setExtendDays('')
      loadData()
    } catch (e) {
      alert((e as Error).message)
    }
  }

  const handleCancelSub = async (userId: number) => {
    if (!confirm('Cancel subscription?')) return
    try {
      await api.admin.cancelSubscription(userId)
      loadData()
    } catch (e) {
      alert((e as Error).message)
    }
  }

  const handleCreatePromo = async () => {
    const isCashPlan = promoType === 'cash_plan'
    if (!isCashPlan && !promoValue) return
    if (isCashPlan && (!promoPlanId || !promoCashAmount)) {
      alert('Для cash_plan нужен план и сумма в рублях')
      return
    }
    try {
      const count = parseInt(promoCount) || 1
      const maxUses = promoMaxUses ? parseInt(promoMaxUses) : undefined
      const value = isCashPlan ? 1 : parseFloat(promoValue)
      const planId = isCashPlan ? promoPlanId : undefined
      const cashAmount = isCashPlan ? parseFloat(promoCashAmount) : undefined
      if (count > 1) {
        const result = await api.admin.createBulkPromoCodes(count, promoType, value, maxUses, undefined, undefined, planId, cashAmount)
        alert(`Created ${result.count} codes:\n${result.codes.join('\n')}`)
      } else {
        const result = await api.admin.createPromoCode(promoType, value, maxUses, undefined, undefined, planId, cashAmount)
        alert(`Created code: ${result.code}`)
      }
      setShowPromoModal(false)
      setPromoValue('')
      setPromoCount('1')
      setPromoPlanId('')
      setPromoCashAmount('')
      loadData()
    } catch (e) {
      alert((e as Error).message)
    }
  }

  const handleApproveCash = async (paymentId: string) => {
    try {
      await api.admin.approveCashPayment(paymentId)
      loadData()
    } catch (e) {
      alert((e as Error).message)
    }
  }

  const handleRejectCash = async (paymentId: string) => {
    const reason = prompt('Причина отказа (опционально)') || ''
    try {
      await api.admin.rejectCashPayment(paymentId, reason)
      loadData()
    } catch (e) {
      alert((e as Error).message)
    }
  }

  const handleDeactivatePromo = async (code: string) => {
    if (!confirm(`Deactivate ${code}?`)) return
    try {
      await api.admin.deactivatePromoCode(code)
      loadData()
    } catch (e) {
      alert((e as Error).message)
    }
  }

  const resetPlanForm = () => {
    setSelectedPlan(null)
    setPlanName('')
    setPlanDescription('')
    setPlanDuration('')
    setPlanTraffic('')
    setPlanDevices('3')
    setPlanPriceTon('')
    setPlanPriceStars('')
    setPlanPriceUsd('')
    setPlanSortOrder('0')
    setPlanIsActive(true)
  }

  const openPlanModal = (plan?: PlanItem) => {
    if (plan) {
      setSelectedPlan(plan)
      setPlanName(plan.name)
      setPlanDescription(plan.description)
      setPlanDuration(plan.duration_days.toString())
      setPlanTraffic(plan.traffic_gb.toString())
      setPlanDevices(plan.max_devices.toString())
      setPlanPriceTon(plan.price_ton.toString())
      setPlanPriceStars(plan.price_stars.toString())
      setPlanPriceUsd(plan.price_usd.toString())
      setPlanSortOrder(plan.sort_order.toString())
      setPlanIsActive(plan.is_active)
    } else {
      resetPlanForm()
    }
    setShowPlanModal(true)
  }

  const handleSavePlan = async () => {
    if (!planName || !planDuration || !planTraffic) {
      alert('Заполните обязательные поля: название, длительность, трафик')
      return
    }
    try {
      const data = {
        name: planName,
        description: planDescription,
        duration_days: parseInt(planDuration),
        traffic_gb: parseInt(planTraffic),
        max_devices: parseInt(planDevices) || 3,
        price_ton: parseFloat(planPriceTon) || 0,
        price_stars: parseInt(planPriceStars) || 0,
        price_usd: parseFloat(planPriceUsd) || 0,
        sort_order: parseInt(planSortOrder) || 0,
        is_active: planIsActive,
      }

      if (selectedPlan) {
        await api.admin.updatePlan(selectedPlan.id, data)
      } else {
        await api.admin.createPlan(data)
      }

      setShowPlanModal(false)
      resetPlanForm()
      await loadData()
    } catch (e) {
      alert((e as Error).message)
    }
  }

  const handleDeletePlan = async (planId: string) => {
    if (!confirm('Удалить тариф?')) return
    try {
      await api.admin.deletePlan(planId)
      await loadData()
    } catch (e) {
      alert((e as Error).message)
    }
  }

  const handleTogglePlanActive = async (plan: PlanItem) => {
    try {
      await api.admin.updatePlan(plan.id, { is_active: !plan.is_active })
      await loadData()
    } catch (e) {
      alert((e as Error).message)
    }
  }

  // Server functions
  const resetServerForm = () => {
    setSelectedServer(null)
    setServerName('')
    setServerCountry('')
    setServerCity('')
    setServerFlagEmoji('')
    setServerXuiBaseUrl('')
    setServerXuiUsername('')
    setServerXuiPassword('')
    setServerXuiInboundId('1')
    setServerAddress('')
    setServerSortOrder('0')
    setServerCapacity('100')
    setServerIsActive(true)
    setServerSaving(false)
  }

  const openServerModal = (server?: ServerItem) => {
    if (server) {
      setSelectedServer(server)
      setServerName(server.name)
      setServerCountry(server.country)
      setServerCity(server.city || '')
      setServerFlagEmoji(server.flag_emoji)
      setServerXuiBaseUrl(server.xui_base_url)
      setServerXuiUsername(server.xui_username)
      setServerXuiPassword(server.xui_password)
      setServerXuiInboundId(server.xui_inbound_id.toString())
      setServerAddress(server.server_address)
      setServerSortOrder(server.sort_order.toString())
      setServerCapacity(server.capacity?.toString() || '100')
      setServerIsActive(server.is_active)
    } else {
      resetServerForm()
    }
    setShowServerModal(true)
  }

  const handleSaveServer = async () => {
    if (!serverName || !serverXuiBaseUrl || !serverAddress) {
      alert('Заполните обязательные поля: название, XUI URL, адрес сервера')
      return
    }
    setServerSaving(true)
    try {
      const data = {
        name: serverName,
        country: serverCountry,
        city: serverCity || undefined,
        flag_emoji: serverFlagEmoji,
        xui_base_url: serverXuiBaseUrl,
        xui_username: serverXuiUsername,
        xui_password: serverXuiPassword,
        xui_inbound_id: parseInt(serverXuiInboundId) || 1,
        server_address: serverAddress,
        server_port: 443, // Will be updated from XUI
        public_key: '',
        short_id: '',
        server_name: 'www.google.com',
        sort_order: parseInt(serverSortOrder) || 0,
        capacity: parseInt(serverCapacity) || 100,
        is_active: serverIsActive,
      }

      let serverId: string
      if (selectedServer) {
        await api.admin.updateServer(selectedServer.id, data)
        serverId = selectedServer.id
      } else {
        const result = await api.admin.createServer(data)
        serverId = result.id
      }

      // Auto-fetch VPN params from XUI
      try {
        const testResult = await api.admin.testServerConnection(serverId)
        if (testResult.connected) {
          await api.admin.updateServer(serverId, {
            server_port: testResult.port,
            public_key: testResult.public_key,
            short_id: testResult.short_id,
            server_name: testResult.server_name,
          })
        } else {
          alert(`Сервер создан, но не удалось подключиться к XUI: ${testResult.error}`)
        }
      } catch (e) {
        alert(`Сервер создан, но ошибка проверки: ${(e as Error).message}`)
      }

      setShowServerModal(false)
      resetServerForm()
      await loadData()
    } catch (e) {
      alert((e as Error).message)
    } finally {
      setServerSaving(false)
    }
  }

  const handleDeleteServer = async (serverId: string) => {
    if (!confirm('Удалить сервер?')) return
    try {
      await api.admin.deleteServer(serverId)
      await loadData()
    } catch (e) {
      alert((e as Error).message)
    }
  }

  const handleToggleServerActive = async (server: ServerItem) => {
    try {
      await api.admin.updateServer(server.id, { is_active: !server.is_active })
      await loadData()
    } catch (e) {
      alert((e as Error).message)
    }
  }

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return '-'
    return new Date(dateStr).toLocaleDateString('ru-RU')
  }

  if (error === 'access denied') {
    return (
      <div className="p-4 text-center">
        <h1 className="text-xl font-bold text-red-500">Access Denied</h1>
        <p className="text-hint mt-2">You are not an admin</p>
      </div>
    )
  }

  return (
    <div className="p-4">
      <div className="flex justify-between items-center mb-4">
        <h1 className="text-2xl font-bold">Admin Panel</h1>
        <button
          onClick={() => navigate('/')}
          className="bg-tg-secondary-bg px-3 py-1.5 rounded-lg text-sm"
        >
          ← Выход
        </button>
      </div>

      {/* Tabs */}
      <div className="flex gap-2 mb-4 overflow-x-auto">
        {(['stats', 'users', 'bans', 'promo', 'cash', 'plans', 'servers', 'settings'] as Tab[]).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`px-4 py-2 rounded-lg whitespace-nowrap ${
              tab === t ? 'bg-tg-button text-white' : 'bg-tg-secondary-bg'
            }`}
          >
            {t === 'stats' && 'Stats'}
            {t === 'users' && 'Users'}
            {t === 'bans' && 'Bans'}
            {t === 'promo' && 'Promo'}
            {t === 'cash' && '💵'}
            {t === 'plans' && 'Plans'}
            {t === 'servers' && '🖥️'}
            {t === 'settings' && '⚙️'}
          </button>
        ))}
      </div>

      {loading && <div className="text-center py-8">Loading...</div>}

      {error && error !== 'access denied' && (
        <div className="text-red-500 mb-4">{error}</div>
      )}

      {/* Stats Tab */}
      {tab === 'stats' && stats && !loading && (
        <div className="grid grid-cols-2 gap-3">
          <div className="card">
            <p className="text-hint text-sm">Users</p>
            <p className="text-2xl font-bold">{stats.total_users}</p>
          </div>
          <div className="card">
            <p className="text-hint text-sm">Active Subs</p>
            <p className="text-2xl font-bold">{stats.active_subscriptions}</p>
          </div>
          <div className="card">
            <p className="text-hint text-sm">Banned</p>
            <p className="text-2xl font-bold">{stats.banned_users}</p>
          </div>
          <div className="card">
            <p className="text-hint text-sm">Promo Codes</p>
            <p className="text-2xl font-bold">{stats.active_promo_codes}</p>
          </div>
        </div>
      )}

      {/* Users Tab */}
      {tab === 'users' && !loading && (
        <div>
          <div className="flex gap-2 mb-4">
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search by ID or username..."
              className="input flex-1"
              onKeyDown={(e) => e.key === 'Enter' && searchUsers()}
            />
            <button onClick={searchUsers} className="btn-primary px-4">
              Search
            </button>
          </div>
          <p className="text-hint text-sm mb-2">Total: {usersTotal}</p>
          <div className="space-y-2">
            {users.map((user) => (
              <div key={user.id} className="card">
                <div className="flex justify-between items-start">
                  <div>
                    <p className="font-semibold">
                      {user.first_name || user.username || `User ${user.id}`}
                    </p>
                    <p className="text-hint text-sm">ID: {user.id}</p>
                    <p className="text-hint text-sm">Balance: {user.balance?.toFixed(4)} TON</p>
                    {user.subscription && (
                      <p className="text-sm text-green-500">
                        Sub: {user.subscription.status} (until {formatDate(user.subscription.expires_at)})
                      </p>
                    )}
                  </div>
                  <div className="flex flex-col gap-1">
                    <button
                      onClick={() => {
                        setSelectedUser(user)
                        setShowBalanceModal(true)
                      }}
                      className="text-xs bg-tg-secondary-bg px-2 py-1 rounded"
                    >
                      + Balance
                    </button>
                    {user.subscription && (
                      <>
                        <button
                          onClick={() => {
                            setSelectedUser(user)
                            setShowExtendModal(true)
                          }}
                          className="text-xs bg-tg-secondary-bg px-2 py-1 rounded"
                        >
                          Extend
                        </button>
                        <button
                          onClick={() => handleCancelSub(user.id)}
                          className="text-xs bg-red-500 text-white px-2 py-1 rounded"
                        >
                          Cancel
                        </button>
                      </>
                    )}
                    <button
                      onClick={() => {
                        setSelectedUser(user)
                        setShowBanModal(true)
                      }}
                      className="text-xs bg-red-500 text-white px-2 py-1 rounded"
                    >
                      Ban
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Bans Tab */}
      {tab === 'bans' && !loading && (
        <div className="space-y-2">
          {bans.length === 0 && <p className="text-hint">No active bans</p>}
          {bans.map((ban) => (
            <div key={ban.id} className="card">
              <div className="flex justify-between items-start">
                <div>
                  {ban.user_id && <p className="font-semibold">User ID: {ban.user_id}</p>}
                  {ban.ip_address && <p className="font-semibold">IP: {ban.ip_address}</p>}
                  <p className="text-hint text-sm">Reason: {ban.reason || '-'}</p>
                  <p className="text-hint text-sm">Banned: {formatDate(ban.banned_at)}</p>
                  {ban.expires_at && <p className="text-hint text-sm">Expires: {formatDate(ban.expires_at)}</p>}
                </div>
                <button
                  onClick={() => ban.user_id && handleUnbanUser(ban.user_id)}
                  className="text-xs bg-green-500 text-white px-2 py-1 rounded"
                >
                  Unban
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Promo Tab */}
      {tab === 'promo' && !loading && (
        <div>
          <button
            onClick={() => setShowPromoModal(true)}
            className="btn-primary w-full mb-4"
          >
            + Create Promo Code
          </button>
          <div className="space-y-2">
            {promos.map((promo) => {
              const promoText = promo.type === 'balance'
                ? `${promo.value} TON на баланс`
                : `${promo.value} дней подписки`

              const shareText = `🎁 Промокод ZyVPN!\n\n▶️ Код: \`${promo.code}\`\n💎 Бонус: ${promoText}\n\n👉 Активируй: @zyvpn_bot → Баланс → Промокод`

              const handleCopy = async () => {
                try {
                  await navigator.clipboard.writeText(promo.code)
                  const tg = window.Telegram?.WebApp
                  if (tg?.showPopup) {
                    tg.showPopup({ message: 'Скопировано!' })
                  }
                } catch {
                  const input = document.createElement('input')
                  input.value = promo.code
                  document.body.appendChild(input)
                  input.select()
                  document.execCommand('copy')
                  document.body.removeChild(input)
                }
              }

              const handleShare = () => {
                const tg = window.Telegram?.WebApp
                if (tg?.openTelegramLink) {
                  const encodedText = encodeURIComponent(shareText)
                  tg.openTelegramLink(`https://t.me/share/url?url=&text=${encodedText}`)
                } else if (navigator.share) {
                  navigator.share({ text: shareText })
                } else {
                  navigator.clipboard.writeText(shareText)
                }
              }

              return (
                <div key={promo.id} className="card">
                  <div className="flex justify-between items-start gap-2">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <button
                          onClick={handleCopy}
                          className="font-mono font-bold truncate text-left active:opacity-50"
                        >
                          {promo.code}
                        </button>
                        <button
                          onClick={handleShare}
                          className="text-tg-link text-lg shrink-0 active:opacity-50"
                          title="Share"
                        >
                          📤
                        </button>
                      </div>
                      <p className="text-hint text-sm">{promoText}</p>
                      <p className="text-hint text-sm">
                        Used: {promo.used_count}{promo.max_uses ? `/${promo.max_uses}` : ''}
                      </p>
                      {!promo.is_active && <p className="text-red-500 text-sm">Inactive</p>}
                    </div>
                    {promo.is_active && (
                      <button
                        onClick={() => handleDeactivatePromo(promo.code)}
                        className="text-xs bg-red-500 text-white px-2 py-1 rounded shrink-0"
                      >
                        ✕
                      </button>
                    )}
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      )}

      {/* Cash payments Tab */}
      {tab === 'cash' && !loading && (
        <div>
          <p className="text-hint text-sm mb-3">
            Ожидают подтверждения админом. Подтверди — подписка активируется.
          </p>
          <div className="space-y-2">
            {cashPayments.length === 0 && <p className="text-hint">Нет ожидающих платежей</p>}
            {cashPayments.map((p: any) => {
              const planName = plans.find(pl => pl.id === p.plan_id)?.name || p.plan_id
              return (
                <div key={p.id} className="card">
                  <div className="flex justify-between items-start">
                    <div>
                      <p className="font-semibold">{planName}</p>
                      <p className="text-sm text-hint">user_id: {p.user_id}</p>
                      <p className="text-sm">{p.amount} ₽</p>
                      <p className="text-xs text-hint">
                        {new Date(p.created_at).toLocaleString('ru-RU')}
                      </p>
                    </div>
                    <div className="flex flex-col gap-2 ml-2">
                      <button
                        onClick={() => handleApproveCash(p.id)}
                        className="btn-primary text-sm"
                      >
                        ✓ Получил
                      </button>
                      <button
                        onClick={() => handleRejectCash(p.id)}
                        className="bg-red-500 text-white text-sm py-2 px-4 rounded-lg"
                      >
                        Отклонить
                      </button>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      )}

      {/* Plans Tab */}
      {tab === 'plans' && !loading && (
        <div>
          <button
            onClick={() => openPlanModal()}
            className="btn-primary w-full mb-4"
          >
            + Создать тариф
          </button>
          <div className="space-y-2">
            {plans.length === 0 && <p className="text-hint">Нет тарифов</p>}
            {plans.map((plan) => (
              <div key={plan.id} className={`card ${!plan.is_active ? 'opacity-50' : ''}`}>
                <div className="flex justify-between items-start">
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <p className="font-semibold">{plan.name}</p>
                      {!plan.is_active && <span className="text-xs bg-red-500 text-white px-2 py-0.5 rounded">OFF</span>}
                    </div>
                    <p className="text-hint text-sm">{plan.description}</p>
                    <div className="text-hint text-sm mt-1">
                      <p>{plan.duration_days} дней • {plan.traffic_gb} GB • {plan.max_devices} устр.</p>
                      <p className="font-medium text-tg-text">
                        {plan.price_ton} TON / {plan.price_stars} ⭐ / ${plan.price_usd}
                      </p>
                    </div>
                  </div>
                  <div className="flex flex-col gap-1">
                    <button
                      onClick={() => openPlanModal(plan)}
                      className="text-xs bg-tg-secondary-bg px-2 py-1 rounded"
                    >
                      Edit
                    </button>
                    <button
                      onClick={() => handleTogglePlanActive(plan)}
                      className={`text-xs px-2 py-1 rounded ${plan.is_active ? 'bg-yellow-500' : 'bg-green-500'} text-white`}
                    >
                      {plan.is_active ? 'OFF' : 'ON'}
                    </button>
                    <button
                      onClick={() => handleDeletePlan(plan.id)}
                      className="text-xs bg-red-500 text-white px-2 py-1 rounded"
                    >
                      Delete
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Servers Tab */}
      {tab === 'servers' && !loading && (
        <div>
          <button
            onClick={() => openServerModal()}
            className="btn-primary w-full mb-4"
          >
            + Добавить сервер
          </button>
          <div className="space-y-2">
            {servers.length === 0 && <p className="text-hint">Нет серверов</p>}
            {servers.map((server) => (
              <div key={server.id} className={`card ${!server.is_active ? 'opacity-50' : ''}`}>
                <div className="flex justify-between items-start">
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <span className="text-lg">{server.flag_emoji}</span>
                      <p className="font-semibold">{server.name}</p>
                      {!server.is_active && <span className="text-xs bg-red-500 text-white px-2 py-0.5 rounded">OFF</span>}
                      <span className={`text-xs px-2 py-0.5 rounded ${
                        server.status === 'online' ? 'bg-green-500 text-white' :
                        server.status === 'offline' ? 'bg-red-500 text-white' :
                        'bg-gray-400 text-white'
                      }`}>
                        {server.status || 'unknown'}
                      </span>
                    </div>
                    <p className="text-hint text-sm">{server.country}{server.city ? `, ${server.city}` : ''}</p>
                    <p className="text-hint text-xs mt-1">
                      {server.server_address}:{server.server_port}
                    </p>
                    <p className="text-xs mt-1">
                      <span className={`font-medium ${
                        server.current_load / server.capacity * 100 > 80 ? 'text-red-500' :
                        server.current_load / server.capacity * 100 > 50 ? 'text-yellow-500' :
                        'text-green-500'
                      }`}>
                        {server.current_load}/{server.capacity}
                      </span>
                      <span className="text-hint"> клиентов</span>
                      {server.ping_ms && <span className="text-hint ml-2">• {server.ping_ms}ms</span>}
                    </p>
                  </div>
                  <div className="flex flex-col gap-1">
                    <button
                      onClick={() => openServerModal(server)}
                      className="text-xs bg-tg-secondary-bg px-2 py-1 rounded"
                    >
                      Edit
                    </button>
                    <button
                      onClick={() => handleToggleServerActive(server)}
                      className={`text-xs px-2 py-1 rounded ${server.is_active ? 'bg-yellow-500' : 'bg-green-500'} text-white`}
                    >
                      {server.is_active ? 'OFF' : 'ON'}
                    </button>
                    <button
                      onClick={() => handleDeleteServer(server.id)}
                      className="text-xs bg-red-500 text-white px-2 py-1 rounded"
                    >
                      Delete
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Settings Tab */}
      {tab === 'settings' && !loading && (
        <div className="space-y-6">
          {/* Topup Bonus */}
          <div className="card">
            <h3 className="font-semibold mb-2">Бонус при пополнении TON</h3>
            <p className="text-hint text-sm mb-4">
              Процент бонуса при пополнении баланса через TON
            </p>
            <div className="flex items-center gap-4">
              <input
                type="range"
                min="0"
                max="10"
                step="0.1"
                value={topupBonus}
                onChange={(e) => setTopupBonus(parseFloat(e.target.value))}
                className="flex-1 h-2 bg-tg-secondary-bg rounded-lg appearance-none cursor-pointer"
              />
              <span className="font-bold w-16 text-right">{topupBonus.toFixed(1)}%</span>
            </div>
            <button
              onClick={async () => {
                try {
                  await api.admin.setTopupBonus(topupBonus)
                  alert('Сохранено!')
                } catch (e) {
                  alert((e as Error).message)
                }
              }}
              className="btn-primary w-full mt-4"
            >
              Сохранить
            </button>
          </div>

          {/* Referral Bonus % */}
          <div className="card">
            <h3 className="font-semibold mb-2">Реферальный бонус %</h3>
            <p className="text-hint text-sm mb-4">
              Процент от каждого платежа приглашённого пользователя
            </p>
            <div className="flex items-center gap-4">
              <input
                type="range"
                min="0"
                max="20"
                step="0.5"
                value={referralBonus}
                onChange={(e) => setReferralBonus(parseFloat(e.target.value))}
                className="flex-1 h-2 bg-tg-secondary-bg rounded-lg appearance-none cursor-pointer"
              />
              <span className="font-bold w-16 text-right">{referralBonus.toFixed(1)}%</span>
            </div>
            <button
              onClick={async () => {
                try {
                  await api.admin.setReferralBonus(referralBonus)
                  alert('Сохранено!')
                } catch (e) {
                  alert((e as Error).message)
                }
              }}
              className="btn-primary w-full mt-4"
            >
              Сохранить
            </button>
          </div>

          {/* Referral Bonus Days */}
          <div className="card">
            <h3 className="font-semibold mb-2">Бонусные дни рефереру</h3>
            <p className="text-hint text-sm mb-4">
              Дни подписки рефереру при первой оплате приглашённого
            </p>
            <div className="flex items-center gap-4">
              <input
                type="range"
                min="0"
                max="30"
                step="1"
                value={referralBonusDays}
                onChange={(e) => setReferralBonusDays(parseInt(e.target.value))}
                className="flex-1 h-2 bg-tg-secondary-bg rounded-lg appearance-none cursor-pointer"
              />
              <span className="font-bold w-20 text-right">{referralBonusDays} дн.</span>
            </div>
            <button
              onClick={async () => {
                try {
                  await api.admin.setReferralBonusDays(referralBonusDays)
                  alert('Сохранено!')
                } catch (e) {
                  alert((e as Error).message)
                }
              }}
              className="btn-primary w-full mt-4"
            >
              Сохранить
            </button>
          </div>

          {/* Region Switch Price */}
          <div className="card">
            <h3 className="font-semibold mb-2">Стоимость смены региона</h3>
            <p className="text-hint text-sm mb-4">
              Цена в TON за смену региона (сервера) VPN
            </p>
            <div className="flex items-center gap-4">
              <input
                type="range"
                min="0"
                max="1"
                step="0.01"
                value={regionSwitchPrice}
                onChange={(e) => setRegionSwitchPrice(parseFloat(e.target.value))}
                className="flex-1 h-2 bg-tg-secondary-bg rounded-lg appearance-none cursor-pointer"
              />
              <span className="font-bold w-20 text-right">{regionSwitchPrice.toFixed(2)} TON</span>
            </div>
            <button
              onClick={async () => {
                try {
                  await api.admin.setRegionSwitchPrice(regionSwitchPrice)
                  alert('Сохранено!')
                } catch (e) {
                  alert((e as Error).message)
                }
              }}
              className="btn-primary w-full mt-4"
            >
              Сохранить
            </button>
          </div>
        </div>
      )}

      {/* Balance Modal */}
      {showBalanceModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
          <div className="bg-tg-bg rounded-xl p-4 w-full max-w-sm">
            <h3 className="font-bold mb-4">Add Balance</h3>
            <p className="text-hint text-sm mb-2">User: {selectedUser?.first_name || selectedUser?.id}</p>
            <input
              type="number"
              step="0.0001"
              value={balanceAmount}
              onChange={(e) => setBalanceAmount(e.target.value)}
              placeholder="Amount in TON"
              className="input w-full mb-4"
            />
            <div className="flex gap-2">
              <button onClick={() => setShowBalanceModal(false)} className="flex-1 bg-tg-secondary-bg py-2 rounded-lg">
                Cancel
              </button>
              <button onClick={handleAddBalance} className="flex-1 btn-primary">
                Add
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Ban Modal */}
      {showBanModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
          <div className="bg-tg-bg rounded-xl p-4 w-full max-w-sm">
            <h3 className="font-bold mb-4">Ban User</h3>
            <p className="text-hint text-sm mb-2">User: {selectedUser?.first_name || selectedUser?.id}</p>
            <input
              type="text"
              value={banReason}
              onChange={(e) => setBanReason(e.target.value)}
              placeholder="Reason (optional)"
              className="input w-full mb-4"
            />
            <div className="flex gap-2">
              <button onClick={() => setShowBanModal(false)} className="flex-1 bg-tg-secondary-bg py-2 rounded-lg">
                Cancel
              </button>
              <button onClick={handleBanUser} className="flex-1 bg-red-500 text-white py-2 rounded-lg">
                Ban
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Extend Modal */}
      {showExtendModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
          <div className="bg-tg-bg rounded-xl p-4 w-full max-w-sm">
            <h3 className="font-bold mb-4">Extend Subscription</h3>
            <p className="text-hint text-sm mb-2">User: {selectedUser?.first_name || selectedUser?.id}</p>
            <input
              type="number"
              value={extendDays}
              onChange={(e) => setExtendDays(e.target.value)}
              placeholder="Days to add"
              className="input w-full mb-4"
            />
            <div className="flex gap-2">
              <button onClick={() => setShowExtendModal(false)} className="flex-1 bg-tg-secondary-bg py-2 rounded-lg">
                Cancel
              </button>
              <button onClick={handleExtendSub} className="flex-1 btn-primary">
                Extend
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Promo Modal */}
      {showPromoModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
          <div className="bg-tg-bg rounded-xl p-4 w-full max-w-sm">
            <h3 className="font-bold mb-4">Create Promo Code</h3>
            <div className="space-y-3">
              <div className="flex gap-2 flex-wrap">
                <button
                  onClick={() => setPromoType('balance')}
                  className={`flex-1 py-2 rounded-lg text-sm ${promoType === 'balance' ? 'bg-tg-button text-white' : 'bg-tg-secondary-bg'}`}
                >
                  Баланс (TON)
                </button>
                <button
                  onClick={() => setPromoType('days')}
                  className={`flex-1 py-2 rounded-lg text-sm ${promoType === 'days' ? 'bg-tg-button text-white' : 'bg-tg-secondary-bg'}`}
                >
                  Дни
                </button>
                <button
                  onClick={() => setPromoType('region_switch')}
                  className={`flex-1 py-2 rounded-lg text-sm ${promoType === 'region_switch' ? 'bg-tg-button text-white' : 'bg-tg-secondary-bg'}`}
                >
                  Регион
                </button>
                <button
                  onClick={() => setPromoType('cash_plan')}
                  className={`flex-1 py-2 rounded-lg text-sm ${promoType === 'cash_plan' ? 'bg-tg-button text-white' : 'bg-tg-secondary-bg'}`}
                >
                  💵 План за нал
                </button>
              </div>
              {promoType === 'cash_plan' ? (
                <>
                  <select
                    value={promoPlanId}
                    onChange={(e) => setPromoPlanId(e.target.value)}
                    className="input w-full"
                  >
                    <option value="">Выберите тариф</option>
                    {plans.map((p: any) => (
                      <option key={p.id} value={p.id}>
                        {p.name} ({p.duration_days} дн / {p.traffic_gb || '∞'} ГБ)
                      </option>
                    ))}
                  </select>
                  <input
                    type="number"
                    step="1"
                    value={promoCashAmount}
                    onChange={(e) => setPromoCashAmount(e.target.value)}
                    placeholder="Сумма налом, ₽"
                    className="input w-full"
                  />
                </>
              ) : (
                <input
                  type="number"
                  step="0.0001"
                  value={promoValue}
                  onChange={(e) => setPromoValue(e.target.value)}
                  placeholder={promoType === 'balance' ? 'Сумма TON' : promoType === 'days' ? 'Кол-во дней' : 'Кол-во смен региона'}
                  className="input w-full"
                />
              )}
              <input
                type="number"
                value={promoCount}
                onChange={(e) => setPromoCount(e.target.value)}
                placeholder="Count (1 for single)"
                className="input w-full"
              />
              <input
                type="number"
                value={promoMaxUses}
                onChange={(e) => setPromoMaxUses(e.target.value)}
                placeholder="Max uses per code"
                className="input w-full"
              />
            </div>
            <div className="flex gap-2 mt-4">
              <button onClick={() => setShowPromoModal(false)} className="flex-1 bg-tg-secondary-bg py-2 rounded-lg">
                Cancel
              </button>
              <button onClick={handleCreatePromo} className="flex-1 btn-primary">
                Create
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Plan Modal */}
      {showPlanModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
          <div className="bg-tg-bg rounded-xl p-4 w-full max-w-sm max-h-[90vh] overflow-y-auto">
            <h3 className="font-bold mb-4">{selectedPlan ? 'Редактировать тариф' : 'Создать тариф'}</h3>
            <div className="space-y-3">
              <input
                type="text"
                value={planName}
                onChange={(e) => setPlanName(e.target.value)}
                placeholder="Название *"
                className="input w-full"
              />
              <input
                type="text"
                value={planDescription}
                onChange={(e) => setPlanDescription(e.target.value)}
                placeholder="Описание"
                className="input w-full"
              />
              <div className="grid grid-cols-3 gap-2">
                <div>
                  <label className="text-hint text-xs">Дней *</label>
                  <input
                    type="number"
                    value={planDuration}
                    onChange={(e) => setPlanDuration(e.target.value)}
                    placeholder="30"
                    className="input w-full"
                  />
                </div>
                <div>
                  <label className="text-hint text-xs">GB *</label>
                  <input
                    type="number"
                    value={planTraffic}
                    onChange={(e) => setPlanTraffic(e.target.value)}
                    placeholder="100"
                    className="input w-full"
                  />
                </div>
                <div>
                  <label className="text-hint text-xs">Устр.</label>
                  <input
                    type="number"
                    value={planDevices}
                    onChange={(e) => setPlanDevices(e.target.value)}
                    placeholder="3"
                    className="input w-full"
                  />
                </div>
              </div>
              <div className="grid grid-cols-3 gap-2">
                <div>
                  <label className="text-hint text-xs">TON</label>
                  <input
                    type="number"
                    step="0.01"
                    value={planPriceTon}
                    onChange={(e) => setPlanPriceTon(e.target.value)}
                    placeholder="1.5"
                    className="input w-full"
                  />
                </div>
                <div>
                  <label className="text-hint text-xs">Stars</label>
                  <input
                    type="number"
                    value={planPriceStars}
                    onChange={(e) => setPlanPriceStars(e.target.value)}
                    placeholder="100"
                    className="input w-full"
                  />
                </div>
                <div>
                  <label className="text-hint text-xs">USD</label>
                  <input
                    type="number"
                    step="0.01"
                    value={planPriceUsd}
                    onChange={(e) => setPlanPriceUsd(e.target.value)}
                    placeholder="3.99"
                    className="input w-full"
                  />
                </div>
              </div>
              <div className="flex items-center gap-4">
                <div className="flex-1">
                  <label className="text-hint text-xs">Порядок</label>
                  <input
                    type="number"
                    value={planSortOrder}
                    onChange={(e) => setPlanSortOrder(e.target.value)}
                    placeholder="0"
                    className="input w-full"
                  />
                </div>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={planIsActive}
                    onChange={(e) => setPlanIsActive(e.target.checked)}
                    className="w-5 h-5"
                  />
                  <span className="text-sm">Активен</span>
                </label>
              </div>
            </div>
            <div className="flex gap-2 mt-4">
              <button
                onClick={() => {
                  setShowPlanModal(false)
                  resetPlanForm()
                }}
                className="flex-1 bg-tg-secondary-bg py-2 rounded-lg"
              >
                Отмена
              </button>
              <button onClick={handleSavePlan} className="flex-1 btn-primary">
                {selectedPlan ? 'Сохранить' : 'Создать'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Server Modal */}
      {showServerModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
          <div className="bg-tg-bg rounded-xl p-4 w-full max-w-sm max-h-[90vh] overflow-y-auto">
            <h3 className="font-bold mb-4">{selectedServer ? 'Редактировать сервер' : 'Добавить сервер'}</h3>
            <div className="space-y-3">
              {/* Basic info */}
              <div className="grid grid-cols-3 gap-2">
                <div className="col-span-2">
                  <label className="text-hint text-xs">Название *</label>
                  <input
                    type="text"
                    value={serverName}
                    onChange={(e) => setServerName(e.target.value)}
                    placeholder="Нидерланды"
                    className="input w-full"
                  />
                </div>
                <div>
                  <label className="text-hint text-xs">Флаг</label>
                  <input
                    type="text"
                    value={serverFlagEmoji}
                    onChange={(e) => setServerFlagEmoji(e.target.value)}
                    placeholder="🇳🇱"
                    className="input w-full"
                  />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-2">
                <div>
                  <label className="text-hint text-xs">Страна</label>
                  <input
                    type="text"
                    value={serverCountry}
                    onChange={(e) => setServerCountry(e.target.value)}
                    placeholder="Netherlands"
                    className="input w-full"
                  />
                </div>
                <div>
                  <label className="text-hint text-xs">Город</label>
                  <input
                    type="text"
                    value={serverCity}
                    onChange={(e) => setServerCity(e.target.value)}
                    placeholder="Amsterdam"
                    className="input w-full"
                  />
                </div>
              </div>

              {/* Server address */}
              <div>
                <label className="text-hint text-xs">IP/Домен VPN сервера *</label>
                <input
                  type="text"
                  value={serverAddress}
                  onChange={(e) => setServerAddress(e.target.value)}
                  placeholder="185.123.45.67 или vpn.example.com"
                  className="input w-full"
                />
              </div>

              {/* XUI Panel */}
              <div className="border-t border-tg-secondary-bg pt-3">
                <p className="text-hint text-xs mb-2">3X-UI Panel</p>
                <input
                  type="text"
                  value={serverXuiBaseUrl}
                  onChange={(e) => setServerXuiBaseUrl(e.target.value)}
                  placeholder="https://panel.example.com:2053 *"
                  className="input w-full mb-2"
                />
                <div className="grid grid-cols-2 gap-2">
                  <input
                    type="text"
                    value={serverXuiUsername}
                    onChange={(e) => setServerXuiUsername(e.target.value)}
                    placeholder="admin"
                    className="input w-full"
                  />
                  <input
                    type="password"
                    value={serverXuiPassword}
                    onChange={(e) => setServerXuiPassword(e.target.value)}
                    placeholder="password"
                    className="input w-full"
                  />
                </div>
                <div className="mt-2">
                  <label className="text-hint text-xs">Inbound ID</label>
                  <input
                    type="number"
                    value={serverXuiInboundId}
                    onChange={(e) => setServerXuiInboundId(e.target.value)}
                    placeholder="1"
                    className="input w-full"
                  />
                </div>
                <p className="text-hint text-xs mt-2">
                  Порт, ключи и SNI подтянутся автоматически
                </p>
              </div>

              {/* Capacity and sort order */}
              <div className="grid grid-cols-2 gap-2 pt-2">
                <div>
                  <label className="text-hint text-xs">Capacity (макс. клиентов)</label>
                  <input
                    type="number"
                    value={serverCapacity}
                    onChange={(e) => setServerCapacity(e.target.value)}
                    placeholder="100"
                    className="input w-full"
                  />
                </div>
                <div>
                  <label className="text-hint text-xs">Порядок</label>
                  <input
                    type="number"
                    value={serverSortOrder}
                    onChange={(e) => setServerSortOrder(e.target.value)}
                    placeholder="0"
                    className="input w-full"
                  />
                </div>
              </div>
              <label className="flex items-center gap-2 cursor-pointer pt-2">
                <input
                  type="checkbox"
                  checked={serverIsActive}
                  onChange={(e) => setServerIsActive(e.target.checked)}
                  className="w-5 h-5"
                />
                <span className="text-sm">Активен</span>
              </label>
            </div>
            <div className="flex gap-2 mt-4">
              <button
                onClick={() => {
                  setShowServerModal(false)
                  resetServerForm()
                }}
                className="flex-1 bg-tg-secondary-bg py-2 rounded-lg"
                disabled={serverSaving}
              >
                Отмена
              </button>
              <button
                onClick={handleSaveServer}
                className="flex-1 btn-primary"
                disabled={serverSaving}
              >
                {serverSaving ? 'Сохранение...' : (selectedServer ? 'Сохранить' : 'Создать')}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
