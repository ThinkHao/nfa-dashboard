export function sanitizeScopeOptionValues(values: unknown[]): string[] {
  if (!Array.isArray(values)) return []
  const normalized = values
    .map((value) => String(value ?? '').trim())
    .filter((value) => value !== '' && value.toUpperCase() !== 'NULL')
  return Array.from(new Set(normalized))
}
