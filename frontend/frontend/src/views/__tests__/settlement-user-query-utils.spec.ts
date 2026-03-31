import { describe, expect, it } from 'vitest'
import { buildMonthlyAmountColumnView } from '@/views/settlement-user-query-utils'

describe('buildMonthlyAmountColumnView', () => {
  it('builds month columns with school rows and bottom monthly total row', () => {
    const input = [
      { school_name: '学校A', service_date: '2026-02-01', stock_start_at: '2024-01-01', increment_start_at: '2025-01-01', customer_bill: 10, network_line_bill: 20, node_deduction_bill: 30, channel_bill: 40 },
      { school_name: '学校A', service_date: '2026-01-01', stock_start_at: '2024-01-01', increment_start_at: '2025-01-01', customer_bill: 1, network_line_bill: 2, node_deduction_bill: 3, channel_bill: 4 },
      { school_name: '学校B', service_date: '2026-02-28', stock_start_at: '2023-06-01', increment_start_at: '2024-06-01', customer_bill: 5, network_line_bill: null, node_deduction_bill: 6, channel_bill: 7 },
    ]

    const result = buildMonthlyAmountColumnView(input)

    expect(result.months).toEqual(['2026-01', '2026-02'])
    expect(result.rows.map((r) => r.metric)).toEqual(['学校A', '学校B', '总和'])

    const schoolA = result.rows[0]
    expect(schoolA.stockStartAt).toBe('2024-01-01')
    expect(schoolA.incrementStartAt).toBe('2025-01-01')
    expect(schoolA.daily95Mbps).toBe('0.00')
    expect(schoolA.values['2026-01']).toBe('10.00')
    expect(schoolA.values['2026-02']).toBe('100.00')

    const schoolB = result.rows[1]
    expect(schoolB.stockStartAt).toBe('2023-06-01')
    expect(schoolB.incrementStartAt).toBe('2024-06-01')
    expect(schoolB.daily95Mbps).toBe('0.00')
    expect(schoolB.values['2026-01']).toBe('0.00')
    expect(schoolB.values['2026-02']).toBe('18.00')

    const totalRow = result.rows[2]
    expect(totalRow.isTotal).toBe(true)
    expect(totalRow.values['2026-01']).toBe('10.00')
    expect(totalRow.values['2026-02']).toBe('118.00')
  })
})
