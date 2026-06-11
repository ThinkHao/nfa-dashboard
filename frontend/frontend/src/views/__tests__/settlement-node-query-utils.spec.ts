import { describe, expect, it } from 'vitest'
import { aggregateDailyNodeRowsByMonth, buildNodeMonthlyColumnView, enumerateMonthsInRange } from '@/views/settlement-node-query-utils'

describe('enumerateMonthsInRange', () => {
  it('returns every month in the selected inclusive range', () => {
    expect(enumerateMonthsInRange('2026-01', '2026-03')).toEqual(['2026-01', '2026-02', '2026-03'])
  })

  it('returns an empty list when the range is incomplete or reversed', () => {
    expect(enumerateMonthsInRange('', '2026-03')).toEqual([])
    expect(enumerateMonthsInRange('2026-04', '2026-03')).toEqual([])
  })
})

describe('buildNodeMonthlyColumnView', () => {
  it('builds node rows and a bottom total row for selected months', () => {
    const monthlyRows = [
      { display_name: '节点A', region: '华北', cp: 'CT', service_month: '2026-01', mbps_95: 10, total_bill: 100 },
      { display_name: '节点A', region: '华北', cp: 'CT', service_month: '2026-03', mbps_95: 30, total_bill: 300 },
      { display_name: '节点B', region: '华南', cp: 'CM', service_month: '2026-03', mbps_95: 5, total_bill: 50 },
    ]
    const dailyRows = [
      { display_name: '节点A', settlement_time: '2026-01-10', mbps_95: 8 },
      { display_name: '节点B', settlement_time: '2026-03-01', mbps_95: 4 },
    ]

    const result = buildNodeMonthlyColumnView(monthlyRows, dailyRows, {
      allowedMonthRange: { startMonth: '2026-01', endMonth: '2026-03' },
    })

    expect(result.months).toEqual(['2026-01', '2026-03'])
    expect(result.rows.map((row) => row.metric)).toEqual(['节点A', '节点B', '总和'])

    const nodeA = result.rows[0]
    expect(nodeA.monthlyDaily95Values['2026-01']).toBe('10.00')
    expect(nodeA.monthlyDaily95Values['2026-03']).toBe('30.00')
    expect(nodeA.monthlyAmountValues['2026-01']).toBe('100.00')
    expect(nodeA.monthlyAmountValues['2026-03']).toBe('300.00')

    const total = result.rows[result.rows.length - 1]
    expect(total.isTotal).toBe(true)
    expect(total.monthlyDaily95Values['2026-01']).toBe('10.00')
    expect(total.monthlyDaily95Values['2026-03']).toBe('35.00')
    expect(total.monthlyAmountValues['2026-01']).toBe('100.00')
    expect(total.monthlyAmountValues['2026-03']).toBe('350.00')
  })
})

describe('aggregateDailyNodeRowsByMonth', () => {
  it('aggregates daily95 rows by node and month for monthly settlement display', () => {
    const rows = [
      {
        display_name: '节点A',
        region: '华北',
        cp: 'CT',
        settlement_time: '2026-03-01 00:00:00',
        settlement_mode: 'daily_95_avg',
        unit_base: 1000,
        mbps_95: 10,
        cp_bill: 1,
        traffic_bill: 2,
        rack_bill: 3,
        other_bill: 4,
        total_bill: 10,
      },
      {
        display_name: '节点A',
        region: '华北',
        cp: 'CT',
        settlement_time: '2026-03-02 00:00:00',
        settlement_mode: 'daily_95_avg',
        unit_base: 1000,
        mbps_95: 20,
        cp_bill: 10,
        traffic_bill: 20,
        rack_bill: 30,
        other_bill: 40,
        total_bill: 100,
      },
      {
        display_name: '节点B',
        region: '华南',
        cp: 'CM',
        settlement_time: '2026-04-01 00:00:00',
        settlement_mode: 'daily_95_avg',
        unit_base: 1024,
        mbps_95: 5,
        total_bill: 8,
      },
    ]

    const result = aggregateDailyNodeRowsByMonth(rows)

    expect(result).toHaveLength(2)
    expect(result[0]).toMatchObject({
      display_name: '节点B',
      service_month: '2026-04',
      settlement_mode: 'daily_95_avg',
      unit_base: 1024,
      mbps_95: 5,
      total_bill: 8,
    })
    expect(result[1]).toMatchObject({
      display_name: '节点A',
      service_month: '2026-03',
      settlement_mode: 'daily_95_avg',
      unit_base: 1000,
      mbps_95: 15,
      cp_bill: 11,
      traffic_bill: 22,
      rack_bill: 33,
      other_bill: 44,
      total_bill: 110,
    })
  })
})
