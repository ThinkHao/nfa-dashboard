export type NodeMonthlyMetricRow = {
  id?: string
  metric: string
  rowType?: 'node' | 'total'
  isTotal?: boolean
  region?: string
  cp?: string
  monthlyDaily95Values: Record<string, string>
  monthlyAmountValues: Record<string, string>
  children?: NodeMonthlyMetricRow[]
}

type BuildOptions = {
  allowedMonthRange?: { startMonth: string; endMonth: string } | null
}

function isValidMonth(value: string): boolean {
  return /^\d{4}-\d{2}$/.test(value)
}

export function enumerateMonthsInRange(startMonth: string, endMonth: string): string[] {
  if (!isValidMonth(startMonth) || !isValidMonth(endMonth) || startMonth > endMonth) return []
  const [startYear, startM] = startMonth.split('-').map(Number)
  const [endYear, endM] = endMonth.split('-').map(Number)
  const months: string[] = []
  const cursor = new Date(startYear, startM - 1, 1)
  const end = new Date(endYear, endM - 1, 1)
  while (cursor.getTime() <= end.getTime()) {
    months.push(`${cursor.getFullYear()}-${String(cursor.getMonth() + 1).padStart(2, '0')}`)
    cursor.setMonth(cursor.getMonth() + 1)
  }
  return months
}

function parseServiceMonth(value: unknown): string {
  const raw = value == null ? '' : String(value).trim()
  if (!raw) return ''
  const datePart = raw.includes('T') ? raw.split('T')[0] : (raw.includes(' ') ? raw.split(' ')[0] : raw)
  return datePart.slice(0, 7)
}

function inAllowedMonthRange(month: string, allowed?: BuildOptions['allowedMonthRange']): boolean {
  if (!month) return false
  if (!allowed || !allowed.startMonth || !allowed.endMonth) return true
  return month >= allowed.startMonth && month <= allowed.endMonth
}

function toNum(value: unknown): number {
  if (value == null || value === '') return 0
  const n = Number(value)
  return Number.isFinite(n) ? n : 0
}

function fmt(value: number): string {
  return value.toFixed(2)
}

function monthValuesTemplate(months: string[]): Record<string, number> {
  const v: Record<string, number> = {}
  for (const m of months) v[m] = 0
  return v
}

type DailyNodeAgg = {
  row: any
  mbpsSum: number
  mbpsCount: number
  cpBill: number
  trafficBill: number
  rackBill: number
  otherBill: number
  totalBill: number
}

export function aggregateDailyNodeRowsByMonth(rows: any[]): any[] {
  const grouped = new Map<string, DailyNodeAgg>()

  for (const row of rows || []) {
    const month = parseServiceMonth(row?.settlement_time || row?.service_month)
    if (!month) continue
    const displayName = String(row?.display_name || '').trim() || '-'
    const region = String(row?.region || '')
    const cp = String(row?.cp || '')
    const unitBase = row?.unit_base ?? ''
    const key = `${displayName}__${region}__${cp}__${unitBase}__${month}`
    if (!grouped.has(key)) {
      grouped.set(key, {
        row: {
          ...row,
          service_month: month,
          settlement_time: `${month}-01 00:00:00`,
          settlement_mode: row?.settlement_mode || 'daily_95_avg',
        },
        mbpsSum: 0,
        mbpsCount: 0,
        cpBill: 0,
        trafficBill: 0,
        rackBill: 0,
        otherBill: 0,
        totalBill: 0,
      })
    }
    const agg = grouped.get(key)!
    const mbps = Number(row?.mbps_95)
    if (Number.isFinite(mbps)) {
      agg.mbpsSum += mbps
      agg.mbpsCount += 1
    }
    agg.cpBill += toNum(row?.cp_bill)
    agg.trafficBill += toNum(row?.traffic_bill)
    agg.rackBill += toNum(row?.rack_bill)
    agg.otherBill += toNum(row?.other_bill)
    agg.totalBill += toNum(row?.total_bill)
  }

  return Array.from(grouped.values()).map((agg) => ({
    ...agg.row,
    mbps_95: agg.mbpsCount > 0 ? agg.mbpsSum / agg.mbpsCount : 0,
    cp_bill: agg.cpBill,
    traffic_bill: agg.trafficBill,
    rack_bill: agg.rackBill,
    other_bill: agg.otherBill,
    total_bill: agg.totalBill,
  })).sort((a, b) => {
    const da = String(a?.service_month || a?.settlement_time || '')
    const db = String(b?.service_month || b?.settlement_time || '')
    const dateCmp = db.localeCompare(da)
    if (dateCmp !== 0) return dateCmp
    return String(a?.display_name || '').localeCompare(String(b?.display_name || ''))
  })
}

