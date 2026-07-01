export interface TrafficP95InputPoint {
  time: number
  recvBps: number
}

export interface TrafficP95Result {
  valueBps: number
  timeMs: number
  index: number
  total: number
}

export function calculateTrafficP95(points: TrafficP95InputPoint[]): TrafficP95Result | null {
  const validPoints = points
    .map((point) => ({
      timeMs: Number(point.time),
      valueBps: Number(point.recvBps),
    }))
    .filter((point) => Number.isFinite(point.timeMs) && Number.isFinite(point.valueBps))

  const total = validPoints.length
  if (total === 0) return null

  const sorted = [...validPoints].sort((a, b) => {
    if (b.valueBps !== a.valueBps) return b.valueBps - a.valueBps
    return a.timeMs - b.timeMs
  })
  const index = Math.max(0, Math.min(total - 1, total - Math.ceil(total * 0.95)))
  const selected = sorted[index]

  return {
    valueBps: selected.valueBps,
    timeMs: selected.timeMs,
    index,
    total,
  }
}
