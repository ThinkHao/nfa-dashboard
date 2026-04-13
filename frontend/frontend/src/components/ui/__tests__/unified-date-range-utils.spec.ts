import { describe, expect, it } from 'vitest'

import { buildRangeValue, expandMonthRangeToDateTime, normalizeRangeValue, splitRangeValue } from '../unified-date-range-utils'

describe('unified-date-range-utils', () => {
  it('builds a range tuple only when both ends exist', () => {
    expect(buildRangeValue('2026-04-01', '2026-04-30')).toEqual(['2026-04-01', '2026-04-30'])
    expect(buildRangeValue('', '2026-04-30')).toBeNull()
    expect(buildRangeValue('2026-04-01', '')).toBeNull()
  })

  it('splits range tuple back into start and end values', () => {
    expect(splitRangeValue(['2026-04-01T00:00:00.000Z', '2026-04-10T00:00:00.000Z'])).toEqual({
      start: '2026-04-01T00:00:00.000Z',
      end: '2026-04-10T00:00:00.000Z',
    })

    expect(splitRangeValue(null)).toEqual({ start: '', end: '' })
  })

  it('normalizes daterange boundaries to start and end of day', () => {
    expect(normalizeRangeValue(['2026-04-01', '2026-04-10'], 'daterange', 'YYYY-MM-DD HH:mm:ss')).toEqual([
      '2026-04-01 00:00:00',
      '2026-04-10 23:59:59',
    ])
  })

  it('normalizes timestamp ranges to local start and end of day', () => {
    const start = String(new Date(2026, 3, 1, 12, 30, 0, 0).getTime())
    const end = String(new Date(2026, 3, 10, 8, 45, 0, 0).getTime())
    const normalized = normalizeRangeValue([start, end], 'datetimerange', 'x')
    expect(normalized).toEqual([
      String(new Date(2026, 3, 1, 0, 0, 0, 0).getTime()),
      String(new Date(2026, 3, 10, 23, 59, 59, 0).getTime()),
    ])
  })

  it('expands month ranges into month boundary datetimes', () => {
    expect(expandMonthRangeToDateTime('2026-04', '2026-05')).toEqual({
      start: '2026-04-01 00:00:00',
      end: '2026-05-31 23:59:59',
    })
  })
})
