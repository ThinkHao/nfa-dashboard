export interface KeySchoolRow {
  school_id?: string | null
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

/** 某 school_id 是否重点院校。 */
export function isKeySchool(set: Set<string>, schoolId: string | null | undefined): boolean {
  if (schoolId == null || schoolId === '') return false
  return set.has(String(schoolId))
}
