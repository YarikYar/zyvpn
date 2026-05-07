export type TrafficLimit =
  | { kind: "unlimited" }
  | { kind: "limit"; gb: number }

export type ActiveSubscription = {
  planId: string
  planName: string
  startDate: Date
  endDate: Date
  totalDays: number
  remainingDays: number
  trafficUsedGb: number
  trafficLimit: TrafficLimit
}

export const MOCK_ACTIVE_SUBSCRIPTION: ActiveSubscription | null = {
  planId: "pro",
  planName: "Pro",
  startDate: new Date(2026, 4, 6),
  endDate: new Date(2026, 5, 5),
  totalDays: 30,
  remainingDays: 29,
  trafficUsedGb: 0,
  trafficLimit: { kind: "unlimited" },
}

export const MOCK_ACTIVE_REGION_CODE = "DE"

export type DayStatus = "ok" | "minor" | "major"

export type UptimeDay = {
  date: Date
  status: DayStatus
}

export type IncidentTitleKey = "packetLoss" | "maintenance"

export type Incident = {
  id: string
  date: Date
  durationLabel: string
  titleKey: IncidentTitleKey
  resolved: boolean
}

function buildUptimeMock(): UptimeDay[] {
  const days: UptimeDay[] = []
  const today = new Date()
  for (let i = 29; i >= 0; i--) {
    const d = new Date(today)
    d.setDate(today.getDate() - i)
    let status: DayStatus = "ok"
    if (i === 11) status = "minor"
    if (i === 22) status = "major"
    days.push({ date: d, status })
  }
  return days
}

export const MOCK_UPTIME: UptimeDay[] = buildUptimeMock()

export const MOCK_INCIDENTS: Incident[] = [
  {
    id: "inc-2",
    date: (() => {
      const d = new Date()
      d.setDate(d.getDate() - 11)
      return d
    })(),
    durationLabel: "12 min",
    titleKey: "packetLoss",
    resolved: true,
  },
  {
    id: "inc-1",
    date: (() => {
      const d = new Date()
      d.setDate(d.getDate() - 22)
      return d
    })(),
    durationLabel: "3h 04m",
    titleKey: "maintenance",
    resolved: true,
  },
]

