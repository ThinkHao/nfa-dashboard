import type { TrafficByteUnitBase, TrafficRateUnit } from '@/utils/traffic-units'
import { normalizeByteUnitBase, settlementValueToRate } from '@/utils/traffic-units'

export type MonthlyMetricRow = {
  id?: string
  metric: string
  rowType?: 'region' | 'school' | 'cp' | 'total'
  isTotal?: boolean
  region?: string
  schoolId?: string
  schoolName?: string
  cp?: string
  stockStartAt?: string
  incrementStartAt?: string
  values: Record<string, string>
  monthlyDaily95Values: Record<string, string>
  monthlyAmountValues: Record<string, string>
  children?: MonthlyMetricRow[]
}

export type MonthRangeValue = [string, string] | null
type BuildMonthlyAmountColumnViewOptions = {
  treeByRegionSchoolCp?: boolean
  rateUnit?: TrafficRateUnit
  unitBase?: TrafficByteUnitBase
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

export function normalizeDateText(value: unknown): string {
  const raw = value == null ? '' : String(value).trim()
  if (!raw) return ''
  if (raw.includes('T')) return raw.split('T')[0]
  if (raw.includes(' ')) return raw.split(' ')[0]
  return raw
}

function parseDateOnly(value: unknown, monthBoundary: 'start' | 'end' = 'start'): Date | null {
  const normalized = normalizeDateText(value)
  if (!normalized) return null
  const monthMatch = normalized.match(/^(\d{4})-(\d{2})$/)
  if (monthMatch) {
    const y = Number(monthMatch[1])
    const m = Number(monthMatch[2])
    if (!Number.isFinite(y) || !Number.isFinite(m) || m < 1 || m > 12) return null
    const day = monthBoundary === 'end' ? new Date(y, m, 0).getDate() : 1
    return new Date(y, m - 1, day)
  }
  const dateMatch = normalized.match(/^(\d{4})-(\d{2})-(\d{2})/)
  if (!dateMatch) return null
  const y = Number(dateMatch[1])
  const m = Number(dateMatch[2])
  const d = Number(dateMatch[3])
  if (!Number.isFinite(y) || !Number.isFinite(m) || !Number.isFinite(d)) return null
  return new Date(y, m - 1, d)
}

export function pickEffectiveRate(rates: any[], serviceDateText: string): any | null {
  const serviceDate = parseDateOnly(serviceDateText, 'end')
  if (!serviceDate) return null
  const candidates = (rates || []).filter((rate) => {
    const startDate = parseDateOnly(rate?.start_at)
    if (!startDate) return true
    return startDate.getTime() <= serviceDate.getTime()
  })
  if (!candidates.length) return null
  return [...candidates].sort((a, b) => {
    const ad = parseDateOnly(a?.start_at)?.getTime() || 0
    const bd = parseDateOnly(b?.start_at)?.getTime() || 0
    return bd - ad
  })[0]
}

function monthValuesTemplate(months: string[]): Record<string, number> {
  const values: Record<string, number> = {}
  for (const month of months) values[month] = 0
  return values
}

function monthTextTemplate(months: string[], fallback = '0.00'): Record<string, string> {
  const values: Record<string, string> = {}
  for (const month of months) values[month] = fallback
  return values
}

function sumMonthlyAmount(row: any): number {
  return AMOUNT_FIELDS.reduce((acc, def) => acc + toAmount(row?.[def.key]), 0)
}

function normalizeSchoolId(row: any): string {
  const id = row?.school_id
  return id == null ? '' : String(id).trim()
}

function computeMonthlyDaily95AvgByGroup(
  dailyRows: any[],
  toGroupKey: (row: any) => string,
  rateUnit: TrafficRateUnit,
  unitBase: TrafficByteUnitBase,
): Map<string, Record<string, number>> {
  const groupDailyRawSum = new Map<string, Map<string, number>>()
  for (const row of dailyRows || []) {
    const groupKey = toGroupKey(row)
    const serviceDate = normalizeDateText(row?.service_date)
    if (!groupKey || !serviceDate) continue
    const raw95 = Number(row?.settlement_value ?? NaN)
    if (!Number.isFinite(raw95)) continue
    if (!groupDailyRawSum.has(groupKey)) groupDailyRawSum.set(groupKey, new Map<string, number>())
    const dayMap = groupDailyRawSum.get(groupKey)!
    dayMap.set(serviceDate, (dayMap.get(serviceDate) || 0) + raw95)
  }

  const result = new Map<string, Record<string, number>>()
  for (const [groupKey, dayMap] of groupDailyRawSum.entries()) {
    const monthAgg = new Map<string, { sumRate: number; dayCount: number }>()
    for (const [serviceDate, dailyRawSum] of dayMap.entries()) {
      const month = parseServiceMonth(serviceDate)
      if (!month) continue
      const prev = monthAgg.get(month) || { sumRate: 0, dayCount: 0 }
      prev.sumRate += settlementValueToRate(dailyRawSum, rateUnit, unitBase)
      prev.dayCount += 1
      monthAgg.set(month, prev)
    }
    const monthlyAverage: Record<string, number> = {}
    for (const [month, agg] of monthAgg.entries()) {
      monthlyAverage[month] = agg.dayCount > 0 ? agg.sumRate / agg.dayCount : 0
    }
    result.set(groupKey, monthlyAverage)
  }
  return result
}

function buildFlatRows(months: string[], rawRows: any[], dailyRows: any[], rateUnit: TrafficRateUnit, unitBase: TrafficByteUnitBase): MonthlyMetricRow[] {
  const bySchool = new Map<string, {
    schoolId: string
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
        schoolId: normalizeSchoolId(row),
        monthValues: monthValuesTemplate(months),
        stockStartAt: normalizeDateText(row?.stock_start_at),
        incrementStartAt: normalizeDateText(row?.increment_start_at),
      })
    }
    const sum = sumMonthlyAmount(row)
    const schoolAgg = bySchool.get(schoolName)!
    if (!schoolAgg.schoolId) schoolAgg.schoolId = normalizeSchoolId(row)
    schoolAgg.monthValues[month] += sum
    if (!schoolAgg.stockStartAt) schoolAgg.stockStartAt = normalizeDateText(row?.stock_start_at)
    if (!schoolAgg.incrementStartAt) schoolAgg.incrementStartAt = normalizeDateText(row?.increment_start_at)
    monthlyTotal[month] += sum
  }

  const monthlyDaily95BySchool = computeMonthlyDaily95AvgByGroup(
    dailyRows || [],
    (row) => String(row?.school_name || '').trim() || '-',
    rateUnit,
    unitBase,
  )
  const totalDaily95ByMonth: Record<string, number> = monthValuesTemplate(months)
  const schoolRows: MonthlyMetricRow[] = Array.from(bySchool.entries())
    .sort((a, b) => a[0].localeCompare(b[0]))
    .map(([schoolName, agg]) => {
      const monthlyAmountValues: Record<string, string> = {}
      const monthlyDaily95Values: Record<string, string> = {}
      const schoolMonthlyDaily95 = monthlyDaily95BySchool.get(schoolName) || {}
      for (const month of months) {
        monthlyAmountValues[month] = fmtAmount(agg.monthValues[month] || 0)
        const month95 = Number(schoolMonthlyDaily95[month] || 0)
        monthlyDaily95Values[month] = fmtAmount(month95)
        totalDaily95ByMonth[month] += month95
      }
      return {
        id: `school:${schoolName}`,
        metric: schoolName,
        rowType: 'school',
        schoolId: agg.schoolId || undefined,
        schoolName,
        stockStartAt: agg.stockStartAt || '-',
        incrementStartAt: agg.incrementStartAt || '-',
        values: monthlyAmountValues,
        monthlyAmountValues,
        monthlyDaily95Values,
      }
    })

  const totalAmountValues: Record<string, string> = {}
  const totalDaily95Values: Record<string, string> = {}
  for (const month of months) {
    totalAmountValues[month] = fmtAmount(monthlyTotal[month] || 0)
    totalDaily95Values[month] = fmtAmount(totalDaily95ByMonth[month] || 0)
  }
  schoolRows.push({
    id: 'total',
    metric: '总和',
    rowType: 'total',
    isTotal: true,
    stockStartAt: '-',
    incrementStartAt: '-',
    values: totalAmountValues,
    monthlyAmountValues: totalAmountValues,
    monthlyDaily95Values: totalDaily95Values,
  })

  return schoolRows
}

