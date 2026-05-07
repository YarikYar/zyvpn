import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClientProvider } from '@tanstack/react-query'
import { TonConnectUIProvider } from '@tonconnect/ui-react'

import './index.css'
import App from './App.tsx'
import { initTelegram } from '@/lib/telegram'
import { queryClient } from '@/lib/queries'

initTelegram()

const TONCONNECT_MANIFEST_URL =
  import.meta.env.VITE_TONCONNECT_MANIFEST_URL ??
  `${window.location.origin}/tonconnect-manifest.json`

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <TonConnectUIProvider manifestUrl={TONCONNECT_MANIFEST_URL}>
      <QueryClientProvider client={queryClient}>
        <App />
      </QueryClientProvider>
    </TonConnectUIProvider>
  </StrictMode>,
)
