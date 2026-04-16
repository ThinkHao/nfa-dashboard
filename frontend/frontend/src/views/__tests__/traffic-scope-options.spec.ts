import { describe, expect, it } from 'vitest'

import {
  buildTrafficScopeOptionRequest,
  formatTrafficScopeSchoolOptionLabel,
  shouldUseRemoteSchoolSearch,
} from '../traffic-scope-options'

describe('traffic-scope-options', () => {
  it('builds school option request without region/cp linkage', () => {
    expect(
      buildTrafficScopeOptionRequest('school', '中国海洋'),
    ).toEqual({
      dimension: 'school',
      q: '中国海洋',
      limit: 50,
    })
  })

  it('formats school option labels with id and scope context', () => {
    expect(
      formatTrafficScopeSchoolOptionLabel({
        school_id: '1138',
        school_name: '中国海洋大学',
        region: '山东省',
        cp: 'bilibili',
      }),
    ).toBe('中国海洋大学 (1138) | 山东省 / bilibili')
  })

  it('only uses remote search for school dimension', () => {
    expect(shouldUseRemoteSchoolSearch('school')).toBe(true)
    expect(shouldUseRemoteSchoolSearch('region')).toBe(false)
    expect(shouldUseRemoteSchoolSearch('cp')).toBe(false)
  })
})