function buildTreeRows(months: string[], rawRows: any[], dailyRows: any[], rateUnit: TrafficRateUnit, unitBase: TrafficByteUnitBase): MonthlyMetricRow[] {
  type CpAgg = {
    region: string
    schoolId: string
    schoolName: string
    cp: string
    monthValues: Record<string, number>
    stockStartAt: string
    incrementStartAt: string
  }
  type SchoolAgg = {
    region: string
    schoolId: string
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
    const schoolId = normalizeSchoolId(row)
    const cp = String(row?.cp || '').trim() || '未知CP'
    if (!regionMap.has(region)) {
      regionMap.set(region, { region, schoolMap: new Map<string, SchoolAgg>() })
    }
    const regionAgg = regionMap.get(region)!
    if (!regionAgg.schoolMap.has(schoolName)) {
      regionAgg.schoolMap.set(schoolName, { region, schoolId, schoolName, cpMap: new Map<string, CpAgg>() })
    }
    const schoolAgg = regionAgg.schoolMap.get(schoolName)!
    if (!schoolAgg.schoolId) schoolAgg.schoolId = schoolId
    if (!schoolAgg.cpMap.has(cp)) {
      schoolAgg.cpMap.set(cp, {
        region,
        schoolId,
        schoolName,
        cp,
        monthValues: monthValuesTemplate(months),
        stockStartAt: normalizeDateText(row?.stock_start_at),
        incrementStartAt: normalizeDateText(row?.increment_start_at),
      })
    }
    const cpAgg = schoolAgg.cpMap.get(cp)!
    if (!cpAgg.schoolId) cpAgg.schoolId = schoolId
    const amount = sumMonthlyAmount(row)
    cpAgg.monthValues[month] += amount
    if (!cpAgg.stockStartAt) cpAgg.stockStartAt = normalizeDateText(row?.stock_start_at)
    if (!cpAgg.incrementStartAt) cpAgg.incrementStartAt = normalizeDateText(row?.increment_start_at)
    totalMonthValues[month] += amount
  }

  const cpMonthlyDaily95 = computeMonthlyDaily95AvgByGroup(dailyRows, (row) => {
    const region = String(row?.region || '').trim() || '未知区域'
    const schoolName = String(row?.school_name || '').trim() || '-'
    const cp = String(row?.cp || '').trim() || '未知CP'
    return `${region}__${schoolName}__${cp}`
  }, rateUnit, unitBase)

  const rows: MonthlyMetricRow[] = []
  const sortedRegions = Array.from(regionMap.keys()).sort((a, b) => a.localeCompare(b))
  for (const region of sortedRegions) {
    const regionAgg = regionMap.get(region)!
    const regionValuesNum: Record<string, number> = monthValuesTemplate(months)
    const regionDaily95Num: Record<string, number> = monthValuesTemplate(months)
    const schoolChildren: MonthlyMetricRow[] = []
    const sortedSchools = Array.from(regionAgg.schoolMap.keys()).sort((a, b) => a.localeCompare(b))
    for (const schoolName of sortedSchools) {
      const schoolAgg = regionAgg.schoolMap.get(schoolName)!
      const schoolValuesNum: Record<string, number> = monthValuesTemplate(months)
      const schoolDaily95Num: Record<string, number> = monthValuesTemplate(months)
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
        const cpDaily95ByMonth = cpMonthlyDaily95.get(cpKey) || {}
        const cpAmountValues: Record<string, string> = {}
        const cpDaily95Values: Record<string, string> = {}
        for (const month of months) {
          cpAmountValues[month] = fmtAmount(cpAgg.monthValues[month] || 0)
          const month95 = Number(cpDaily95ByMonth[month] || 0)
          cpDaily95Values[month] = fmtAmount(month95)
          schoolDaily95Num[month] += month95
        }
        cpChildren.push({
          id: `cp:${region}:${schoolName}:${cp}`,
          rowType: 'cp',
          metric: `CP：${cp}`,
          region,
          schoolId: cpAgg.schoolId || undefined,
          schoolName,
          cp,
          stockStartAt: cpAgg.stockStartAt || '-',
          incrementStartAt: cpAgg.incrementStartAt || '-',
          values: cpAmountValues,
          monthlyAmountValues: cpAmountValues,
          monthlyDaily95Values: cpDaily95Values,
        })
      }

      for (const month of months) {
        regionValuesNum[month] += schoolValuesNum[month]
        regionDaily95Num[month] += schoolDaily95Num[month]
      }
      const schoolAmountValues: Record<string, string> = {}
      const schoolDaily95Values: Record<string, string> = {}
      for (const month of months) {
        schoolAmountValues[month] = fmtAmount(schoolValuesNum[month] || 0)
        schoolDaily95Values[month] = fmtAmount(schoolDaily95Num[month] || 0)
      }
      schoolChildren.push({
        id: `school:${region}:${schoolName}`,
        rowType: 'school',
        metric: `学校：${schoolName}`,
        region,
        schoolId: schoolAgg.schoolId || undefined,
        schoolName,
        stockStartAt: schoolStockStartAt || '-',
        incrementStartAt: schoolIncrementStartAt || '-',
        values: schoolAmountValues,
        monthlyAmountValues: schoolAmountValues,
        monthlyDaily95Values: schoolDaily95Values,
        children: cpChildren,
      })
    }

    const regionAmountValues: Record<string, string> = {}
    const regionDaily95Values: Record<string, string> = {}
    for (const month of months) {
      regionAmountValues[month] = fmtAmount(regionValuesNum[month] || 0)
      regionDaily95Values[month] = fmtAmount(regionDaily95Num[month] || 0)
    }
    rows.push({
      id: `region:${region}`,
      rowType: 'region',
      metric: `区域：${region}`,
      region,
      stockStartAt: '-',
      incrementStartAt: '-',
      values: regionAmountValues,
      monthlyAmountValues: regionAmountValues,
      monthlyDaily95Values: regionDaily95Values,
      children: schoolChildren,
    })
  }

  const totalAmountValues: Record<string, string> = {}
  const totalDaily95Values: Record<string, string> = monthTextTemplate(months, '0.00')
  for (const month of months) {
    totalAmountValues[month] = fmtAmount(totalMonthValues[month] || 0)
    totalDaily95Values[month] = fmtAmount(rows.reduce((sum, row) => sum + Number(row.monthlyDaily95Values?.[month] || 0), 0))
  }
  rows.push({
    id: 'total',
    metric: '总和',
    rowType: 'total',
    isTotal: true,
    stockStartAt: '-',
    incrementStartAt: '-',
    values: totalAmountValues,
    monthlyAmountValues: totalAmountValues,
    monthlyDaily95Values: totalDaily95Values,
  })

  return rows
}

