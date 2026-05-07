import { useCallback, useEffect, useState } from "react"

export type Theme = "light" | "dark"

function readDom(): Theme {
  if (typeof document === "undefined") return "light"
  return document.documentElement.classList.contains("dark") ? "dark" : "light"
}

export function useTheme() {
  const [theme, setThemeState] = useState<Theme>(() => readDom())

  useEffect(() => {
    const observer = new MutationObserver(() => {
      setThemeState(readDom())
    })
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ["class"],
    })
    return () => observer.disconnect()
  }, [])

  const setTheme = useCallback((next: Theme) => {
    document.documentElement.classList.toggle("dark", next === "dark")
    try {
      window.localStorage.setItem("theme", next)
    } catch {
      /* noop */
    }
  }, [])

  return { theme, setTheme }
}
