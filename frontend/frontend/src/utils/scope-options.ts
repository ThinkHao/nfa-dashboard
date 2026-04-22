export function sanitizeScopeOptionValues(values: unknown[]): string[] {
  if (!Array.isArray(values)) return []
  return values
    .map((value) => String(value ?? '').trim())
    .filter((value) => value !== '' && value.toUpperCase() !== 'NULL')
}

