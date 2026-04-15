export type TrafficTimeRangeOption =
  | 'last1h'
  | 'last3h'
  | 'last6h'
  | 'last12h'
  | 'last24h'
  | 'last2d'
  | 'last7d'
  | 'last30d'
  | 'custom'

function pad(value: number): string {
  return String(value).padStart(2, '0')
}

export function formatLocalDateTime(date: Date): string {
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
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
    case 'last3h':
      offsetMs = 3 * 60 * 60 * 1000
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
