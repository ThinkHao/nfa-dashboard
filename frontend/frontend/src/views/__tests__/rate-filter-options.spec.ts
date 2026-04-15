import { describe, expect, it } from 'vitest'

import { buildRateSchoolOptions, sanitizeScopeOptions } from '../rate-filter-options'

describe('rate-filter-options', () => {
  it('filters blank and NULL scope options', () => {
    expect(sanitizeScopeOptions(['华东', '', null, 'NULL', '  华北  '])).toEqual(['华东', '华北'])
  })

  it('merges base schools with rate schools and marks schools already in rate data', () => {
    expect(
      buildRateSchoolOptions(
        [{ school_name: '上海大学' }, { school_name: '复旦大学' }],
        [{ school_name: '复旦大学' }, { school_name: '交通大学' }],
      ),
    ).toEqual([
      { name: '复旦大学', inRate: true },
      { name: '交通大学', inRate: true },
      { name: '上海大学', inRate: false },
    ])
  })
})
