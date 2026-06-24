import { describe, it, expect } from 'vitest'
import { buildKeySchoolSet, isKeySchool } from '../key-school-utils'

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

  it('isKeySchool 按 school_id 命中', () => {
    const set = new Set(['s1', 's3'])
    expect(isKeySchool(set, 's1')).toBe(true)
    expect(isKeySchool(set, 's2')).toBe(false)
    expect(isKeySchool(set, undefined)).toBe(false)
  })
})