export function buildNodeMonthlyColumnView(
  monthlyRows: any[],
  dailyRows: any[],
  options: BuildOptions = {},
): { months: string[]; rows: NodeMonthlyMetricRow[] } {
  const allowed = options.allowedMonthRange

  // 收集月份集合（从月95数据中提取）
  const monthSet = new Set<string>()
  for (const row of monthlyRows || []) {
    const month = parseServiceMonth(row?.service_month || row?.settlement_time)
    if (month && inAllowedMonthRange(month, allowed)) monthSet.add(month)
  }
  for (const row of dailyRows || []) {
    const month = parseServiceMonth(row?.settlement_time)
    if (month && inAllowedMonthRange(month, allowed)) monthSet.add(month)
  }
  const months = Array.from(monthSet).sort((a, b) => a.localeCompare(b))
  if (!months.length) return { months: [], rows: [] }

  // ── 月95数据：按节点聚合 ──
  type NodeAgg = {
    region: string
    cp: string
    month95: Record<string, number>   // 月95值 (mbps_95)
    monthAmount: Record<string, number> // 月金额 (total_bill)
  }
  const nodeMap = new Map<string, NodeAgg>()

  for (const row of monthlyRows || []) {
    const month = parseServiceMonth(row?.service_month || row?.settlement_time)
    if (!month || !inAllowedMonthRange(month, allowed)) continue
    const name = String(row?.display_name || '').trim() || '-'
    if (!nodeMap.has(name)) {
      nodeMap.set(name, { region: String(row?.region || ''), cp: String(row?.cp || ''), month95: monthValuesTemplate(months), monthAmount: monthValuesTemplate(months) })
    }
    const agg = nodeMap.get(name)!
    agg.month95[month] += toNum(row?.mbps_95)
    agg.monthAmount[month] += toNum(row?.total_bill)
  }

  // ── 日95数据：按节点+月份计算日均95 ──
  const dailyAvg95 = new Map<string, Record<string, number>>()
  {
    // node -> month -> { sum, count }
    const tmp = new Map<string, Map<string, { sum: number; cnt: number }>>()
    for (const row of dailyRows || []) {
      const day = String(row?.settlement_time || '').slice(0, 10)
      const month = parseServiceMonth(day)
      if (!month || !inAllowedMonthRange(month, allowed)) continue
      const name = String(row?.display_name || '').trim() || '-'
      if (!tmp.has(name)) tmp.set(name, new Map())
      const mm = tmp.get(name)!
      if (!mm.has(month)) mm.set(month, { sum: 0, cnt: 0 })
      const agg = mm.get(month)!
      agg.sum += toNum(row?.mbps_95)
      agg.cnt += 1
    }
    for (const [name, mm] of tmp) {
      const rec: Record<string, number> = {}
      for (const [month, agg] of mm) {
        rec[month] = agg.cnt > 0 ? agg.sum / agg.cnt : 0
      }
      dailyAvg95.set(name, rec)
    }
  }

  // ── 构建行 ──
  const sortedNames = Array.from(nodeMap.keys()).sort((a, b) => a.localeCompare(b))
  const totalMonth95 = monthValuesTemplate(months)
  const totalMonthAmount = monthValuesTemplate(months)

  const rows: NodeMonthlyMetricRow[] = sortedNames.map((name) => {
    const agg = nodeMap.get(name)!
    const daily95 = dailyAvg95.get(name) || {}
    const monthlyDaily95Values: Record<string, string> = {}
    const monthlyAmountValues: Record<string, string> = {}
    for (const m of months) {
      // 如果有月95数据就用月95，否则用日均95
      const v95 = agg.month95[m] || daily95[m] || 0
      monthlyDaily95Values[m] = fmt(v95)
      monthlyAmountValues[m] = fmt(agg.monthAmount[m])
      totalMonth95[m] += v95
      totalMonthAmount[m] += agg.monthAmount[m]
    }
    return {
      id: `node:${name}`,
      metric: name,
      rowType: 'node',
      region: agg.region,
      cp: agg.cp,
      monthlyDaily95Values,
      monthlyAmountValues,
    }
  })

  // 总和行
  const totalDaily95Values: Record<string, string> = {}
  const totalAmountValues: Record<string, string> = {}
  for (const m of months) {
    totalDaily95Values[m] = fmt(totalMonth95[m])
    totalAmountValues[m] = fmt(totalMonthAmount[m])
  }
  rows.push({
    id: 'total',
    metric: '总和',
    rowType: 'total',
    isTotal: true,
    monthlyDaily95Values: totalDaily95Values,
    monthlyAmountValues: totalAmountValues,
  })

  return { months, rows }
}
