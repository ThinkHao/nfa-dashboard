export function formatMbps95(value: unknown): string {
  const n = Number(value ?? 0)
  return Number.isFinite(n) ? n.toFixed(2) : '0.00'
}
