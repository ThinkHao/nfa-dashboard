import { describe, expect, it } from 'vitest'
import { comparisonRatioPercent, sampleComparisonChartPoints } from '../edc-nfa-chart-sampling'

describe('sampleComparisonChartPoints', () => {
  it('keeps the original points when the chart is already small', () => {
    const points = [{ bucket_5m: '2026-01-01T00:00:00Z', edc_mbps: 1, nfa_mbps: 2 }]

    expect(sampleComparisonChartPoints(points)).toBe(points)
  })

  it('caps long charts while preserving each bucket extrema and order', () => {
    const points = Array.from({ length: 10_000 }, (_, index) => ({
      bucket_5m: new Date(index * 300_000).toISOString(),
      edc_mbps: 0,
      nfa_mbps: 0,
      ratio: 0,
    }))
    points[5].edc_mbps = 1000
    points[15].nfa_mbps = 2000
    points[4321].ratio = 2

    const sampled = sampleComparisonChartPoints(points)

    expect(sampled.length).toBeLessThanOrEqual(5000)
    expect(sampled[0]).toBe(points[0])
    expect(sampled.at(-1)).toBe(points.at(-1))
    expect(sampled).toContain(points[5])
    expect(sampled).toContain(points[15])
    expect(sampled).toContain(points[4321])
    expect(sampled.map((point) => points.indexOf(point))).toEqual(
      [...sampled].map((point) => points.indexOf(point)).sort((a, b) => a - b),
    )
  })

  it('converts the supplied or derived ratio to percent and leaves zero EDC undefined', () => {
    expect(comparisonRatioPercent({ bucket_5m: '', edc_mbps: 100, nfa_mbps: 85, ratio: 0.8567 })).toBe(85.67)
    expect(comparisonRatioPercent({ bucket_5m: '', edc_mbps: 200, nfa_mbps: 150 })).toBe(75)
    expect(comparisonRatioPercent({ bucket_5m: '', edc_mbps: 0, nfa_mbps: 0 })).toBeNull()
  })
})
