export interface DailyTrafficVolumeRow {
  date: string
  school_id?: string
  school_name: string
  region: string
  cp: string
  service_bytes: number
}

function pad(value: number): string {
  return String(value).padStart(2, '0')
}

export function formatLocalDate(date: Date): string {
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

export function defaultDailyTrafficDateRange(now: Date = new Date()): [string, string] {
  const end = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const start = new Date(end)
  start.setDate(start.getDate() - 6)
  return [formatLocalDate(start), formatLocalDate(end)]
}

export function formatDailyTrafficBytes(bytes: number, base: 1000 | 1024 = 1024): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  const index = Math.min(Math.floor(Math.log(bytes) / Math.log(base)), units.length - 1)
  const value = bytes / Math.pow(base, index)
  return `${value.toFixed(value >= 100 ? 0 : value >= 10 ? 1 : 2)} ${units[index]}`
}

export function totalDailyTrafficBytes(rows: DailyTrafficVolumeRow[]): number {
  return rows.reduce((total, row) => total + (Number(row.service_bytes) || 0), 0)
}

export function meanDailyTrafficBytes(rows: DailyTrafficVolumeRow[]): number {
  return rows.length ? totalDailyTrafficBytes(rows) / rows.length : 0
}
