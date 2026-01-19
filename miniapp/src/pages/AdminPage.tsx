import { useEffect, useState } from 'react'
import { api } from '../api/client'

type Tab = 'stats' | 'users' | 'bans' | 'promo'

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

export default function AdminPage() {
  const [tab, setTab] = useState<Tab>('stats')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [stats, setStats] = useState<Stats | null>(null)
  const [users, setUsers] = useState<UserItem[]>([])
  const [usersTotal, setUsersTotal] = useState(0)
  const [bans, setBans] = useState<BanItem[]>([])
  const [promos, setPromos] = useState<PromoItem[]>([])
  const [search, setSearch] = useState('')
  const [selectedUser, setSelectedUser] = useState<UserItem | null>(null)

  // Modal states
  const [showBalanceModal, setShowBalanceModal] = useState(false)
  const [showBanModal, setShowBanModal] = useState(false)
  const [showExtendModal, setShowExtendModal] = useState(false)
  const [showPromoModal, setShowPromoModal] = useState(false)
  const [balanceAmount, setBalanceAmount] = useState('')
  const [banReason, setBanReason] = useState('')
  const [extendDays, setExtendDays] = useState('')
  const [promoType, setPromoType] = useState<'balance' | 'days'>('balance')
  const [promoValue, setPromoValue] = useState('')
  const [promoCount, setPromoCount] = useState('1')
  const [promoMaxUses, setPromoMaxUses] = useState('1')

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
    if (!promoValue) return
    try {
      const count = parseInt(promoCount) || 1
      const maxUses = promoMaxUses ? parseInt(promoMaxUses) : undefined
      if (count > 1) {
        const result = await api.admin.createBulkPromoCodes(count, promoType, parseFloat(promoValue), maxUses)
        alert(`Created ${result.count} codes:\n${result.codes.join('\n')}`)
      } else {
        const result = await api.admin.createPromoCode(promoType, parseFloat(promoValue), maxUses)
        alert(`Created code: ${result.code}`)
      }
      setShowPromoModal(false)
      setPromoValue('')
      setPromoCount('1')
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
      <h1 className="text-2xl font-bold mb-4">Admin Panel</h1>

      {/* Tabs */}
      <div className="flex gap-2 mb-4 overflow-x-auto">
        {(['stats', 'users', 'bans', 'promo'] as Tab[]).map((t) => (
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
              <div className="flex gap-2">
                <button
                  onClick={() => setPromoType('balance')}
                  className={`flex-1 py-2 rounded-lg ${promoType === 'balance' ? 'bg-tg-button text-white' : 'bg-tg-secondary-bg'}`}
                >
                  Balance (TON)
                </button>
                <button
                  onClick={() => setPromoType('days')}
                  className={`flex-1 py-2 rounded-lg ${promoType === 'days' ? 'bg-tg-button text-white' : 'bg-tg-secondary-bg'}`}
                >
                  Days
                </button>
              </div>
              <input
                type="number"
                step="0.0001"
                value={promoValue}
                onChange={(e) => setPromoValue(e.target.value)}
                placeholder={promoType === 'balance' ? 'TON amount' : 'Days count'}
                className="input w-full"
              />
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
    </div>
  )
}
