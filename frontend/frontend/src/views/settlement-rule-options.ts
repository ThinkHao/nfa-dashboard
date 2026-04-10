export function mergeScopeOptions(selected: string[], options: string[]) {
  const merged: string[] = []
  const seen = new Set<string>()

  for (const value of [...selected, ...options]) {
    const normalized = String(value || '').trim()
    if (!normalized || seen.has(normalized)) continue
    seen.add(normalized)
    merged.push(normalized)
  }

  return merged
}
