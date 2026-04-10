import { describe, expect, it } from 'vitest'

import { mergeScopeOptions } from '../settlement-rule-options'

describe('settlement-rule-options', () => {
  it('preserves selected legacy values while merging fetched options', () => {
    expect(mergeScopeOptions(['华北', '历史CP'], ['华东', '华北', 'CMCC'])).toEqual(['华北', '历史CP', '华东', 'CMCC'])
  })

  it('drops empty values when building option lists', () => {
    expect(mergeScopeOptions([' ', '华南'], ['', 'CT'])).toEqual(['华南', 'CT'])
  })
})
