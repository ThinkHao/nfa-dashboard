import { describe, expect, it } from 'vitest'

import { buildSettlementDayRange, normalizeSettlementDayRange, splitSettlementDayRange } from '../settlement-day-range'

describe('settlement-day-range', () => {
  it('expands selected dates to day boundaries', () => {
    expect(normalizeSettlementDayRange(['2026-02-01', '2026-02-28'])).toEqual([
      '2026-02-01 00:00:00',
      '2026-02-28 23:59:59',
    ])
  })

  it('keeps an already normalized settlement range unchanged', () => {
    expect(normalizeSettlementDayRange(['2026-02-01 00:00:00', '2026-02-28 23:59:59'])).toEqual([
      '2026-02-01 00:00:00',
      '2026-02-28 23:59:59',
    ])
  })

  it('rebuilds the picker range from stored filter values', () => {
    expect(buildSettlementDayRange('2026-02-01 00:00:00', '2026-02-28 23:59:59')).toEqual([
      '2026-02-01 00:00:00',
      '2026-02-28 23:59:59',
    ])
  })

  it('splits settlement range into request-ready filter values', () => {
    expect(splitSettlementDayRange(['2026-02-01', '2026-02-28'])).toEqual({
      start: '2026-02-01 00:00:00',
      end: '2026-02-28 23:59:59',
    })
  })
})
