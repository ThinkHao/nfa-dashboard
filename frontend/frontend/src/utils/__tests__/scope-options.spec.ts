import { describe, expect, it } from 'vitest'

import { sanitizeScopeOptionValues } from '@/utils/scope-options'

describe('sanitizeScopeOptionValues', () => {
  it('trims and removes duplicate scope option values while preserving order', () => {
    expect(sanitizeScopeOptionValues(['上海市', ' 上海市 ', '北京市', '上海市'])).toEqual([
      '上海市',
      '北京市',
    ])
  })

  it('filters empty and NULL values before deduplication', () => {
    expect(sanitizeScopeOptionValues(['', null, 'NULL', ' null ', '广东省', '广东省'])).toEqual([
      '广东省',
    ])
  })
})
