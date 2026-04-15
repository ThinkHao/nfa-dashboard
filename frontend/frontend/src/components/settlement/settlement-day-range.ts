import { buildRangeValue, normalizeRangeValue, type RangeValue } from '@/components/ui/unified-date-range-utils'

export function normalizeSettlementDayRange(value: RangeValue): RangeValue {
  return normalizeRangeValue(value, 'daterange', 'YYYY-MM-DD HH:mm:ss')
}

export function buildSettlementDayRange(start?: string | null, end?: string | null): RangeValue {
  return normalizeSettlementDayRange(buildRangeValue(start, end))
}

export function splitSettlementDayRange(value: RangeValue): { start: string; end: string } {
  const normalized = normalizeSettlementDayRange(value)
  return {
    start: normalized?.[0] || '',
    end: normalized?.[1] || '',
  }
}
