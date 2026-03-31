export type MonthlyMetricRow = {
  metric: string
  isTotal?: boolean
  stockStartAt?: string
  incrementStartAt?: string
  daily95Mbps?: string
  values: Record<string, string>
}

type AmountDef = { key: string }

const AMOUNT_FIELDS: AmountDef[] = [
  { key: 'customer_bill' },
  { key: 'network_line_bill' },
  { key: 'node_deduction_bill' },
  { key: 'channel_bill' },
]

function parseServiceMonth(value: unknown): string {
  const raw = value == null ? '' : String(value).trim()
  if (!raw) return ''
  const datePart = raw.includes('T') ? raw.split('T')[0] : (raw.includes(' ') ? raw.split(' ')[0] : raw)
  return datePart.slice(0, 7)
}

function toAmount(value: unknown): number {
  if (value == null || value === '') return 0
  const n = Number(value)
  return Number.isFinite(n) ? n : 0
}

function fmtAmount(value: number): string {
  return value.toFixed(2)
}

export function normalizeDateText(value: unknown): string {
  const raw = value == null ? '' : String(value).trim()
  if (!raw) return ''
  if (raw.includes('T')) return raw.split('T')[0]
  if (raw.includes(' ')) return raw.split(' ')[0]
  return raw
}

export function buildMonthlyAmountColumnView(rawRows: any[]): { months: string[]; rows: MonthlyMetricRow[] } {
  const monthSet = new Set<string>()
  for (const row of rawRows || []) {
    const month = parseServiceMonth(row?.service_date)
    if (month) monthSet.add(month)
  }

  const months = Array.from(monthSet).sort((a, b) => a.localeCompare(b))
  const bySchool = new Map<string, {
    monthValues: Record<string, number>
    stockStartAt: string
    incrementStartAt: string
    daily95Sum: number
    daily95Count: number
  }>()
  const monthlyTotal: Record<string, number> = {}
  for (const month of months) monthlyTotal[month] = 0

  for (const row of rawRows || []) {
    const month = parseServiceMonth(row?.service_date)
    if (!month) continue
    const schoolName = String(row?.school_name || '').trim() || '-'
    if (!bySchool.has(schoolName)) {
      const init: Record<string, number> = {}
      for (const m of months) init[m] = 0
      bySchool.set(schoolName, {
        monthValues: init,
        stockStartAt: normalizeDateText(row?.stock_start_at),
        incrementStartAt: normalizeDateText(row?.increment_start_at),
        daily95Sum: 0,
        daily95Count: 0,
      })
    }
    const sum = AMOUNT_FIELDS.reduce((acc, def) => acc + toAmount(row?.[def.key]), 0)
    const schoolAgg = bySchool.get(schoolName)!
    schoolAgg.monthValues[month] += sum
    if (!schoolAgg.stockStartAt) schoolAgg.stockStartAt = normalizeDateText(row?.stock_start_at)
    if (!schoolAgg.incrementStartAt) schoolAgg.incrementStartAt = normalizeDateText(row?.increment_start_at)
    const raw95 = Number(row?.settlement_value ?? NaN)
    if (Number.isFinite(raw95)) {
      schoolAgg.daily95Sum += (raw95 * 8) / 60 / 1_000_000
      schoolAgg.daily95Count += 1
    }
    monthlyTotal[month] += sum
  }

  const schoolRows: MonthlyMetricRow[] = Array.from(bySchool.entries())
    .sort((a, b) => a[0].localeCompare(b[0]))
    .map(([schoolName, agg]) => {
      const values: Record<string, string> = {}
      for (const month of months) values[month] = fmtAmount(agg.monthValues[month] || 0)
      return {
        metric: schoolName,
        stockStartAt: agg.stockStartAt || '-',
        incrementStartAt: agg.incrementStartAt || '-',
        daily95Mbps: agg.daily95Count > 0 ? fmtAmount(agg.daily95Sum / agg.daily95Count) : '0.00',
        values,
      }
    })

  const totalValues: Record<string, string> = {}
  for (const month of months) {
    totalValues[month] = fmtAmount(monthlyTotal[month] || 0)
  }
  schoolRows.push({ metric: '总和', isTotal: true, stockStartAt: '-', incrementStartAt: '-', daily95Mbps: '-', values: totalValues })

  return { months, rows: schoolRows }
}
