import type { TrafficScopeCondition, TrafficScopeRuleGroup } from '@/types/api'

export function createEmptyTrafficScopeCondition(): TrafficScopeCondition {
  return {
    dimension_type: 'region',
    dimension_value: '',
  }
}

export function createEmptyTrafficScopeRuleGroup(): TrafficScopeRuleGroup {
  return {
    rule_type: 'allow',
    conditions: [createEmptyTrafficScopeCondition()],
  }
}

export function normalizeTrafficScopeRuleGroups(groups: TrafficScopeRuleGroup[]): TrafficScopeRuleGroup[] {
  return groups
    .map((group) => ({
      rule_type: group.rule_type,
      conditions: (group.conditions || [])
        .map((condition) => ({
          dimension_type: condition.dimension_type,
          dimension_value: (condition.dimension_value || '').trim(),
        }))
        .filter((condition) => condition.dimension_value),
    }))
    .filter((group) => group.conditions.length > 0)
}
