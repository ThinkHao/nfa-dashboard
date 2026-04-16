import type { TrafficScopeOptionItem, TrafficScopeOptionParams } from '@/types/api'

type TrafficScopeSchoolLikeOption = Partial<TrafficScopeOptionItem> & {
  school_id?: string
  school_name?: string
  region?: string
  cp?: string
}

export function shouldUseRemoteSchoolSearch(dimension: 'region' | 'cp' | 'school') {
  return dimension === 'school'
}

export function buildTrafficScopeOptionRequest(
  dimension: 'region' | 'cp' | 'school',
  q = '',
): TrafficScopeOptionParams {
  const payload: TrafficScopeOptionParams = { dimension }
  const keyword = q.trim()

  if (dimension === 'school') {
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
