import { describe, expect, it } from 'vitest'
import { bitsPerSecondToRate, rateUnitDivisor, settlementValueToRate } from '@/utils/traffic-units'

describe('traffic-units', () => {
  it('uses decimal divisors by default to keep backward compatibility', () => {
    expect(rateUnitDivisor('Mbps')).toBe(1_000_000)
    expect(rateUnitDivisor('Gbps')).toBe(1_000_000_000)
  })

  it('supports binary unit base when unitBase=1024', () => {
    expect(rateUnitDivisor('Mbps', 1024)).toBe(1_048_576)
    expect(rateUnitDivisor('Gbps', 1024)).toBe(1_073_741_824)
  })

  it('converts the same raw value differently under 1000 and 1024 base', () => {
    const bitsPerSecond = 1_500_000_000
    expect(bitsPerSecondToRate(bitsPerSecond, 'Gbps', 1000)).toBeCloseTo(1.5, 6)
    expect(bitsPerSecondToRate(bitsPerSecond, 'Gbps', 1024)).toBeCloseTo(1.396983862, 6)
  })

  it('applies unit base in settlementValueToRate', () => {
    const rawSettlementValue = 11_069_624_206.896551
    expect(settlementValueToRate(rawSettlementValue, 'Gbps', 1000)).toBeCloseTo(1.475949894, 6)
    expect(settlementValueToRate(rawSettlementValue, 'Gbps', 1024)).toBeCloseTo(1.374585456, 6)
  })
})
