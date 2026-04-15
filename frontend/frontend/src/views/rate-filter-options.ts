import api from '@/api'

export type RateSchoolOption = { name: string; inRate: boolean }

export function sanitizeScopeOptions(values: unknown[]): string[] {
  return Array.isArray(values)
    ? values
        .map((value) => String(value || '').trim())
        .filter((value) => value && value !== 'NULL')
    : []
}

export function buildRateSchoolOptions(baseSchools: any[], rateSchools: any[]): RateSchoolOption[] {
  const meta = new Map<string, RateSchoolOption>()

  const ensure = (name: string) => {
    if (!meta.has(name)) meta.set(name, { name, inRate: false })
    return meta.get(name)!
  }

  for (const item of Array.isArray(baseSchools) ? baseSchools : []) {
    const name = String(item?.school_name || item?.name || item || '').trim()
    if (name) ensure(name)
  }

  for (const item of Array.isArray(rateSchools) ? rateSchools : []) {
    const name = String(item?.school_name || item?.name || '').trim()
    if (name) ensure(name).inRate = true
  }

  return Array.from(meta.values()).sort((a, b) => a.name.localeCompare(b.name, 'zh-CN'))
}

export async function loadVisibleRateScopeOptions(): Promise<{ regions: string[]; cps: string[] }> {
  const [regions, cps] = await Promise.all([
    (api as any).v2.getRegions(),
    (api as any).v2.getCPs(),
  ])

  return {
    regions: sanitizeScopeOptions(regions),
    cps: sanitizeScopeOptions(cps),
  }
}

export async function searchRateSchoolOptions(
  params: {
    region?: string
    cp?: string
    schoolName?: string
  },
  listRates: (params?: any) => Promise<any>,
): Promise<RateSchoolOption[]> {
  const [baseSchools, rateSchools] = await Promise.all([
    (api as any).v2.getSchools({
      region: params.region || undefined,
      cp: params.cp || undefined,
      school_name: params.schoolName || undefined,
      limit: 200,
      offset: 0,
    }),
    listRates({
      region: params.region || undefined,
      cp: params.cp || undefined,
      school_name: params.schoolName || undefined,
      page: 1,
      page_size: params.schoolName ? 500 : 5000,
    }),
  ])

  const baseItems = Array.isArray((baseSchools as any)?.items)
    ? (baseSchools as any).items
    : (Array.isArray(baseSchools) ? baseSchools : [])
  const rateItems = Array.isArray((rateSchools as any)?.items) ? (rateSchools as any).items : []

  return buildRateSchoolOptions(baseItems, rateItems)
}
