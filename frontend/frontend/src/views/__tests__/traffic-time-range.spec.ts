import { describe, expect, it } from 'vitest'

import { clearTrafficCustomRange, resolvePresetTrafficRange } from '../traffic-time-range'

describe('traffic-time-range', () => {
  it('uses the last hour preset as the initial local time range', () => {
    const now = new Date(2026, 3, 14, 11, 27, 29)

    expect(resolvePresetTrafficRange('last1h', now)).toEqual([
      '2026-04-14 10:27:29',
      '2026-04-14 11:27:29',
    ])
  })

  it('clears the custom picker state when switching to custom mode', () => {
    expect(resolvePresetTrafficRange('custom', new Date(2026, 3, 14, 11, 27, 29))).toBeNull()
    expect(clearTrafficCustomRange()).toBeNull()
  })

  it('builds longer preset ranges without using utc formatting', () => {
    const now = new Date(2026, 3, 14, 11, 27, 29)

    expect(resolvePresetTrafficRange('last2d', now)).toEqual([
      '2026-04-12 11:27:29',
      '2026-04-14 11:27:29',
    ])
  })
})
