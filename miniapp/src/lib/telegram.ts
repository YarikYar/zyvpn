import {
  init,
  viewport,
  themeParams,
} from "@telegram-apps/sdk"

const SAFE_AREA_KEYS = ["top", "bottom", "left", "right"] as const

function setSafeAreaVar(key: (typeof SAFE_AREA_KEYS)[number], value: number) {
  document.documentElement.style.setProperty(
    `--tg-safe-area-inset-${key}`,
    `${value}px`,
  )
}

function clearSafeAreaVars() {
  for (const key of SAFE_AREA_KEYS) {
    document.documentElement.style.removeProperty(`--tg-safe-area-inset-${key}`)
  }
}

export async function initTelegram() {
  try {
    init()
  } catch (err) {
    console.warn("[telegram] init skipped:", err)
    clearSafeAreaVars()
    return
  }

  try {
    if (viewport.mount.isAvailable()) {
      await viewport.mount()
      if (viewport.expand.isAvailable()) viewport.expand()
      if (viewport.bindCssVars.isAvailable()) viewport.bindCssVars()

      const apply = () => {
        const insets = viewport.safeAreaInsets()
        for (const key of SAFE_AREA_KEYS) setSafeAreaVar(key, insets[key])
      }
      apply()
      viewport.safeAreaInsets.sub(apply)
    }
  } catch (err) {
    console.warn("[telegram] viewport mount failed:", err)
  }

  try {
    if (themeParams.mount.isAvailable()) {
      themeParams.mount()
      if (themeParams.bindCssVars.isAvailable()) themeParams.bindCssVars()
    }
  } catch (err) {
    console.warn("[telegram] themeParams mount failed:", err)
  }
}

type HapticStyle = "light" | "medium" | "heavy" | "rigid" | "soft"
type HapticNotification = "success" | "warning" | "error"

type TelegramWebApp = {
  openLink?: (url: string, options?: { try_instant_view?: boolean }) => void
  openTelegramLink?: (url: string) => void
  openInvoice?: (url: string, callback: InvoiceCallback) => void
  HapticFeedback?: {
    impactOccurred?: (style: HapticStyle) => void
    notificationOccurred?: (kind: HapticNotification) => void
    selectionChanged?: () => void
  }
}

function getTelegramWebApp(): TelegramWebApp | undefined {
  return (window as { Telegram?: { WebApp?: TelegramWebApp } }).Telegram?.WebApp
}

export type InvoiceCallback = (
  status: "paid" | "failed" | "cancelled" | "pending",
) => void

export function openExternalLink(url: string) {
  const tg = getTelegramWebApp()
  if (tg?.openLink) {
    tg.openLink(url)
    return
  }
  window.open(url, "_blank", "noopener,noreferrer")
}

export function openInvoice(url: string, callback: InvoiceCallback) {
  const tg = getTelegramWebApp()
  if (tg?.openInvoice) {
    tg.openInvoice(url, callback)
    return
  }
  openExternalLink(url)
}

export async function copyToClipboard(text: string): Promise<boolean> {
  if (!text) return false
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      /* fall through to legacy path */
    }
  }
  try {
    const textarea = document.createElement("textarea")
    textarea.value = text
    textarea.setAttribute("readonly", "")
    textarea.style.position = "fixed"
    textarea.style.opacity = "0"
    textarea.style.pointerEvents = "none"
    document.body.appendChild(textarea)
    textarea.select()
    const ok = document.execCommand("copy")
    document.body.removeChild(textarea)
    return ok
  } catch {
    return false
  }
}

export function shareLink(url: string, text?: string) {
  const tg = getTelegramWebApp()
  if (tg?.openTelegramLink) {
    const params = new URLSearchParams({ url, text: text ?? "" })
    tg.openTelegramLink(`https://t.me/share/url?${params.toString()}`)
    return
  }
  const nav = navigator as Navigator & {
    share?: (data: { url?: string; text?: string; title?: string }) => Promise<void>
  }
  if (nav.share) {
    nav.share({ url, text }).catch(() => {
      openExternalLink(url)
    })
    return
  }
  openExternalLink(url)
}

export function triggerHaptic(
  kind:
    | { type: "impact"; style?: HapticStyle }
    | { type: "notification"; status: HapticNotification }
    | { type: "selection" } = { type: "selection" },
) {
  const hf = getTelegramWebApp()?.HapticFeedback
  if (!hf) return
  try {
    if (kind.type === "impact") {
      hf.impactOccurred?.(kind.style ?? "light")
      return
    }
    if (kind.type === "notification") {
      hf.notificationOccurred?.(kind.status)
      return
    }
    hf.selectionChanged?.()
  } catch {
    /* noop */
  }
}
