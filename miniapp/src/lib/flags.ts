import type { ComponentType } from "react"

import DE from "country-flag-icons/react/3x2/DE"
import FI from "country-flag-icons/react/3x2/FI"
import FR from "country-flag-icons/react/3x2/FR"
import GB from "country-flag-icons/react/3x2/GB"
import HK from "country-flag-icons/react/3x2/HK"
import JP from "country-flag-icons/react/3x2/JP"
import NL from "country-flag-icons/react/3x2/NL"
import SE from "country-flag-icons/react/3x2/SE"
import SG from "country-flag-icons/react/3x2/SG"
import US from "country-flag-icons/react/3x2/US"

export type FlagComponent = ComponentType<{
  className?: string
  "aria-hidden"?: boolean
  title?: string
}>

const flags: Record<string, FlagComponent> = {
  DE,
  FI,
  FR,
  GB,
  HK,
  JP,
  NL,
  SE,
  SG,
  US,
}

export function flagFor(code: string | undefined | null): FlagComponent | null {
  if (!code) return null
  return flags[code.toUpperCase()] ?? null
}

export function regionTintFor(code: string | undefined | null): string {
  switch (code?.toUpperCase()) {
    case "DE":
      return "from-amber-500/12 via-red-500/8"
    case "FI":
      return "from-sky-500/14 via-blue-500/8"
    case "NL":
      return "from-red-500/12 via-blue-500/10"
    case "US":
      return "from-indigo-500/14 via-blue-500/8"
    case "JP":
      return "from-pink-500/12 via-red-500/8"
    case "SG":
      return "from-red-500/14 via-rose-500/8"
    case "FR":
      return "from-blue-500/12 via-red-500/8"
    case "GB":
      return "from-rose-500/12 via-blue-500/8"
    case "SE":
      return "from-blue-500/12 via-amber-500/8"
    case "HK":
      return "from-red-500/14 via-rose-500/8"
    default:
      return "from-foreground/8 via-foreground/4"
  }
}

export function regionGlowFor(code: string | undefined | null): string {
  switch (code?.toUpperCase()) {
    case "DE":
      return "rgba(239, 68, 68, 0.22)"
    case "FI":
      return "rgba(59, 130, 246, 0.25)"
    case "NL":
      return "rgba(220, 38, 38, 0.22)"
    case "US":
      return "rgba(99, 102, 241, 0.25)"
    case "JP":
      return "rgba(236, 72, 153, 0.22)"
    case "SG":
      return "rgba(225, 29, 72, 0.25)"
    case "FR":
      return "rgba(59, 130, 246, 0.22)"
    case "GB":
      return "rgba(244, 63, 94, 0.22)"
    case "SE":
      return "rgba(59, 130, 246, 0.22)"
    case "HK":
      return "rgba(220, 38, 38, 0.22)"
    default:
      return "rgba(120, 120, 120, 0.2)"
  }
}

export function pingTone(ping: number) {
  if (ping < 50) return "text-emerald-600 dark:text-emerald-400"
  if (ping < 120) return "text-amber-600 dark:text-amber-400"
  return "text-red-600 dark:text-red-400"
}

export function loadFromPing(ping: number): "low" | "medium" | "high" {
  if (ping < 60) return "low"
  if (ping < 130) return "medium"
  return "high"
}

export function loadFromPercent(percent: number | undefined): "low" | "medium" | "high" {
  if (typeof percent !== "number" || !Number.isFinite(percent)) return "low"
  if (percent < 40) return "low"
  if (percent < 80) return "medium"
  return "high"
}
