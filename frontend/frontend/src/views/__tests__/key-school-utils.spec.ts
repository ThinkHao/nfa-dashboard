import { describe, it, expect } from 'vitest'
import { buildKeySchoolSet, buildKeySchoolNameSet, isKeySchool } from '../key-school-utils'

describe('key-school-utils', () => {
  it('buildKeySchoolSet 收集 is_key_school===1 的 school_id（OR by school_id）', () => {
    const rows = [
      { school_id: 's1', is_key_school: 1 },
      { school_id: 's2', is_key_school: 0 },
      { school_id: 's3', is_key_school: 1 },
      { school_id: 's3', is_key_school: 0 }, // 同 id 另一 cp，非重点
    ]
    const set = buildKeySchoolSet(rows)
    expect(set.has('s1')).toBe(true)
    expect(set.has('s2')).toBe(false)
    expect(set.has('s3')).toBe(true) // OR：任一行为 1 即重点
  })

  it('buildKeySchoolNameSet 收集 is_key_school===1 的 school_name（结算视图按校名标注）', () => {
    const rows = [
      { school_id: '1590', school_name: '北京交通大学', is_key_school: 1 },
      { school_id: '1534', school_name: '北京服装学院', is_key_school: 0 },
      { school_id: '1051', school_name: '北京航空航天大学', is_key_school: 1 },
    ]
    const set = buildKeySchoolNameSet(rows)
    expect(set.has('北京交通大学')).toBe(true)
    expect(set.has('北京服装学院')).toBe(false)
    expect(set.has('北京航空航天大学')).toBe(true)
  })

  it('isKeySchool 按 key（school_id 或 school_name）命中，并容忍首尾空格', () => {
    const set = new Set(['s1', 's3'])
    expect(isKeySchool(set, 's1')).toBe(true)
    expect(isKeySchool(set, ' s3 ')).toBe(true)
    expect(isKeySchool(set, 's2')).toBe(false)
    expect(isKeySchool(set, undefined)).toBe(false)
    const names = new Set(['北京交通大学'])
    expect(isKeySchool(names, '北京交通大学')).toBe(true)
  })
})
