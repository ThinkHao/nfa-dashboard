export interface ComparisonChartPoint {
  bucket_5m: string
  edc_mbps: number
  nfa_mbps: number
  ratio?: number | null
}

const maxComparisonChartPoints = 5000
const pointsPerSeries = Math.floor(maxComparisonChartPoints / 3)

function ratioForPoint(point: ComparisonChartPoint): number {
  if (point.ratio != null && Number.isFinite(point.ratio)) return point.ratio
  return point.edc_mbps > 0 ? point.nfa_mbps / point.edc_mbps : 0
}

function largestTriangleThreeBuckets<T extends ComparisonChartPoint>(
  points: T[],
  xValues: number[],
  targetLength: number,
  valueForPoint: (point: T) => number,
): number[] {
  if (points.length <= targetLength) return points.map((_, index) => index)

  const sampledIndexes = [0]
  const bucketWidth = (points.length - 2) / (targetLength - 2)
  let anchor = 0

  for (let bucket = 0; bucket < targetLength - 2; bucket += 1) {
    const averageStart = Math.floor((bucket + 1) * bucketWidth) + 1
    const averageEnd = Math.min(Math.floor((bucket + 2) * bucketWidth) + 1, points.length)
    let averageX = 0
    let averageY = 0
    const averageCount = Math.max(averageEnd - averageStart, 1)
    for (let index = averageStart; index < averageEnd; index += 1) {
      averageX += xValues[index]
      averageY += valueForPoint(points[index])
    }
    averageX /= averageCount
    averageY /= averageCount

    const rangeStart = Math.floor(bucket * bucketWidth) + 1
    const rangeEnd = Math.min(Math.floor((bucket + 1) * bucketWidth) + 1, points.length - 1)
    let selectedIndex = rangeStart
    let largestArea = -1
    for (let index = rangeStart; index < rangeEnd; index += 1) {
      const area = Math.abs(
        (xValues[anchor] - averageX) * (valueForPoint(points[index]) - valueForPoint(points[anchor]))
        - (xValues[anchor] - xValues[index]) * (averageY - valueForPoint(points[anchor])),
      )
      if (area > largestArea) {
        largestArea = area
        selectedIndex = index
      }
    }
    sampledIndexes.push(selectedIndex)
    anchor = selectedIndex
  }

  sampledIndexes.push(points.length - 1)
  return sampledIndexes
}

export function sampleComparisonChartPoints<T extends ComparisonChartPoint>(points: T[]): T[] {
  if (points.length <= maxComparisonChartPoints) return points

  const xValues = points.map((point) => Date.parse(point.bucket_5m))
  const edcIndexes = largestTriangleThreeBuckets(points, xValues, pointsPerSeries, (point) => point.edc_mbps)
  const nfaIndexes = largestTriangleThreeBuckets(points, xValues, pointsPerSeries, (point) => point.nfa_mbps)
  const ratioIndexes = largestTriangleThreeBuckets(points, xValues, pointsPerSeries, ratioForPoint)
  const sampledIndexes = [...new Set([...edcIndexes, ...nfaIndexes, ...ratioIndexes])].sort((a, b) => a - b)
  return sampledIndexes.map((index) => points[index])
}

export function comparisonRatioPercent(point: ComparisonChartPoint): number | null {
  const ratio = point.ratio ?? (point.edc_mbps > 0 ? point.nfa_mbps / point.edc_mbps : null)
  return ratio != null && Number.isFinite(ratio) ? ratio * 100 : null
}