export function buildMonthlyAmountColumnView(rawRows: any[], dailyRows: any[] = rawRows, options: BuildMonthlyAmountColumnViewOptions = {}): { months: string[]; rows: MonthlyMetricRow[] } {
  const rateUnit: TrafficRateUnit = options.rateUnit === 'Gbps' ? 'Gbps' : 'Mbps'
  const unitBase: TrafficByteUnitBase = normalizeByteUnitBase(options.unitBase, 1000)
  const clippedRawRows = clipRowsByAllowedMonthRange(rawRows || [], options.allowedMonthRange)
  const clippedDailyRows = clipRowsByAllowedMonthRange(dailyRows || [], options.allowedMonthRange)
  const monthSet = new Set<string>()
  for (const row of clippedRawRows || []) {
    const month = parseServiceMonth(row?.service_date)
    if (month && inAllowedMonthRange(month, options.allowedMonthRange)) monthSet.add(month)
  }

  const months = Array.from(monthSet).sort((a, b) => a.localeCompare(b))
  if (options.treeByRegionSchoolCp) {
    return { months, rows: buildTreeRows(months, clippedRawRows, clippedDailyRows, rateUnit, unitBase) }
  }
  return { months, rows: buildFlatRows(months, clippedRawRows, clippedDailyRows, rateUnit, unitBase) }
}
