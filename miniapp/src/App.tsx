import { useRef, useState, type ReactNode } from 'react'
import { AnimatePresence, motion } from 'motion/react'

import { MaintenanceScreen, SplashScreen } from '@/components/maintenance-screen'
import { NoInitDataScreen } from '@/components/no-init-data-screen'
import { SiteBalanceChip } from '@/components/site-balance-chip'
import { SiteBrandChip } from '@/components/site-brand-chip'
import { SiteHeader } from '@/components/site-header'
import { AdminSection } from '@/components/sections/admin'
import { MainSection } from '@/components/sections/main'
import { ReferralsSection } from '@/components/sections/referrals'
import { SettingsSection } from '@/components/sections/settings'
import {
  SubscriptionSection,
  type SubscriptionView,
} from '@/components/sections/subscription'
import { WalletSection } from '@/components/sections/wallet'
import { hasInitData } from '@/lib/api'
import { BackButtonProvider } from '@/lib/back-button'
import { LocaleProvider } from '@/lib/i18n'
import { navItems, type NavKey } from '@/lib/nav'
import { useMe } from '@/lib/queries'
import { scrollMainToTop } from '@/lib/use-scroll-reset'

const tabOrder: NavKey[] = navItems.map((n) => n.key)

const TRANSITION = {
  type: "spring",
  bounce: 0,
  duration: 0.45,
} as const

const variants = {
  enter: (d: number) => ({ opacity: 0, x: d * 28 }),
  center: { opacity: 1, x: 0 },
  exit: (d: number) => ({ opacity: 0, x: d * -28 }),
}

function App() {
  if (!hasInitData()) {
    return (
      <LocaleProvider>
        <NoInitDataScreen />
      </LocaleProvider>
    )
  }

  return (
    <LocaleProvider>
      <BackButtonProvider>
        <AuthGate>
          <AppShell />
        </AuthGate>
      </BackButtonProvider>
    </LocaleProvider>
  )
}

function AuthGate({ children }: { children: ReactNode }) {
  const meQuery = useMe()
  if (meQuery.isPending) return <SplashScreen />
  if (meQuery.isError) {
    return (
      <MaintenanceScreen
        onRetry={() => {
          void meQuery.refetch()
        }}
        retrying={meQuery.isFetching}
      />
    )
  }
  return <>{children}</>
}

function AppShell() {
  const [active, setActive] = useState<NavKey>('Main')
  const [walletOpen, setWalletOpen] = useState(false)
  const [adminOpen, setAdminOpen] = useState(false)
  const [pendingSubView, setPendingSubView] =
    useState<SubscriptionView | null>(null)
  const previousTabRef = useRef<NavKey>('Main')
  const directionRef = useRef(0)

  const handleNavChange = (next: NavKey) => {
    if (adminOpen) {
      scrollMainToTop()
      directionRef.current = -1
      setAdminOpen(false)
      if (next !== active) setActive(next)
      return
    }
    if (walletOpen) {
      scrollMainToTop()
      directionRef.current = -1
      setWalletOpen(false)
      if (next !== active) setActive(next)
      return
    }
    if (next === active) return
    scrollMainToTop()
    directionRef.current =
      tabOrder.indexOf(next) > tabOrder.indexOf(active) ? 1 : -1
    setActive(next)
  }

  const handleOpenWallet = () => {
    if (walletOpen) return
    scrollMainToTop()
    directionRef.current = 1
    setWalletOpen(true)
  }

  const handleCloseWallet = () => {
    scrollMainToTop()
    directionRef.current = -1
    setWalletOpen(false)
  }

  const handleOpenAdmin = () => {
    if (adminOpen) return
    scrollMainToTop()
    directionRef.current = 1
    setAdminOpen(true)
  }

  const handleCloseAdmin = () => {
    scrollMainToTop()
    directionRef.current = -1
    setAdminOpen(false)
  }

  const handleShowKey = () => {
    scrollMainToTop()
    previousTabRef.current = active
    setPendingSubView('success')
    if (active !== 'Subscription') {
      directionRef.current =
        tabOrder.indexOf('Subscription') > tabOrder.indexOf(active) ? 1 : -1
      setActive('Subscription')
    }
  }

  const handleSubscriptionExternalBack = () => {
    scrollMainToTop()
    const target = previousTabRef.current
    directionRef.current =
      tabOrder.indexOf(target) > tabOrder.indexOf(active) ? 1 : -1
    setActive(target)
  }

  const direction = directionRef.current

  return (
    <>
      <SiteBrandChip />
      <SiteBalanceChip onClick={handleOpenWallet} />
      <main
        className="relative overflow-x-clip overflow-y-auto pb-24 sm:pb-28"
        style={{
          height: '100dvh',
          paddingTop:
            'calc(var(--tg-safe-area-inset-top, env(safe-area-inset-top)) + 5rem)',
        }}
      >
        <AnimatePresence mode="popLayout" initial={false} custom={direction}>
          <motion.div
            key={adminOpen ? 'admin' : walletOpen ? 'wallet' : active}
            custom={direction}
            variants={variants}
            initial="enter"
            animate="center"
            exit="exit"
            transition={TRANSITION}
          >
            {adminOpen ? (
              <AdminSection onClose={handleCloseAdmin} />
            ) : walletOpen ? (
              <WalletSection onClose={handleCloseWallet} />
            ) : (
              <>
                {active === 'Main' && <MainSection onShowKey={handleShowKey} />}
                {active === 'Subscription' && (
                  <SubscriptionSection
                    requestView={pendingSubView}
                    onConsumeRequest={() => setPendingSubView(null)}
                    onExternalBack={handleSubscriptionExternalBack}
                  />
                )}
                {active === 'Referrals' && <ReferralsSection />}
                {active === 'Settings' && (
                  <SettingsSection onOpenAdmin={handleOpenAdmin} />
                )}
              </>
            )}
          </motion.div>
        </AnimatePresence>
      </main>
      <SiteHeader active={active} onChange={handleNavChange} />
    </>
  )
}

export default App
