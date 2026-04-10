import type { TrafficScopeCondition, TrafficScopeOptionItem, TrafficScopeOptionParams } from '@/types/api'

type TrafficScopeSchoolLikeOption = Partial<TrafficScopeOptionItem> & {
  school_id?: string
  school_name?: string
  region?: string
  cp?: string
}

export function shouldUseRemoteSchoolSearch(dimension: TrafficScopeCondition['dimension_type']) {
  return dimension === 'school'
}

export function buildTrafficScopeOptionRequest(
  dimension: TrafficScopeCondition['dimension_type'],
  conditions: TrafficScopeCondition[],
  q = '',
): TrafficScopeOptionParams {
  const payload: TrafficScopeOptionParams = { dimension }
  const regionValue = conditions.find((condition) => condition.dimension_type === 'region')?.dimension_value?.trim()
  const cpValue = conditions.find((condition) => condition.dimension_type === 'cp')?.dimension_value?.trim()
  const keyword = q.trim()

  if (dimension === 'school') {
    if (regionValue) payload.region = regionValue
    if (cpValue) payload.cp = cpValue
    if (keyword) payload.q = keyword
    payload.limit = 50
    return payload
  }

  if (keyword) payload.q = keyword
  payload.limit = 200
  return payload
}

export function formatTrafficScopeSchoolOptionLabel(option: TrafficScopeSchoolLikeOption) {
  const dimension = option.dimension ?? (option.school_id ? 'school' : 'region')
  if (dimension !== 'school') {
    return option.label ?? option.value
  }
  const baseLabel = option.label || `${option.school_name || option.value} (${option.school_id || option.value})`
  const scope = [option.region, option.cp].filter(Boolean).join(' / ')
  return scope ? `${baseLabel} | ${scope}` : baseLabel
}
