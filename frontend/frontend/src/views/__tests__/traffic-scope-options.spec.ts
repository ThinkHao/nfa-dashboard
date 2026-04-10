import { describe, expect, it } from 'vitest'

import {
  buildTrafficScopeOptionRequest,
  formatTrafficScopeSchoolOptionLabel,
  shouldUseRemoteSchoolSearch,
} from '../traffic-scope-options'

describe('traffic-scope-options', () => {
  it('builds school option request from group context', () => {
    expect(
      buildTrafficScopeOptionRequest('school', [
        { dimension_type: 'region', dimension_value: '山东省' },
        { dimension_type: 'cp', dimension_value: 'bilibili' },
        { dimension_type: 'school', dimension_value: '' },
      ], '中国海洋'),
    ).toEqual({
      dimension: 'school',
      region: '山东省',
      cp: 'bilibili',
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
