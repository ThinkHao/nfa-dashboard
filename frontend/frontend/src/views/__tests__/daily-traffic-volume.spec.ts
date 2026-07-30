import { describe, expect, it } from 'vitest'

import {
  defaultDailyTrafficDateRange,
  formatDailyTrafficBytes,
  meanDailyTrafficBytes,
  totalDailyTrafficBytes,
  uniqueDailyTrafficSchools,
} from '../daily-traffic-volume'

describe('daily traffic volume helpers', () => {
  it('builds an inclusive seven-day local date range', () => {
    expect(defaultDailyTrafficDateRange(new Date(2026, 6, 29, 13, 30))).toEqual([
      '2026-07-23',
      '2026-07-29',
    ])
  })

  it('formats byte volumes with the configured unit base', () => {
    expect(formatDailyTrafficBytes(1_000_000_000, 1000)).toBe('1.00 GB')
    expect(formatDailyTrafficBytes(1024 ** 3, 1024)).toBe('1.00 GB')
  })

  it('sums service bytes without adding another time or bit conversion', () => {
    const rows = [
      { date: '2026-07-28', school_name: 'A', region: 'R', cp: 'C', service_bytes: 100 },
      { date: '2026-07-29', school_name: 'A', region: 'R', cp: 'C', service_bytes: 250 },
    ]
    expect(totalDailyTrafficBytes(rows)).toBe(350)
    expect(meanDailyTrafficBytes(rows)).toBe(175)
    expect(meanDailyTrafficBytes([])).toBe(0)
  })

  it('deduplicates school options when CP is not selected', () => {
    expect(uniqueDailyTrafficSchools([
      { school_name: '学校A', region: '北京市', cp: 'bilibili' },
      { school_name: '学校A', region: '北京市', cp: 'douyin' },
      { school_name: '学校B', region: '北京市', cp: 'bilibili' },
    ])).toEqual([
      { school_name: '学校A', region: '北京市', cp: 'bilibili' },
      { school_name: '学校B', region: '北京市', cp: 'bilibili' },
    ])
  })
})
