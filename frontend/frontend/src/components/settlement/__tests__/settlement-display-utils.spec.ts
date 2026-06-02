import { describe, expect, it } from 'vitest'

import { formatMbps95 } from '../settlement-display-utils'

describe('formatMbps95', () => {
  it('formats node settlement 95 Mbps values with two decimals', () => {
    expect(formatMbps95(12)).toBe('12.00')
    expect(formatMbps95(12.3456)).toBe('12.35')
    expect(formatMbps95(null)).toBe('0.00')
    expect(formatMbps95(undefined)).toBe('0.00')
  })
})
