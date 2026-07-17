import { resolveMonthRangeDateTime } from './settlement-user-query-utils'
import { sanitizeScopeOptionValues } from '@/utils/scope-options'

export type SettlementQueryFilters = {
  userId: number | null
  srcRegion: string
  region: string
  cp: string
  schoolName: string
}

type SettlementSchoolFilterRow = {
  src_region?: string | null
  region?: string | null
  cp?: string | null
}

function uniqueValues(values: Array<string | null | undefined>): string[] {
  return sanitizeScopeOptionValues(values).sort()
}

export function buildSettlementSchoolFilterOptions<T extends SettlementSchoolFilterRow>(
  schools: T[],
  srcRegion: string,
  region: string,
  cp: string,
): { srcRegions: string[]; regions: string[]; schools: T[] } {
  const rows = Array.isArray(schools) ? schools : []
  const srcScopedRows = srcRegion ? rows.filter((school) => school.src_region === srcRegion) : rows
  const filteredSchools = srcScopedRows.filter((school) => (
    (!region || school.region === region) && (!cp || school.cp === cp)
  ))

  return {
    srcRegions: uniqueValues(rows.map((school) => school.src_region)),
    regions: uniqueValues(srcScopedRows.map((school) => school.region)),
    schools: filteredSchools,
  }
}

export function buildSettlementQueryParams(
  filters: SettlementQueryFilters,
  monthRange: [string, string] | null,
  page: number,
  pageSize: number,
): Record<string, string | number> {
  const params: Record<string, string | number> = { page, page_size: pageSize }
  const { start, end } = resolveMonthRangeDateTime(monthRange)
  if (start) params.start_service_date = start
  if (end) params.end_service_date = end
  if (filters.userId != null && filters.userId > 0) params.channel_owner_user_id = filters.userId
  if (filters.srcRegion) params.src_region = filters.srcRegion
  if (filters.region) params.region = filters.region
  if (filters.cp) params.cp = filters.cp
  if (filters.schoolName) params.school_name = filters.schoolName
  return params
}

export function validateSettlementQueryRange(monthRange: [string, string] | null): string | null {
  if (!monthRange?.[0] || !monthRange?.[1]) return '请先选择服务月份范围'
  const [startYear, startMonth] = monthRange[0].split('-').map(Number)
  const [endYear, endMonth] = monthRange[1].split('-').map(Number)
  const months = (endYear - startYear) * 12 + endMonth - startMonth + 1
  if (months > 12) return '查询时间跨度最多 12 个月'
  return null
}
