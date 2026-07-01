import { describe, expect, it } from 'vitest'
import { calculateTrafficP95 } from '@/views/traffic-percentile'

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
