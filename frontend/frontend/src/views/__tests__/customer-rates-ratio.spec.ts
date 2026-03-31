import { describe, expect, it } from 'vitest'

import {
  clampPercent,
  normalizeRatioPairForEdit,
  normalizeRatioPayloadForSave,
} from '../customer-rates-ratio'

describe('customer-rates-ratio', () => {
  it('clampPercent clamps to [0, 100] with 2 decimals', () => {
    expect(clampPercent(-1)).toBe(0)
    expect(clampPercent(101)).toBe(100)
    expect(clampPercent(33.456)).toBe(33.46)
  })

  it('normalizeRatioPairForEdit keeps ratios independent', () => {
    const result = normalizeRatioPairForEdit({
      incrementStartAt: '2026-01-01',
      stockRatio: 0.9,
      incrementRatio: 0.2,
    })
    expect(result.stockPercent).toBe(90)
    expect(result.incrementPercent).toBe(20)
  })

  it('normalizeRatioPairForEdit defaults to 100/0 when increment start is empty', () => {
    const result = normalizeRatioPairForEdit({
      incrementStartAt: '',
      stockRatio: 0.2,
      incrementRatio: 0.8,
    })
    expect(result.stockPercent).toBe(100)
    expect(result.incrementPercent).toBe(0)
  })

  it('normalizeRatioPayloadForSave keeps independent values when increment start exists', () => {
    const result = normalizeRatioPayloadForSave({
      incrementStartAt: '2026-01-01',
      stockPercent: 70,
      incrementPercent: 20,
    })
    expect(result.stockRatio).toBe(0.7)
    expect(result.incrementRatio).toBe(0.2)
    expect(result.incrementStartAt).toBe('2026-01-01')
  })

  it('normalizeRatioPayloadForSave forces 100/0 when increment start is empty', () => {
    const result = normalizeRatioPayloadForSave({
      incrementStartAt: '',
      stockPercent: 70,
      incrementPercent: 20,
    })
    expect(result.stockRatio).toBe(1)
    expect(result.incrementRatio).toBe(0)
    expect(result.incrementStartAt).toBeUndefined()
  })
})

