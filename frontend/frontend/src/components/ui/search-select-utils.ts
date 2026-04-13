export type SearchOption = {
  label: string
  value: string | number
  raw?: unknown
}

function readOptionField(option: unknown, key: string): unknown {
  if (option && typeof option === 'object' && key in (option as Record<string, unknown>)) {
    return (option as Record<string, unknown>)[key]
  }
  return undefined
}

export function normalizeSearchOptions(
  options: unknown[],
  labelKey = 'label',
  valueKey = 'value',
): SearchOption[] {
  return (options || []).map((option) => {
    if (typeof option === 'string' || typeof option === 'number') {
      return {
        label: String(option),
        value: option,
        raw: option,
      }
    }

    const label = readOptionField(option, labelKey)
    const value = readOptionField(option, valueKey)

    return {
      label: label == null ? '' : String(label),
      value: (value as string | number) ?? '',
      raw: option,
    }
  })
}

export function findFirstMatchingOption(
  options: SearchOption[],
  query: string,
): SearchOption | null {
  const normalizedQuery = String(query || '').trim().toLowerCase()
  if (!normalizedQuery) return options[0] ?? null

  return options.find((option) => option.label.toLowerCase().includes(normalizedQuery)) ?? null
}
