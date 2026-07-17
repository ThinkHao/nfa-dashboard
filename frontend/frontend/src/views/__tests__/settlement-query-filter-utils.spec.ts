import { describe, expect, it } from 'vitest'
import { buildSettlementQueryParams, buildSettlementSchoolFilterOptions, validateSettlementQueryRange } from '@/views/settlement-query-filter-utils'

describe('buildSettlementSchoolFilterOptions', () => {
  const schools = [
    { school_name: '学校A', src_region: '北京市', region: '天津市', cp: 'bilibili' },
    { school_name: '学校B', src_region: '北京市', region: '河北省', cp: 'ali' },
    { school_name: '学校C', src_region: '广东省', region: '天津市', cp: 'bilibili' },
  ]

  it('uses source region to narrow regions without narrowing source-region options', () => {
    const options = buildSettlementSchoolFilterOptions(schools, '北京市', '', '')

    expect(options.srcRegions).toEqual(['北京市', '广东省'])
    expect(options.regions).toEqual(['天津市', '河北省'])
    expect(options.schools.map((school) => school.school_name)).toEqual(['学校A', '学校B'])
  })

  it('does not let the selected school region narrow source-region options', () => {
    const options = buildSettlementSchoolFilterOptions(schools, '', '天津市', '')

    expect(options.srcRegions).toEqual(['北京市', '广东省'])
    expect(options.schools.map((school) => school.school_name)).toEqual(['学校A', '学校C'])
  })
})

describe('buildSettlementQueryParams', () => {
  it('omits optional filters when only the required month range is selected', () => {
    const params = buildSettlementQueryParams(
      { userId: null, srcRegion: '', region: '', cp: '', schoolName: '' },
      ['2026-04', '2026-04'],
      1,
      20,
    )

    expect(params).toEqual({
      page: 1,
      page_size: 20,
      start_service_date: '2026-04-01 00:00:00',
      end_service_date: '2026-04-30 23:59:59',
    })
    expect(params).not.toHaveProperty('channel_owner_user_id')
  })

  it('includes the selected user, region, CP and school', () => {
    const params = buildSettlementQueryParams(
      { userId: 9, srcRegion: '天津市', region: '北京市', cp: 'bilibili', schoolName: '学校A' },
      ['2026-03', '2026-05'],
      2,
      50,
    )

    expect(params).toMatchObject({
      page: 2,
      page_size: 50,
      channel_owner_user_id: 9,
      src_region: '天津市',
      region: '北京市',
      cp: 'bilibili',
      school_name: '学校A',
    })
  })
})

describe('validateSettlementQueryRange', () => {
  it('accepts a valid range without requiring a user', () => {
    expect(validateSettlementQueryRange(['2026-01', '2026-12'])).toBeNull()
  })

  it('requires the service month range', () => {
    expect(validateSettlementQueryRange(null)).toBe('请先选择服务月份范围')
  })

  it('rejects ranges longer than 12 months', () => {
    expect(validateSettlementQueryRange(['2026-01', '2027-01'])).toBe('查询时间跨度最多 12 个月')
  })
})
