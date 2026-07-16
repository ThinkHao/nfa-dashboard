export type NormalizedPaginatedResponse<T> = {
  items: T[]
  total: number
}

export function normalizePaginatedResponse<T>(response: unknown): NormalizedPaginatedResponse<T> {
  if (Array.isArray(response)) return { items: response as T[], total: response.length }
  if (response && typeof response === 'object') {
    const candidate = response as { items?: unknown; total?: unknown }
    if (Array.isArray(candidate.items)) {
      const total = Number(candidate.total)
      return {
        items: candidate.items as T[],
        total: Number.isFinite(total) && total >= 0 ? total : candidate.items.length,
      }
    }
  }
  return { items: [], total: 0 }
}
