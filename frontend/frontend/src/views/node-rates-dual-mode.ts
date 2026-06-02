import type { UpsertRateNodeRequest } from '@/types/api'
import type { RateNode } from '@/types/api'

export type NodeRateMode = 'daily_95_avg' | 'range_95'

export interface NodeRateBaseFields {
  entity_id?: number | null
  display_name?: string | null
  region: string
  cp: string
}

export interface NodeRateModeFields {
  enabled: boolean
  had_existing?: boolean
  unit_base?: number
  cp_fee?: number | null
  cp_fee_owner_id?: number | null
  node_construction_fee?: number | null
  node_construction_fee_owner_id?: number | null
  rack_fee?: number | null
  rack_fee_owner_id?: number | null
  other_fee?: number | null
  other_fee_owner_id?: number | null
}

export type NodeRateModeForm = Record<NodeRateMode, NodeRateModeFields>

export type AggregatedNodeRateRow = RateNode & {
  configured_modes: NodeRateMode[]
  mode_rates: Partial<Record<NodeRateMode, RateNode>>
}

export function isNodeRateModeEnabled(mode: Pick<NodeRateModeFields, 'enabled' | 'node_construction_fee'>): boolean {
  return mode.enabled && mode.node_construction_fee !== undefined && mode.node_construction_fee !== null
}

export function buildNodeRateModePayloads(base: NodeRateBaseFields, modes: NodeRateModeForm): UpsertRateNodeRequest[] {
  return (Object.keys(modes) as NodeRateMode[]).flatMap((mode) => {
    const value = modes[mode]
    if (!isNodeRateModeEnabled(value) && !value.had_existing) return []
    return [{
      ...base,
      settlement_type: mode,
      settlement_mode: mode,
      unit_base: value.unit_base || 1000,
      cp_fee: value.enabled ? value.cp_fee : null,
      cp_fee_owner_id: value.enabled ? value.cp_fee_owner_id : null,
      node_construction_fee: value.enabled ? value.node_construction_fee : null,
      node_construction_fee_owner_id: value.enabled ? value.node_construction_fee_owner_id : null,
      rack_fee: value.enabled ? value.rack_fee : null,
      rack_fee_owner_id: value.enabled ? value.rack_fee_owner_id : null,
      other_fee: value.enabled ? value.other_fee : null,
      other_fee_owner_id: value.enabled ? value.other_fee_owner_id : null,
      enabled: value.enabled,
    }]
  })
}

export function normalizeNodeRateMode(mode?: string): NodeRateMode {
  return mode === 'range_95' || mode === 'monthly95' ? 'range_95' : 'daily_95_avg'
}

function nodeRateScopeKey(row: Pick<RateNode, 'entity_id' | 'region' | 'cp'>): string {
  const entityID = Number(row.entity_id || 0)
  if (entityID > 0) return `entity:${entityID}`
  return `default:${row.region}:${row.cp}`
}

function newerUpdatedAt(a?: string, b?: string): string | undefined {
  if (!a) return b
  if (!b) return a
  return a >= b ? a : b
}

export function aggregateNodeRateRows(rows: RateNode[]): AggregatedNodeRateRow[] {
  const grouped = new Map<string, AggregatedNodeRateRow>()
  for (const row of rows) {
    const mode = normalizeNodeRateMode(row.settlement_mode || row.settlement_type)
    const key = nodeRateScopeKey(row)
    const existing = grouped.get(key)
    if (!existing) {
      grouped.set(key, {
        ...row,
        configured_modes: [mode],
        mode_rates: { [mode]: row },
      })
      continue
    }
    existing.mode_rates[mode] = row
    if (!existing.configured_modes.includes(mode)) {
      existing.configured_modes.push(mode)
    }
    existing.updated_at = newerUpdatedAt(existing.updated_at, row.updated_at)
  }
  return Array.from(grouped.values())
}
