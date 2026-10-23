import { describe, expect, it } from 'vitest'
import {
  aggregateTrafficP95PointsByTime,
  calculateTrafficDailyP95Average,
  calculateTrafficP95,
  selectTrafficP95PointsInTimeRange,
} from '@/views/traffic-percentile'

describe('calculateTrafficP95', () => {
  it('uses the same descending rank as backend settlement95 for 288 points', () => {
    const points = Array.from({ length: 288 }, (_, index) => ({
      time: index,
      recvBps: 288 - index,
    }))

    const result = calculateTrafficP95(points)

    expect(result).toMatchObject({
      valueBps: 274,
      timeMs: 14,
      index: 14,
      total: 288,
    })
  })

  it('calculates p95 from service traffic after callers aggregate v4 and v6 buckets', () => {
    const points = [
      { time: 1_000, recvBps: 200 },
      { time: 2_000, recvBps: 190 },
      ...Array.from({ length: 18 }, (_, index) => ({
        time: 3_000 + index * 1_000,
        recvBps: 180 - index,
      })),
    ]

    const result = calculateTrafficP95(points)

    expect(result?.valueBps).toBe(190)
    expect(result?.timeMs).toBe(2_000)
  })

  it('returns null when there are no valid points', () => {
    expect(calculateTrafficP95([])).toBeNull()
    expect(calculateTrafficP95([{ time: Number.NaN, recvBps: 1 }, { time: 1, recvBps: Number.NaN }])).toBeNull()
  })

  it('uses the earliest time when multiple points share the p95 value', () => {
    const points = [
      { time: 5_000, recvBps: 200 },
      { time: 3_000, recvBps: 190 },
      { time: 1_000, recvBps: 190 },
      ...Array.from({ length: 17 }, (_, index) => ({
        time: 6_000 + index * 1_000,
        recvBps: 180 - index,
      })),
    ]

    const result = calculateTrafficP95(points)

    expect(result?.valueBps).toBe(190)
    expect(result?.timeMs).toBe(1_000)
  })
})

describe('calculateTrafficDailyP95Average', () => {
  it('matches the range p95 when all points are on one day', () => {
    const start = new Date(2026, 0, 12).getTime()
    const points = Array.from({ length: 288 }, (_, index) => ({
      time: start + index * 5 * 60 * 1000,
      recvBps: 288 - index,
    }))

    expect(calculateTrafficDailyP95Average(points)).toMatchObject({
      valueBps: calculateTrafficP95(points)?.valueBps,
      dayCount: 1,
      pointCount: 288,
    })
  })

  it('averages each day p95 equally after timestamp values have been aggregated', () => {
    const firstDay = new Date(2026, 0, 12).getTime()
    const secondDay = new Date(2026, 0, 13).getTime()
    const points = [
      ...Array.from({ length: 100 }, (_, index) => ({
        time: firstDay + index * 5 * 60 * 1000,
        recvBps: 100 - index,
      })),
      ...Array.from({ length: 20 }, (_, index) => ({
        time: secondDay + index * 5 * 60 * 1000,
        recvBps: 210 - index,
      })),
    ]

    expect(calculateTrafficDailyP95Average(points)).toEqual({
      valueBps: 152,
      dayCount: 2,
      pointCount: 120,
    })
  })

  it('sums selected CP values at each timestamp before calculating daily p95', () => {
    const day1 = new Date(2026, 0, 12).getTime()
    const day2 = new Date(2026, 0, 13).getTime()
    const aggregated = aggregateTrafficP95PointsByTime([
      { time: day1 + 12 * 60 * 60 * 1000, recvBps: 80 },
      { time: day1 + 12 * 60 * 60 * 1000, recvBps: 20 },
      { time: day1 + 13 * 60 * 60 * 1000, recvBps: 70 },
      { time: day1 + 13 * 60 * 60 * 1000, recvBps: 10 },
      { time: day2 + 12 * 60 * 60 * 1000, recvBps: 60 },
      { time: day2 + 12 * 60 * 60 * 1000, recvBps: 40 },
      { time: day2 + 13 * 60 * 60 * 1000, recvBps: 90 },
      { time: day2 + 13 * 60 * 60 * 1000, recvBps: 0 },
    ])

    expect(aggregated.map((point) => point.recvBps)).toEqual([100, 80, 100, 90])
    expect(calculateTrafficDailyP95Average(aggregated)).toEqual({
      valueBps: 100,
      dayCount: 2,
      pointCount: 4,
    })
  })

  it('skips invalid points and returns null when no valid days remain', () => {
    expect(calculateTrafficDailyP95Average([
      { time: Number.NaN, recvBps: 1 },
      { time: 1, recvBps: Number.NaN },
    ])).toBeNull()
  })
})

describe('selectTrafficP95PointsInTimeRange', () => {
  const points = [
    { time: 1_000, recvBps: 10 },
    { time: 2_000, recvBps: 20 },
    { time: 3_000, recvBps: 30 },
    { time: 4_000, recvBps: 40 },
  ]

  it('selects the inclusive zoom window using binary-search boundaries', () => {
    expect(selectTrafficP95PointsInTimeRange(points, { startTime: 2_000, endTime: 3_000 })).toEqual(points.slice(1, 3))
    expect(selectTrafficP95PointsInTimeRange(points, { startTime: 3_000, endTime: 2_000 })).toEqual(points.slice(1, 3))
  })

  it('returns the existing array for the unzoomed range and no points for an invalid range', () => {
    expect(selectTrafficP95PointsInTimeRange(points, null)).toBe(points)
    expect(selectTrafficP95PointsInTimeRange(points, { startTime: Number.NaN, endTime: 2_000 })).toEqual([])
  })

  it('recomputes p95 from the visible portion after multi-CP values are summed by timestamp', () => {
    const aggregated = aggregateTrafficP95PointsByTime([
      { time: 1_000, recvBps: 1_000 },
      { time: 2_000, recvBps: 100 },
      { time: 2_000, recvBps: 200 },
      { time: 3_000, recvBps: 50 },
    ])

    const visiblePoints = selectTrafficP95PointsInTimeRange(aggregated, { startTime: 2_000, endTime: 3_000 })
    expect(calculateTrafficP95(visiblePoints)?.valueBps).toBe(300)
  })
})
