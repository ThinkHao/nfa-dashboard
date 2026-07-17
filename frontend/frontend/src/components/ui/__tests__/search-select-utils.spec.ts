import { describe, expect, it } from 'vitest'

import { findFirstMatchingOption, normalizeSearchOptions } from '../search-select-utils'

describe('search-select-utils', () => {
  const options = [
    { label: '华东', value: 'east' },
    { label: '华北', value: 'north' },
    { label: '华南', value: 'south' },
  ]

  it('normalizes options into label/value pairs without creating input pseudo options', () => {
    expect(normalizeSearchOptions(options)).toEqual([
      { label: '华东', value: 'east', raw: options[0] },
      { label: '华北', value: 'north', raw: options[1] },
      { label: '华南', value: 'south', raw: options[2] },
    ])
  })

  it('filters blank and NULL labels globally', () => {
    const validOption = { label: '有效选项', value: 'valid' }

    expect(normalizeSearchOptions([
      '',
      null,
      'NULL',
      ' null ',
      { label: 'NULL', value: 'invalid' },
      validOption,
    ])).toEqual([
      { label: '有效选项', value: 'valid', raw: validOption },
    ])
  })

  it('returns the first real matching option for fuzzy input', () => {
    expect(findFirstMatchingOption(options, '华')).toEqual({ label: '华东', value: 'east' })
    expect(findFirstMatchingOption(options, '北')).toEqual({ label: '华北', value: 'north' })
  })

  it('returns null when no real option matches the input', () => {
    expect(findFirstMatchingOption(options, '实时输入内容')).toBeNull()
  })
})
