import { describe, expect, it } from 'vitest'

import {
  expandNodeDailyTaskRange,
  expandNodeMonthlyTaskRange,
} from '../settlement-task-batch-range'

describe('settlement-task-batch-range', () => {
  it('expands a single day range', () => {
    expect(expandNodeDailyTaskRange(['2026-06-08', '2026-06-08'])).toEqual({
      dates: ['2026-06-08'],
    })
  })

  it('expands a multi-day range across months', () => {
    expect(expandNodeDailyTaskRange(['2026-01-30', '2026-02-02'])).toEqual({
      dates: ['2026-01-30', '2026-01-31', '2026-02-01', '2026-02-02'],
    })
  })

  it('rejects a daily range whose end is before start', () => {
    expect(expandNodeDailyTaskRange(['2026-06-09', '2026-06-08'])).toEqual({
      error: '结束日期不能早于开始日期',
    })
  })

  it('rejects a daily range longer than 31 days', () => {
    expect(expandNodeDailyTaskRange(['2026-01-01', '2026-02-01'])).toEqual({
      error: '节点日95一次最多创建31天任务',
    })
  })

  it('expands a single month range', () => {
    expect(expandNodeMonthlyTaskRange(['2026-06', '2026-06'])).toEqual({
      months: ['2026-06'],
    })
  })

  it('expands a month range across years', () => {
    expect(expandNodeMonthlyTaskRange(['2025-11', '2026-02'])).toEqual({
      months: ['2025-11', '2025-12', '2026-01', '2026-02'],
    })
  })

  it('rejects a month range whose end is before start', () => {
    expect(expandNodeMonthlyTaskRange(['2026-07', '2026-06'])).toEqual({
      error: '结束月份不能早于开始月份',
    })
  })

  it('rejects a month range longer than 12 months', () => {
    expect(expandNodeMonthlyTaskRange(['2025-01', '2026-01'])).toEqual({
      error: '节点月95一次最多创建12个月任务',
    })
  })
})
