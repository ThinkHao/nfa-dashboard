import { describe, expect, it } from 'vitest'
import { buildMonthlyAmountColumnView, pickEffectiveRate, resolveMonthRangeDateTime } from '@/views/settlement-user-query-utils'

describe('buildMonthlyAmountColumnView', () => {
  it('builds month columns with school rows and bottom monthly total row', () => {
    const input = [
      { school_id: 's1', school_name: '学校A', service_date: '2026-02-01', stock_start_at: '2024-01-01', increment_start_at: '2025-01-01', customer_bill: 10, network_line_bill: 20, node_deduction_bill: 30, channel_bill: 40 },
      { school_id: 's1', school_name: '学校A', service_date: '2026-01-01', stock_start_at: '2024-01-01', increment_start_at: '2025-01-01', customer_bill: 1, network_line_bill: 2, node_deduction_bill: 3, channel_bill: 4 },
      { school_name: '学校B', service_date: '2026-02-28', stock_start_at: '2023-06-01', increment_start_at: '2024-06-01', customer_bill: 5, network_line_bill: null, node_deduction_bill: 6, channel_bill: 7 },
    ]

    const result = buildMonthlyAmountColumnView(input)

    expect(result.months).toEqual(['2026-01', '2026-02'])
    expect(result.rows.map((r) => r.metric)).toEqual(['学校A', '学校B', '总和'])

    const schoolA = result.rows[0]
    expect(schoolA.schoolId).toBe('s1')
    expect(schoolA.stockStartAt).toBe('2024-01-01')
    expect(schoolA.incrementStartAt).toBe('2025-01-01')
    expect(schoolA.monthlyDaily95Values['2026-01']).toBe('0.00')
    expect(schoolA.monthlyDaily95Values['2026-02']).toBe('0.00')
    expect(schoolA.monthlyAmountValues['2026-01']).toBe('10.00')
    expect(schoolA.monthlyAmountValues['2026-02']).toBe('100.00')

    const schoolB = result.rows[1]
    expect(schoolB.stockStartAt).toBe('2023-06-01')
    expect(schoolB.incrementStartAt).toBe('2024-06-01')
    expect(schoolB.monthlyDaily95Values['2026-01']).toBe('0.00')
    expect(schoolB.monthlyDaily95Values['2026-02']).toBe('0.00')
    expect(schoolB.monthlyAmountValues['2026-01']).toBe('0.00')
    expect(schoolB.monthlyAmountValues['2026-02']).toBe('18.00')

    const totalRow = result.rows[2]
    expect(totalRow.isTotal).toBe(true)
    expect(totalRow.monthlyDaily95Values['2026-01']).toBe('0.00')
    expect(totalRow.monthlyDaily95Values['2026-02']).toBe('0.00')
    expect(totalRow.monthlyAmountValues['2026-01']).toBe('10.00')
    expect(totalRow.monthlyAmountValues['2026-02']).toBe('118.00')
  })

  it('recalculates daily95 by daily school alignment (sum CP first, then average by day)', () => {
    const monthlyRows = [
      { school_name: '学校A', service_date: '2026-01-01', stock_start_at: '2024-01-01', increment_start_at: '2025-01-01', customer_bill: 10, network_line_bill: 0, node_deduction_bill: 0, channel_bill: 0 },
      { school_name: '学校A', service_date: '2026-02-01', stock_start_at: '2024-01-01', increment_start_at: '2025-01-01', customer_bill: 20, network_line_bill: 0, node_deduction_bill: 0, channel_bill: 0 },
      { school_name: '学校B', service_date: '2026-02-01', stock_start_at: '2023-06-01', increment_start_at: '2024-06-01', customer_bill: 5, network_line_bill: 5, node_deduction_bill: 0, channel_bill: 0 },
    ]
    const dailyRows = [
      { school_name: '学校A', service_date: '2026-01-10', cp: 'CT', settlement_value: 75_000_000 },
      { school_name: '学校A', service_date: '2026-01-10', cp: 'CM', settlement_value: 150_000_000 },
      { school_name: '学校A', service_date: '2026-01-11', cp: 'CT', settlement_value: 75_000_000 },
      { school_name: '学校B', service_date: '2026-02-11', cp: 'CT', settlement_value: 37_500_000 },
    ]

    const result = buildMonthlyAmountColumnView(monthlyRows, dailyRows)
    expect(result.months).toEqual(['2026-01', '2026-02'])
    expect(result.rows.map((r) => r.metric)).toEqual(['学校A', '学校B', '总和'])

    const schoolA = result.rows[0]
    expect(schoolA.monthlyDaily95Values['2026-01']).toBe('20.00')
    expect(schoolA.monthlyDaily95Values['2026-02']).toBe('0.00')
    expect(schoolA.monthlyAmountValues['2026-01']).toBe('10.00')
    expect(schoolA.monthlyAmountValues['2026-02']).toBe('20.00')

    const schoolB = result.rows[1]
    expect(schoolB.monthlyDaily95Values['2026-01']).toBe('0.00')
    expect(schoolB.monthlyDaily95Values['2026-02']).toBe('5.00')
    expect(schoolB.monthlyAmountValues['2026-01']).toBe('0.00')
    expect(schoolB.monthlyAmountValues['2026-02']).toBe('10.00')

    const totalRow = result.rows[2]
    expect(totalRow.monthlyDaily95Values['2026-01']).toBe('20.00')
    expect(totalRow.monthlyDaily95Values['2026-02']).toBe('5.00')
    expect(totalRow.monthlyAmountValues['2026-01']).toBe('10.00')
    expect(totalRow.monthlyAmountValues['2026-02']).toBe('30.00')
  })

  it('supports Gbps output when rateUnit is Gbps', () => {
    const monthlyRows = [
      { school_name: '学校A', service_date: '2026-01-01', customer_bill: 1, network_line_bill: 0, node_deduction_bill: 0, channel_bill: 0 },
    ]
    const dailyRows = [
      { school_name: '学校A', service_date: '2026-01-01', cp: 'CT', settlement_value: 75_000_000 },
      { school_name: '学校A', service_date: '2026-01-01', cp: 'CM', settlement_value: 150_000_000 },
    ]
    const result = buildMonthlyAmountColumnView(monthlyRows, dailyRows, { rateUnit: 'Gbps' })
    const schoolA = result.rows.find((r) => r.metric === '学校A')
    expect(schoolA?.monthlyDaily95Values['2026-01']).toBe('0.03')
  })

  it('uses unitBase=1024 when provided', () => {
    const monthlyRows = [
      { school_name: '学校A', service_date: '2026-01-01', customer_bill: 1, network_line_bill: 0, node_deduction_bill: 0, channel_bill: 0 },
    ]
    const dailyRows = [
      { school_name: '学校A', service_date: '2026-01-01', cp: 'CT', settlement_value: 75_000_000 },
      { school_name: '学校A', service_date: '2026-01-01', cp: 'CM', settlement_value: 150_000_000 },
    ]

    const resultDecimal = buildMonthlyAmountColumnView(monthlyRows, dailyRows, { rateUnit: 'Mbps', unitBase: 1000 })
    const resultBinary = buildMonthlyAmountColumnView(monthlyRows, dailyRows, { rateUnit: 'Mbps', unitBase: 1024 })

    const schoolDecimal = resultDecimal.rows.find((r) => r.metric === '学校A')
    const schoolBinary = resultBinary.rows.find((r) => r.metric === '学校A')
    expect(schoolDecimal?.monthlyDaily95Values['2026-01']).toBe('30.00')
    expect(schoolBinary?.monthlyDaily95Values['2026-01']).toBe('28.61')
  })

  it('builds region-school-cp tree rows with subtotal and total in tree mode', () => {
    const monthlyRows = [
      { school_id: 's1', region: '华北', cp: 'CT', school_name: '学校A', service_date: '2026-03', customer_bill: 10, network_line_bill: 1, node_deduction_bill: 1, channel_bill: 1, stock_start_at: '2024-01-01', increment_start_at: '2025-01-01' },
      { school_id: 's1', region: '华北', cp: 'CM', school_name: '学校A', service_date: '2026-03', customer_bill: 20, network_line_bill: 2, node_deduction_bill: 2, channel_bill: 2, stock_start_at: '2024-01-01', increment_start_at: '2025-01-01' },
      { region: '华南', cp: 'CT', school_name: '学校B', service_date: '2026-03', customer_bill: 30, network_line_bill: 3, node_deduction_bill: 3, channel_bill: 3, stock_start_at: '2024-02-01', increment_start_at: '2025-02-01' },
    ]
    const dailyRows = [
      { region: '华北', cp: 'CT', school_name: '学校A', service_date: '2026-03-01', settlement_value: 75_000_000 },
      { region: '华北', cp: 'CM', school_name: '学校A', service_date: '2026-03-01', settlement_value: 150_000_000 },
      { region: '华南', cp: 'CT', school_name: '学校B', service_date: '2026-03-01', settlement_value: 37_500_000 },
    ]

    const result = buildMonthlyAmountColumnView(monthlyRows, dailyRows, { treeByRegionSchoolCp: true })
    expect(result.rows.map((r) => r.rowType)).toEqual(['region', 'region', 'total'])

    const regionNorth = result.rows[0]
    expect(regionNorth.metric).toBe('区域：华北')
    expect(regionNorth.monthlyDaily95Values['2026-03']).toBe('30.00')
    expect(regionNorth.monthlyAmountValues['2026-03']).toBe('39.00')
    expect(regionNorth.children?.map((r) => r.rowType)).toEqual(['school'])

    const schoolA = regionNorth.children?.[0]
    expect(schoolA?.metric).toBe('学校：学校A')
    expect(schoolA?.schoolId).toBe('s1')
    expect(schoolA?.monthlyDaily95Values['2026-03']).toBe('30.00')
    expect(schoolA?.children?.map((r) => r.metric)).toEqual(['CP：CM', 'CP：CT'])
    expect(schoolA?.children?.map((r) => r.schoolId)).toEqual(['s1', 's1'])
    expect(schoolA?.children?.[0]?.monthlyDaily95Values['2026-03']).toBe('20.00')
    expect(schoolA?.children?.[1]?.monthlyDaily95Values['2026-03']).toBe('10.00')

    const total = result.rows[2]
    expect(total.metric).toBe('总和')
    expect(total.monthlyDaily95Values['2026-03']).toBe('35.00')
    expect(total.monthlyAmountValues['2026-03']).toBe('78.00')
  })

  it('clips out-of-range months in utility layer via allowedMonthRange', () => {
    const monthlyRows = [
      { school_name: '学校A', service_date: '2026-01-01', customer_bill: 10, network_line_bill: 0, node_deduction_bill: 0, channel_bill: 0 },
      { school_name: '学校A', service_date: '2026-03-01', customer_bill: 20, network_line_bill: 0, node_deduction_bill: 0, channel_bill: 0 },
      { school_name: '学校A', service_date: '2026-04-01', customer_bill: 30, network_line_bill: 0, node_deduction_bill: 0, channel_bill: 0 },
    ]
    const dailyRows = [
      { school_name: '学校A', service_date: '2026-01-10', cp: 'CT', settlement_value: 75_000_000 },
      { school_name: '学校A', service_date: '2026-03-10', cp: 'CT', settlement_value: 75_000_000 },
      { school_name: '学校A', service_date: '2026-04-10', cp: 'CT', settlement_value: 75_000_000 },
    ]

    const result = buildMonthlyAmountColumnView(monthlyRows, dailyRows, {
      allowedMonthRange: { startMonth: '2026-01', endMonth: '2026-03' },
    })

    expect(result.months).toEqual(['2026-01', '2026-03'])
    const schoolA = result.rows.find((r) => r.metric === '学校A')
    expect(schoolA?.monthlyAmountValues['2026-01']).toBe('10.00')
    expect(schoolA?.monthlyAmountValues['2026-03']).toBe('20.00')
    expect(schoolA?.monthlyAmountValues['2026-04']).toBeUndefined()
    expect(schoolA?.monthlyDaily95Values['2026-01']).toBe('10.00')
    expect(schoolA?.monthlyDaily95Values['2026-03']).toBe('10.00')
  })

  it('clips tree rows by allowedMonthRange and keeps subtotal/total consistent', () => {
    const monthlyRows = [
      { region: '华北', cp: 'CT', school_name: '学校A', service_date: '2026-03', customer_bill: 10, network_line_bill: 0, node_deduction_bill: 0, channel_bill: 0 },
      { region: '华北', cp: 'CT', school_name: '学校A', service_date: '2026-04', customer_bill: 20, network_line_bill: 0, node_deduction_bill: 0, channel_bill: 0 },
      { region: '华南', cp: 'CM', school_name: '学校B', service_date: '2026-03', customer_bill: 30, network_line_bill: 0, node_deduction_bill: 0, channel_bill: 0 },
    ]
    const dailyRows = [
      { region: '华北', cp: 'CT', school_name: '学校A', service_date: '2026-03-01', settlement_value: 75_000_000 },
      { region: '华北', cp: 'CT', school_name: '学校A', service_date: '2026-04-01', settlement_value: 150_000_000 },
      { region: '华南', cp: 'CM', school_name: '学校B', service_date: '2026-03-01', settlement_value: 37_500_000 },
    ]
    const result = buildMonthlyAmountColumnView(monthlyRows, dailyRows, {
      treeByRegionSchoolCp: true,
      allowedMonthRange: { startMonth: '2026-03', endMonth: '2026-03' },
    })

    expect(result.months).toEqual(['2026-03'])
    expect(result.rows.map((r) => r.rowType)).toEqual(['region', 'region', 'total'])
    const north = result.rows.find((r) => r.metric === '区域：华北')
    expect(north?.monthlyAmountValues['2026-03']).toBe('10.00')
    expect(north?.monthlyAmountValues['2026-04']).toBeUndefined()
    const total = result.rows[result.rows.length - 1]
    expect(total.metric).toBe('总和')
    expect(total.monthlyAmountValues['2026-03']).toBe('40.00')
  })
})

describe('resolveMonthRangeDateTime', () => {
  it('returns exact month boundaries based on current monthRange value', () => {
    const boundary = resolveMonthRangeDateTime(['2026-03', '2026-03'])
    expect(boundary.start).toBe('2026-03-01 00:00:00')
    expect(boundary.end).toBe('2026-03-31 23:59:59')
  })

  it('returns empty boundaries when range is not ready', () => {
    expect(resolveMonthRangeDateTime(null)).toEqual({ start: '', end: '' })
    expect(resolveMonthRangeDateTime(['', '2026-03'] as any)).toEqual({ start: '', end: '' })
  })
})

describe('pickEffectiveRate', () => {
  it('uses the end of a YYYY-MM service month when matching start_at', () => {
    const rates = [
      { id: 1, start_at: '2026-04-01', increment_start_at: '2026-04-15' },
      { id: 2, start_at: '2026-06-01', increment_start_at: '2026-06-15' },
    ]

    expect(pickEffectiveRate(rates, '2026-05')).toEqual(rates[0])
  })
})
