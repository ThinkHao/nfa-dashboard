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

function settlementValueToMbps(value: number): number {
  return (value * 8) / 60 / 1_000_000
}

export function normalizeDateText(value: unknown): string {
  const raw = value == null ? '' : String(value).trim()
  if (!raw) return ''
  if (raw.includes('T')) return raw.split('T')[0]
  if (raw.includes(' ')) return raw.split(' ')[0]
  return raw
}

function computeDaily95AvgBySchool(dailyRows: any[]): Map<string, number> {
  const dailySumBySchoolDate = new Map<string, number>()
  for (const row of dailyRows || []) {
    const schoolName = String(row?.school_name || '').trim() || '-'
    const serviceDate = normalizeDateText(row?.service_date)
    if (!serviceDate) continue
    const raw95 = Number(row?.settlement_value ?? NaN)
    if (!Number.isFinite(raw95)) continue
    const key = `${schoolName}__${serviceDate}`
    dailySumBySchoolDate.set(key, (dailySumBySchoolDate.get(key) || 0) + raw95)
  }

  const avgBySchool = new Map<string, { sumMbps: number; dayCount: number }>()
  for (const [key, dailyRawSum] of dailySumBySchoolDate.entries()) {
    const schoolName = key.split('__')[0]
    const prev = avgBySchool.get(schoolName) || { sumMbps: 0, dayCount: 0 }
    prev.sumMbps += settlementValueToMbps(dailyRawSum)
    prev.dayCount += 1
    avgBySchool.set(schoolName, prev)
  }

  const result = new Map<string, number>()
  for (const [schoolName, agg] of avgBySchool.entries()) {
    result.set(schoolName, agg.dayCount > 0 ? agg.sumMbps / agg.dayCount : 0)
  }
  return result
}

export function buildMonthlyAmountColumnView(rawRows: any[], dailyRows: any[] = rawRows): { months: string[]; rows: MonthlyMetricRow[] } {
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
      })
    }
    const sum = AMOUNT_FIELDS.reduce((acc, def) => acc + toAmount(row?.[def.key]), 0)
    const schoolAgg = bySchool.get(schoolName)!
    schoolAgg.monthValues[month] += sum
    if (!schoolAgg.stockStartAt) schoolAgg.stockStartAt = normalizeDateText(row?.stock_start_at)
    if (!schoolAgg.incrementStartAt) schoolAgg.incrementStartAt = normalizeDateText(row?.increment_start_at)
    monthlyTotal[month] += sum
  }

  const daily95AvgBySchool = computeDaily95AvgBySchool(dailyRows || [])
  const schoolRows: MonthlyMetricRow[] = Array.from(bySchool.entries())
    .sort((a, b) => a[0].localeCompare(b[0]))
    .map(([schoolName, agg]) => {
      const values: Record<string, string> = {}
      for (const month of months) values[month] = fmtAmount(agg.monthValues[month] || 0)
      return {
        metric: schoolName,
        stockStartAt: agg.stockStartAt || '-',
        incrementStartAt: agg.incrementStartAt || '-',
        daily95Mbps: fmtAmount(daily95AvgBySchool.get(schoolName) || 0),
        values,
      }
    })

  const totalValues: Record<string, string> = {}
  for (const month of months) {
    totalValues[month] = fmtAmount(monthlyTotal[month] || 0)
  }
  const totalDaily95 = schoolRows.reduce((sum, row) => sum + Number(row.daily95Mbps || 0), 0)
  schoolRows.push({ metric: '总和', isTotal: true, stockStartAt: '-', incrementStartAt: '-', daily95Mbps: fmtAmount(totalDaily95), values: totalValues })

  return { months, rows: schoolRows }
}
