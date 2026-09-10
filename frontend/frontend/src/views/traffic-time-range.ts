export type TrafficTimeRangeOption =
  | 'last1h'
  | 'last6h'
  | 'last12h'
  | 'last24h'
  | 'last2d'
  | 'last7d'
  | 'last30d'
  | 'last6mo'
  | 'last1y'
  | 'custom'

function pad(value: number): string {
  return String(value).padStart(2, '0')
}

export function formatLocalDateTime(date: Date): string {
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

function subtractCalendarMonths(date: Date, months: number): Date {
  const result = new Date(date.getTime())
  const day = result.getDate()

  // 先移到当月 1 号，避免从月末回推时被 Date 自动溢出到下下个月。
  result.setDate(1)
  result.setMonth(result.getMonth() - months)
  const lastDay = new Date(result.getFullYear(), result.getMonth() + 1, 0).getDate()
  result.setDate(Math.min(day, lastDay))
  return result
}

export function resolvePresetTrafficRange(
  option: TrafficTimeRangeOption,
  now: Date = new Date(),
): [string, string] | null {
  let offsetMs: number | null = null

  switch (option) {
    case 'last1h':
      offsetMs = 1 * 60 * 60 * 1000
      break
    case 'last6h':
      offsetMs = 6 * 60 * 60 * 1000
      break
    case 'last12h':
      offsetMs = 12 * 60 * 60 * 1000
      break
    case 'last24h':
      offsetMs = 24 * 60 * 60 * 1000
      break
    case 'last2d':
      offsetMs = 2 * 24 * 60 * 60 * 1000
      break
    case 'last7d':
      offsetMs = 7 * 24 * 60 * 60 * 1000
      break
    case 'last30d':
      offsetMs = 30 * 24 * 60 * 60 * 1000
      break
    case 'last6mo': {
      const end = new Date(now.getTime())
      const start = subtractCalendarMonths(end, 6)
      return [formatLocalDateTime(start), formatLocalDateTime(end)]
    }
    case 'last1y': {
      const end = new Date(now.getTime())
      const start = subtractCalendarMonths(end, 12)
      return [formatLocalDateTime(start), formatLocalDateTime(end)]
    }
    case 'custom':
      return null
  }

  const end = new Date(now.getTime())
  const start = new Date(now.getTime() - offsetMs)
  return [formatLocalDateTime(start), formatLocalDateTime(end)]
}

export function clearTrafficCustomRange(): [string, string] | null {
  return null
}
