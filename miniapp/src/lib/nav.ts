import {
  CreditCard,
  Gift,
  Globe2,
  House,
  Settings,
  type LucideIcon,
} from "lucide-react"

export type NavKey =
  | "Main"
  | "Region"
  | "Subscription"
  | "Referrals"
  | "Settings"

export type NavItem = {
  key: NavKey
  label: string
  icon: LucideIcon
}

export const navItems: NavItem[] = [
  { key: "Main", label: "Main", icon: House },
  { key: "Region", label: "Region", icon: Globe2 },
  { key: "Subscription", label: "Subscription", icon: CreditCard },
  { key: "Referrals", label: "Referrals", icon: Gift },
  { key: "Settings", label: "Settings", icon: Settings },
]
