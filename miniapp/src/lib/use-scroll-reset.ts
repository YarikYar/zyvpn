import { useLayoutEffect } from "react"

export function scrollMainToTop() {
  const el = document.querySelector("main") as HTMLElement | null
  if (!el) return
  el.scrollTop = 0
  el.scrollLeft = 0
}

export function useScrollResetOnChange(dep: unknown) {
  useLayoutEffect(() => {
    scrollMainToTop()
  }, [dep])
}
