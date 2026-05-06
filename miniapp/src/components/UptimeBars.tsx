import type { DailyUptime } from '../api/client'

// Компактный 30-дневный график: для каждого дня — узкая колонка,
// высота отражает uptime (0..1), цвет — диапазон. Серая колонка = нет данных.

interface Props {
  days: DailyUptime[]
}

export default function UptimeBars({ days }: Props) {
  return (
    <div className="flex items-end gap-[1px] h-6" title="Uptime по дням">
      {days.map((d) => {
        if (d.uptime === null) {
          return (
            <div
              key={d.date}
              className="flex-1 bg-gray-300 dark:bg-gray-700 rounded-sm h-2 self-end"
              title={`${d.date}: нет данных`}
            />
          )
        }
        const ratio = d.uptime
        const heightPct = Math.max(8, Math.round(ratio * 100))
        const color =
          ratio >= 0.99
            ? 'bg-green-500'
            : ratio >= 0.95
            ? 'bg-yellow-500'
            : 'bg-red-500'
        return (
          <div
            key={d.date}
            className={`flex-1 ${color} rounded-sm`}
            style={{ height: `${heightPct}%` }}
            title={`${d.date}: ${(ratio * 100).toFixed(2)}%`}
          />
        )
      })}
    </div>
  )
}
