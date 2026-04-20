export type MonthlyMetricRow = {
  id?: string
  metric: string
  rowType?: 'region' | 'school' | 'cp' | 'total'
  isTotal?: boolean
  region?: string
  schoolName?: string
  cp?: string
  stockStartAt?: string
  incrementStartAt?: string
  daily95Mbps?: string
  values: Record<string, string>
  children?: MonthlyMetricRow[]
}

export type MonthRangeValue = [string, string] | null
type BuildMonthlyAmountColumnViewOptions = {
  treeByRegionSchoolCp?: boolean
  allowedMonthRange?: {
    startMonth: string
    endMonth: string
  } | null
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

function inAllowedMonthRange(month: string, allowedMonthRange?: BuildMonthlyAmountColumnViewOptions['allowedMonthRange']): boolean {
  if (!month) return false
  if (!allowedMonthRange || !allowedMonthRange.startMonth || !allowedMonthRange.endMonth) return true
  return month >= allowedMonthRange.startMonth && month <= allowedMonthRange.endMonth
}

function clipRowsByAllowedMonthRange<T extends Record<string, any>>(inputRows: T[], allowedMonthRange?: BuildMonthlyAmountColumnViewOptions['allowedMonthRange']): T[] {
  if (!allowedMonthRange || !allowedMonthRange.startMonth || !allowedMonthRange.endMonth) return inputRows || []
  return (inputRows || []).filter((row) => {
    const month = parseServiceMonth(row?.service_date)
    return inAllowedMonthRange(month, allowedMonthRange)
  })
}

function parseYearMonth(ym: string): Date | null {
  const text = String(ym || '').trim()
  const match = text.match(/^(\d{4})-(\d{2})$/)
  if (!match) return null
  const year = Number(match[1])
  const month = Number(match[2])
  if (!Number.isFinite(year) || !Number.isFinite(month) || month < 1 || month > 12) return null
  return new Date(year, month - 1, 1, 0, 0, 0, 0)
}

function pad2(value: number): string {
  return String(value).padStart(2, '0')
}

function formatDateTime(value: Date): string {
  return `${value.getFullYear()}-${pad2(value.getMonth() + 1)}-${pad2(value.getDate())} ${pad2(value.getHours())}:${pad2(value.getMinutes())}:${pad2(value.getSeconds())}`
}

export function resolveMonthRangeDateTime(range: MonthRangeValue): { start: string; end: string } {
  if (!range || !range[0] || !range[1]) return { start: '', end: '' }
  const startDate = parseYearMonth(range[0])
  const endMonth = parseYearMonth(range[1])
  if (!startDate || !endMonth) return { start: '', end: '' }
  const endDate = new Date(endMonth.getFullYear(), endMonth.getMonth() + 1, 0, 23, 59, 59, 0)
  return { start: formatDateTime(startDate), end: formatDateTime(endDate) }
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

function monthValuesTemplate(months: string[]): Record<string, number> {
  const values: Record<string, number> = {}
  for (const month of months) values[month] = 0
  return values
}

function sumMonthlyAmount(row: any): number {
  return AMOUNT_FIELDS.reduce((acc, def) => acc + toAmount(row?.[def.key]), 0)
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

function buildFlatRows(months: string[], rawRows: any[], dailyRows: any[]): MonthlyMetricRow[] {
  const bySchool = new Map<string, {
    monthValues: Record<string, number>
    stockStartAt: string
    incrementStartAt: string
  }>()
  const monthlyTotal: Record<string, number> = monthValuesTemplate(months)

  for (const row of rawRows || []) {
    const month = parseServiceMonth(row?.service_date)
    if (!month) continue
    const schoolName = String(row?.school_name || '').trim() || '-'
    if (!bySchool.has(schoolName)) {
      bySchool.set(schoolName, {
        monthValues: monthValuesTemplate(months),
        stockStartAt: normalizeDateText(row?.stock_start_at),
        incrementStartAt: normalizeDateText(row?.increment_start_at),
      })
    }
    const sum = sumMonthlyAmount(row)
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
        id: `school:${schoolName}`,
        metric: schoolName,
        rowType: 'school',
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
  schoolRows.push({ id: 'total', metric: '总和', rowType: 'total', isTotal: true, stockStartAt: '-', incrementStartAt: '-', daily95Mbps: fmtAmount(totalDaily95), values: totalValues })

  return schoolRows
}

function computeDaily95AvgByGroup(dailyRows: any[], toGroupKey: (row: any) => string): Map<string, number> {
  const daySumMap = new Map<string, number>()
  for (const row of dailyRows || []) {
    const groupKey = toGroupKey(row)
    const serviceDate = normalizeDateText(row?.service_date)
    if (!groupKey || !serviceDate) continue
    const raw95 = Number(row?.settlement_value ?? NaN)
    if (!Number.isFinite(raw95)) continue
    const dayKey = `${groupKey}__${serviceDate}`
    daySumMap.set(dayKey, (daySumMap.get(dayKey) || 0) + raw95)
  }

  const aggMap = new Map<string, { sumMbps: number; dayCount: number }>()
  for (const [dayKey, rawSum] of daySumMap.entries()) {
    const splitIndex = dayKey.lastIndexOf('__')
    if (splitIndex <= 0) continue
    const groupKey = dayKey.slice(0, splitIndex)
    const prev = aggMap.get(groupKey) || { sumMbps: 0, dayCount: 0 }
    prev.sumMbps += settlementValueToMbps(rawSum)
    prev.dayCount += 1
    aggMap.set(groupKey, prev)
  }

  const result = new Map<string, number>()
  for (const [groupKey, agg] of aggMap.entries()) {
    result.set(groupKey, agg.dayCount > 0 ? agg.sumMbps / agg.dayCount : 0)
  }
  return result
}

function buildTreeRows(months: string[], rawRows: any[], dailyRows: any[]): MonthlyMetricRow[] {
  type CpAgg = {
    region: string
    schoolName: string
    cp: string
    monthValues: Record<string, number>
    stockStartAt: string
    incrementStartAt: string
  }
  type SchoolAgg = {
    region: string
    schoolName: string
    cpMap: Map<string, CpAgg>
  }
  type RegionAgg = {
    region: string
    schoolMap: Map<string, SchoolAgg>
  }

  const regionMap = new Map<string, RegionAgg>()
  const totalMonthValues: Record<string, number> = monthValuesTemplate(months)

  for (const row of rawRows || []) {
    const month = parseServiceMonth(row?.service_date)
    if (!month) continue
    const region = String(row?.region || '').trim() || '未知区域'
    const schoolName = String(row?.school_name || '').trim() || '-'
    const cp = String(row?.cp || '').trim() || '未知CP'
    if (!regionMap.has(region)) {
      regionMap.set(region, { region, schoolMap: new Map<string, SchoolAgg>() })
    }
    const regionAgg = regionMap.get(region)!
    if (!regionAgg.schoolMap.has(schoolName)) {
      regionAgg.schoolMap.set(schoolName, { region, schoolName, cpMap: new Map<string, CpAgg>() })
    }
    const schoolAgg = regionAgg.schoolMap.get(schoolName)!
    if (!schoolAgg.cpMap.has(cp)) {
      schoolAgg.cpMap.set(cp, {
        region,
        schoolName,
        cp,
        monthValues: monthValuesTemplate(months),
        stockStartAt: normalizeDateText(row?.stock_start_at),
        incrementStartAt: normalizeDateText(row?.increment_start_at),
      })
    }
    const cpAgg = schoolAgg.cpMap.get(cp)!
    const amount = sumMonthlyAmount(row)
    cpAgg.monthValues[month] += amount
    if (!cpAgg.stockStartAt) cpAgg.stockStartAt = normalizeDateText(row?.stock_start_at)
    if (!cpAgg.incrementStartAt) cpAgg.incrementStartAt = normalizeDateText(row?.increment_start_at)
    totalMonthValues[month] += amount
  }

  const cpDaily95 = computeDaily95AvgByGroup(dailyRows, (row) => {
    const region = String(row?.region || '').trim() || '未知区域'
    const schoolName = String(row?.school_name || '').trim() || '-'
    const cp = String(row?.cp || '').trim() || '未知CP'
    return `${region}__${schoolName}__${cp}`
  })

  const rows: MonthlyMetricRow[] = []
  const sortedRegions = Array.from(regionMap.keys()).sort((a, b) => a.localeCompare(b))
  for (const region of sortedRegions) {
    const regionAgg = regionMap.get(region)!
    const regionValuesNum: Record<string, number> = monthValuesTemplate(months)
    let regionDaily95 = 0
    const schoolChildren: MonthlyMetricRow[] = []
    const sortedSchools = Array.from(regionAgg.schoolMap.keys()).sort((a, b) => a.localeCompare(b))
    for (const schoolName of sortedSchools) {
      const schoolAgg = regionAgg.schoolMap.get(schoolName)!
      const schoolValuesNum: Record<string, number> = monthValuesTemplate(months)
      let schoolDaily95 = 0
      let schoolStockStartAt = ''
      let schoolIncrementStartAt = ''
      const cpChildren: MonthlyMetricRow[] = []
      const sortedCps = Array.from(schoolAgg.cpMap.keys()).sort((a, b) => a.localeCompare(b))
      for (const cp of sortedCps) {
        const cpAgg = schoolAgg.cpMap.get(cp)!
        for (const month of months) {
          schoolValuesNum[month] += cpAgg.monthValues[month] || 0
        }
        if (!schoolStockStartAt && cpAgg.stockStartAt) schoolStockStartAt = cpAgg.stockStartAt
        if (!schoolIncrementStartAt && cpAgg.incrementStartAt) schoolIncrementStartAt = cpAgg.incrementStartAt
        const cpKey = `${region}__${schoolName}__${cp}`
        const cpDaily95Value = cpDaily95.get(cpKey) || 0
        schoolDaily95 += cpDaily95Value
        const cpValues: Record<string, string> = {}
        for (const month of months) cpValues[month] = fmtAmount(cpAgg.monthValues[month] || 0)
        cpChildren.push({
          id: `cp:${region}:${schoolName}:${cp}`,
          rowType: 'cp',
          metric: `CP：${cp}`,
          region,
          schoolName,
          cp,
          stockStartAt: cpAgg.stockStartAt || '-',
          incrementStartAt: cpAgg.incrementStartAt || '-',
          daily95Mbps: fmtAmount(cpDaily95Value),
          values: cpValues,
        })
      }

      for (const month of months) regionValuesNum[month] += schoolValuesNum[month]
      regionDaily95 += schoolDaily95
      const schoolValues: Record<string, string> = {}
      for (const month of months) schoolValues[month] = fmtAmount(schoolValuesNum[month] || 0)
      schoolChildren.push({
        id: `school:${region}:${schoolName}`,
        rowType: 'school',
        metric: `学校：${schoolName}`,
        region,
        schoolName,
        stockStartAt: schoolStockStartAt || '-',
        incrementStartAt: schoolIncrementStartAt || '-',
        daily95Mbps: fmtAmount(schoolDaily95),
        values: schoolValues,
        children: cpChildren,
      })
    }

    const regionValues: Record<string, string> = {}
    for (const month of months) regionValues[month] = fmtAmount(regionValuesNum[month] || 0)
    rows.push({
      id: `region:${region}`,
      rowType: 'region',
      metric: `区域：${region}`,
      region,
      stockStartAt: '-',
      incrementStartAt: '-',
      daily95Mbps: fmtAmount(regionDaily95),
      values: regionValues,
      children: schoolChildren,
    })
  }

  const totalValues: Record<string, string> = {}
  for (const month of months) totalValues[month] = fmtAmount(totalMonthValues[month] || 0)
  const totalDaily95 = rows.reduce((sum, row) => sum + Number(row.daily95Mbps || 0), 0)
  rows.push({
    id: 'total',
    metric: '总和',
    rowType: 'total',
    isTotal: true,
    stockStartAt: '-',
    incrementStartAt: '-',
    daily95Mbps: fmtAmount(totalDaily95),
    values: totalValues,
  })

  return rows
}

export function buildMonthlyAmountColumnView(rawRows: any[], dailyRows: any[] = rawRows, options: BuildMonthlyAmountColumnViewOptions = {}): { months: string[]; rows: MonthlyMetricRow[] } {
  const clippedRawRows = clipRowsByAllowedMonthRange(rawRows || [], options.allowedMonthRange)
  const clippedDailyRows = clipRowsByAllowedMonthRange(dailyRows || [], options.allowedMonthRange)
  const monthSet = new Set<string>()
  for (const row of clippedRawRows || []) {
    const month = parseServiceMonth(row?.service_date)
    if (month && inAllowedMonthRange(month, options.allowedMonthRange)) monthSet.add(month)
  }

  const months = Array.from(monthSet).sort((a, b) => a.localeCompare(b))
  if (options.treeByRegionSchoolCp) {
    return { months, rows: buildTreeRows(months, clippedRawRows, clippedDailyRows) }
  }
  return { months, rows: buildFlatRows(months, clippedRawRows, clippedDailyRows) }
}
