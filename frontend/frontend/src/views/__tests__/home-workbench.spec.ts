import { describe, expect, it } from 'vitest'

import { buildQuickAccessItems } from '../home-workbench'

describe('home-workbench quick access', () => {
  it('shows only permission-allowed entries and keeps stable order', () => {
    const items = buildQuickAccessItems([
      'traffic.read',
      'settlement.data.read',
      'operation_logs.read',
    ])

    expect(items.map((x) => x.key)).toEqual([
      'traffic',
      'traffic-volume',
      'settlement-user-query',
      'operation-logs',
    ])
  })

  it('returns empty list when user has no related permissions', () => {
    expect(buildQuickAccessItems([])).toEqual([])
  })
})
