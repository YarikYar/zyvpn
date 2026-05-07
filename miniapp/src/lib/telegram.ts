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
