export type TrafficByteUnitBase = 1000 | 1024
export type TrafficRateUnit = 'Mbps' | 'Gbps'

export function normalizeByteUnitBase(value: unknown, fallback: TrafficByteUnitBase = 1024): TrafficByteUnitBase {
  return value === 1000 ? 1000 : value === 1024 ? 1024 : fallback
}

export function normalizeRateUnit(value: unknown, fallback: TrafficRateUnit = 'Mbps'): TrafficRateUnit {
  return value === 'Gbps' ? 'Gbps' : value === 'Mbps' ? 'Mbps' : fallback
}

export function rateUnitDivisor(unit: TrafficRateUnit): number {
  return unit === 'Gbps' ? 1_000_000_000 : 1_000_000
}

export function settlementValueToBitsPerSecond(value: number): number {
  return (value * 8) / 60
}

export function bitsPerSecondToRate(bitsPerSecond: number, unit: TrafficRateUnit): number {
  return bitsPerSecond / rateUnitDivisor(unit)
}

export function settlementValueToRate(value: number, unit: TrafficRateUnit): number {
  return bitsPerSecondToRate(settlementValueToBitsPerSecond(value), unit)
}

export function formatRateValue(
  value: number | null | undefined,
  unit: TrafficRateUnit,
  withUnit = true,
  digits = 2,
): string {
  const n = Number(value)
  const safe = Number.isFinite(n) ? n : 0
  const formatted = safe.toFixed(digits)
  return withUnit ? `${formatted} ${unit}` : formatted
}
