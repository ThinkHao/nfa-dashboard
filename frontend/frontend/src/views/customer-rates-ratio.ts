export function clampPercent(v: number): number {
  if (!Number.isFinite(v)) return 0
  if (v < 0) return 0
  if (v > 100) return 100
  return Math.round(v * 100) / 100
}

interface RatioEditInput {
  incrementStartAt?: string | null
  stockRatio?: number | null
  incrementRatio?: number | null
}

interface RatioEditResult {
  stockPercent: number
  incrementPercent: number
}

export function normalizeRatioPairForEdit(input: RatioEditInput): RatioEditResult {
  if (!input.incrementStartAt) {
    return { stockPercent: 100, incrementPercent: 0 }
  }
  return {
    stockPercent: clampPercent(Number(input.stockRatio ?? 0) * 100),
    incrementPercent: clampPercent(Number(input.incrementRatio ?? 0) * 100),
  }
}

interface RatioSaveInput {
  incrementStartAt?: string | null
  stockPercent: number
  incrementPercent: number
}

interface RatioSaveResult {
  incrementStartAt?: string
  stockRatio: number
  incrementRatio: number
}

export function normalizeRatioPayloadForSave(input: RatioSaveInput): RatioSaveResult {
  if (!input.incrementStartAt) {
    return {
      incrementStartAt: undefined,
      stockRatio: 1,
      incrementRatio: 0,
    }
  }
  return {
    incrementStartAt: input.incrementStartAt,
    stockRatio: clampPercent(input.stockPercent) / 100,
    incrementRatio: clampPercent(input.incrementPercent) / 100,
  }
}

