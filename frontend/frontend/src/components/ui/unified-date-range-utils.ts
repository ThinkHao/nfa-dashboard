export type RangeValue = [string, string] | null
export type UnifiedRangeType = 'daterange' | 'datetimerange' | 'monthrange'

function pad(value: number, size = 2): string {
  return String(value).padStart(size, '0')
}

function parseDateLike(value: string): Date | null {
  const trimmed = String(value || '').trim()
  if (!trimmed) return null

  if (/^\d{13}$/.test(trimmed)) {
    const date = new Date(Number(trimmed))
    return Number.isNaN(date.getTime()) ? null : date
  }

  if (/^\d{10}$/.test(trimmed)) {
    const date = new Date(Number(trimmed) * 1000)
    return Number.isNaN(date.getTime()) ? null : date
  }

  let match = trimmed.match(/^(\d{4})-(\d{2})$/)
  if (match) {
    return new Date(Number(match[1]), Number(match[2]) - 1, 1, 0, 0, 0, 0)
  }

  match = trimmed.match(/^(\d{4})-(\d{2})-(\d{2})$/)
  if (match) {
    return new Date(Number(match[1]), Number(match[2]) - 1, Number(match[3]), 0, 0, 0, 0)
  }

  match = trimmed.match(/^(\d{4})-(\d{2})-(\d{2})[ T](\d{2}):(\d{2})(?::(\d{2}))?$/)
  if (match) {
    return new Date(
      Number(match[1]),
      Number(match[2]) - 1,
      Number(match[3]),
      Number(match[4]),
      Number(match[5]),
      Number(match[6] || 0),
      0,
    )
  }

  const direct = new Date(trimmed)
  return Number.isNaN(direct.getTime()) ? null : direct
}

function startOfDay(date: Date): Date {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate(), 0, 0, 0, 0)
}

function endOfDay(date: Date): Date {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate(), 23, 59, 59, 0)
}

function startOfMonth(date: Date): Date {
  return new Date(date.getFullYear(), date.getMonth(), 1, 0, 0, 0, 0)
}

function endOfMonth(date: Date): Date {
  return new Date(date.getFullYear(), date.getMonth() + 1, 0, 23, 59, 59, 0)
}

function cloneDate(date: Date): Date {
  return new Date(date.getTime())
}

function formatDateTime(date: Date): string {
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

function formatDateOnly(date: Date): string {
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

function formatMonthOnly(date: Date): string {
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}`
}

function formatRangeBoundary(date: Date, valueFormat: string): string {
  if (valueFormat === 'x') return String(date.getTime())
  if (valueFormat === 'YYYY-MM') return formatMonthOnly(date)
  if (valueFormat === 'YYYY-MM-DD') return formatDateOnly(date)
  if (valueFormat === 'YYYY-MM-DD HH:mm:ss') return formatDateTime(date)
  if (valueFormat === 'YYYY-MM-DDTHH:mm:ss.SSSZ') return date.toISOString()
  return formatDateTime(date)
}

export function normalizeRangeValue(
  value: RangeValue,
  type: UnifiedRangeType,
  valueFormat: string,
): RangeValue {
  if (!value || value.length !== 2 || !value[0] || !value[1]) return null

  const startDate = parseDateLike(value[0])
  const endDate = parseDateLike(value[1])
  if (!startDate || !endDate) return value

  if (type === 'monthrange') {
    const start = startOfMonth(startDate)
    const end = endOfMonth(endDate)
    return [formatRangeBoundary(start, valueFormat), formatRangeBoundary(end, valueFormat)]
  }

  if (type === 'datetimerange') {
    return [
      formatRangeBoundary(cloneDate(startDate), valueFormat),
      formatRangeBoundary(cloneDate(endDate), valueFormat),
    ]
  }

  const start = startOfDay(startDate)
  const end = endOfDay(endDate)
  return [formatRangeBoundary(start, valueFormat), formatRangeBoundary(end, valueFormat)]
}

export function buildRangeValue(start?: string | null, end?: string | null): RangeValue {
  if (!start || !end) return null
  return [start, end]
}

export function splitRangeValue(value: RangeValue): { start: string; end: string } {
  if (!value || value.length !== 2) {
    return { start: '', end: '' }
  }

  return {
    start: value[0] || '',
    end: value[1] || '',
  }
}

export function expandMonthRangeToDateTime(startMonth: string, endMonth: string): { start: string; end: string } {
  const normalized = normalizeRangeValue([startMonth, endMonth], 'monthrange', 'YYYY-MM-DD HH:mm:ss')
  if (!normalized) return { start: '', end: '' }
  return {
    start: normalized[0],
    end: normalized[1],
  }
}
