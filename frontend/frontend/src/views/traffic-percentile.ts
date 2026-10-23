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

export interface TrafficDailyP95AverageResult {
  valueBps: number
  dayCount: number
  pointCount: number
}

export function aggregateTrafficP95PointsByTime(points: TrafficP95InputPoint[]): TrafficP95InputPoint[] {
  const totalsByTime = new Map<number, number>()
  for (const point of points) {
    const time = Number(point.time)
    if (!Number.isFinite(time)) continue
    const valueBps = Number(point.recvBps)
    totalsByTime.set(time, (totalsByTime.get(time) || 0) + valueBps)
  }
  return Array.from(totalsByTime, ([time, recvBps]) => ({ time, recvBps }))
    .sort((a, b) => a.time - b.time)
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

export function calculateTrafficDailyP95Average(points: TrafficP95InputPoint[]): TrafficDailyP95AverageResult | null {
  const pointsByDay = new Map<string, TrafficP95InputPoint[]>()

  for (const point of points) {
    const timeMs = Number(point.time)
    const valueBps = Number(point.recvBps)
    if (!Number.isFinite(timeMs) || !Number.isFinite(valueBps)) continue

    const date = new Date(timeMs)
    if (!Number.isFinite(date.getTime())) continue
    const day = `${date.getFullYear()}-${date.getMonth() + 1}-${date.getDate()}`
    const dayPoints = pointsByDay.get(day) || []
    dayPoints.push({ time: timeMs, recvBps: valueBps })
    pointsByDay.set(day, dayPoints)
  }

  let totalP95Bps = 0
  let dayCount = 0
  let pointCount = 0
  for (const dayPoints of pointsByDay.values()) {
    const dailyP95 = calculateTrafficP95(dayPoints)
    if (!dailyP95) continue
    totalP95Bps += dailyP95.valueBps
    dayCount += 1
    pointCount += dailyP95.total
  }

  if (dayCount === 0) return null
  return { valueBps: totalP95Bps / dayCount, dayCount, pointCount }
}
