export interface KeySchoolRow {
  school_id?: string | null
  school_name?: string | null
  is_key_school?: number | string | null
}

/** 收集重点院校 school_id 集合（按 school_id OR 聚合）。 */
export function buildKeySchoolSet(rows: KeySchoolRow[] | null | undefined): Set<string> {
  const set = new Set<string>()
  for (const r of rows || []) {
    const id = r?.school_id
    if (id == null || id === '') continue
    if (Number(r?.is_key_school) === 1) set.add(String(id))
  }
  return set
}

/**
 * 收集重点院校 school_name 集合（按 school_name OR 聚合）。
 * 用于结算视图：其数据源（settlement_customer/_monthly）只有 school_name、没有 school_id，
 * 只能按校名标注。纯展示用途，不参与任何结算数值计算。
 */
export function buildKeySchoolNameSet(rows: KeySchoolRow[] | null | undefined): Set<string> {
  const set = new Set<string>()
  for (const r of rows || []) {
    const name = r?.school_name
    if (name == null || name === '') continue
    if (Number(r?.is_key_school) === 1) set.add(String(name).trim())
  }
  return set
}

/** 某 key（school_id 或 school_name）是否命中重点院校集合。 */
export function isKeySchool(set: Set<string>, key: string | null | undefined): boolean {
  if (key == null || key === '') return false
  return set.has(String(key).trim())
}
