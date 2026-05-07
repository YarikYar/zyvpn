import {
  Crown,
  Gift,
  Shield,
  Sparkles,
  type LucideIcon,
} from "lucide-react"

import type { Plan } from "@/lib/api"

export type PlanVariant = "friend" | "starter" | "basic" | "pro"

export type PlanMeta = {
  variant: PlanVariant
  Icon: LucideIcon
  tint: string
  iconColor: string
  kicker: string
  glow: string
}

const meta: Record<PlanVariant, Omit<PlanMeta, "variant">> = {
  friend: {
    Icon: Gift,
    tint: "from-amber-500/12 via-orange-500/8",
    iconColor: "text-amber-500/20",
    kicker: "text-amber-600 dark:text-amber-400",
    glow: "rgba(245, 158, 11, 0.22)",
  },
  starter: {
    Icon: Sparkles,
    tint: "from-sky-500/12 via-blue-500/8",
    iconColor: "text-sky-500/20",
    kicker: "text-sky-600 dark:text-sky-400",
    glow: "rgba(59, 130, 246, 0.25)",
  },
  basic: {
    Icon: Shield,
    tint: "from-emerald-500/12 via-teal-500/8",
    iconColor: "text-emerald-500/20",
    kicker: "text-emerald-600 dark:text-emerald-400",
    glow: "rgba(16, 185, 129, 0.25)",
  },
  pro: {
    Icon: Crown,
    tint: "from-violet-500/12 via-fuchsia-500/8",
    iconColor: "text-violet-500/20",
    kicker: "text-violet-600 dark:text-violet-400",
    glow: "rgba(168, 85, 247, 0.25)",
  },
}

export function planMetaFor(plan: Plan): PlanMeta {
  const variant = derivePlanVariant(plan)
  return { variant, ...meta[variant] }
}

export function planMetaForVariant(variant: PlanVariant): PlanMeta {
  return { variant, ...meta[variant] }
}

function derivePlanVariant(plan: Plan): PlanVariant {
  const id = plan.id.toLowerCase()
  const name = plan.name.toLowerCase()

  if (plan.visible_to_referrer_id || id.includes("friend") || name.includes("friend")) {
    return "friend"
  }
  if (
    plan.popular ||
    id.includes("pro") ||
    id.includes("premium") ||
    name.includes("pro") ||
    name.includes("premium") ||
    plan.traffic_gb === null
  ) {
    return "pro"
  }
  if (
    id.includes("basic") ||
    id.includes("standard") ||
    name.includes("basic") ||
    name.includes("standard") ||
    plan.duration_days >= 30
  ) {
    return "basic"
  }
  return "starter"
}

export function planTierLabelKey(plan: Plan): "referral" | "entry" | "standard" | "premium" {
  const variant = derivePlanVariant(plan)
  switch (variant) {
    case "friend":
      return "referral"
    case "starter":
      return "entry"
    case "basic":
      return "standard"
    case "pro":
      return "premium"
  }
}

export function planActiveShadow(variant: PlanVariant) {
  return `0 18px 44px -18px rgba(0,0,0,0.4), 0 0 36px ${meta[variant].glow}`
}
