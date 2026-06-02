import { describe, expect, it } from 'vitest'

import { aggregateNodeRateRows, buildNodeRateModePayloads, isNodeRateModeEnabled } from '../node-rates-dual-mode'

describe('node-rates-dual-mode', () => {
  it('treats a mode as enabled only when traffic unit price is present', () => {
    expect(isNodeRateModeEnabled({ enabled: true, node_construction_fee: 2.8 })).toBe(true)
    expect(isNodeRateModeEnabled({ enabled: true, node_construction_fee: null })).toBe(false)
    expect(isNodeRateModeEnabled({ enabled: false, node_construction_fee: 2.8 })).toBe(false)
  })

  it('builds one payload for each enabled mode with traffic unit price', () => {
    const payloads = buildNodeRateModePayloads(
      { entity_id: 12, display_name: 'TJ-Node-A', region: '天津', cp: 'bilibili' },
      {
        daily_95_avg: { enabled: true, unit_base: 1000, node_construction_fee: 2.8, rack_fee: 100 },
        range_95: { enabled: true, unit_base: 1024, node_construction_fee: 3.2, other_fee: 20 },
      },
    )

    expect(payloads).toHaveLength(2)
    expect(payloads[0]).toMatchObject({ settlement_mode: 'daily_95_avg', settlement_type: 'daily_95_avg', unit_base: 1000, node_construction_fee: 2.8, rack_fee: 100 })
    expect(payloads[1]).toMatchObject({ settlement_mode: 'range_95', settlement_type: 'range_95', unit_base: 1024, node_construction_fee: 3.2, other_fee: 20 })
  })

  it('skips modes without traffic unit price', () => {
    const payloads = buildNodeRateModePayloads(
      { region: '天津', cp: 'bilibili' },
      {
        daily_95_avg: { enabled: true, unit_base: 1000, node_construction_fee: undefined, rack_fee: 100 },
        range_95: { enabled: true, unit_base: 1000, node_construction_fee: 3.2 },
      },
    )

    expect(payloads).toHaveLength(1)
    expect(payloads[0].settlement_mode).toBe('range_95')
  })

  it('aggregates two settlement modes for the same node into one list row', () => {
    const rows = aggregateNodeRateRows([
      {
        id: 1,
        entity_id: 12,
        display_name: 'TJ-Node-A',
        region: '天津',
        cp: 'bilibili',
        settlement_type: 'daily_95_avg',
        settlement_mode: 'daily_95_avg',
        unit_base: 1000,
        node_construction_fee: 2.8,
        updated_at: '2026-06-01 10:00:00',
      },
      {
        id: 2,
        entity_id: 12,
        display_name: 'TJ-Node-A',
        region: '天津',
        cp: 'bilibili',
        settlement_type: 'range_95',
        settlement_mode: 'range_95',
        unit_base: 1000,
        node_construction_fee: 3.2,
        updated_at: '2026-06-01 11:00:00',
      },
    ])

    expect(rows).toHaveLength(1)
    expect(rows[0].configured_modes).toEqual(['daily_95_avg', 'range_95'])
    expect(rows[0].mode_rates?.daily_95_avg?.node_construction_fee).toBe(2.8)
    expect(rows[0].mode_rates?.range_95?.node_construction_fee).toBe(3.2)
    expect(rows[0].updated_at).toBe('2026-06-01 11:00:00')
  })
})
