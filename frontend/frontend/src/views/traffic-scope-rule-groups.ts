import type { TrafficScopeCondition, TrafficScopeRuleGroup } from '@/types/api'

export type TrafficScopeConditionForm = Omit<TrafficScopeCondition, 'dimension_value'> & {
  dimension_values: string[];
}

export type TrafficScopeRuleGroupForm = Omit<TrafficScopeRuleGroup, 'conditions'> & {
  conditions: TrafficScopeConditionForm[];
}

export function createEmptyTrafficScopeCondition(): TrafficScopeConditionForm {
  return {
    dimension_type: 'region',
    dimension_values: [],
  }
}

export function createEmptyTrafficScopeRuleGroup(): TrafficScopeRuleGroupForm {
  return {
    rule_type: 'allow',
    conditions: [createEmptyTrafficScopeCondition()],
  }
}

export function fromApiToUiRuleGroups(groups: TrafficScopeRuleGroup[]): TrafficScopeRuleGroupForm[] {
  return groups
    .map((group) => ({
      id: group.id,
      user_id: group.user_id,
      rule_type: group.rule_type,
      created_at: group.created_at,
      updated_at: group.updated_at,
      conditions: (group.conditions || [])
        .reduce<TrafficScopeConditionForm[]>((acc, condition) => {
          const value = (condition.dimension_value || '').trim()
          if (!value) {
            return acc
          }
          const existing = acc.find((item) => item.dimension_type === condition.dimension_type)
          if (existing) {
            if (!existing.dimension_values.includes(value)) {
              existing.dimension_values.push(value)
            }
            return acc
          }
          acc.push({
            dimension_type: condition.dimension_type,
            dimension_values: [value],
          })
          return acc
        }, []),
    }))
}

export function fromUiToApiRuleGroups(groups: TrafficScopeRuleGroupForm[]): TrafficScopeRuleGroup[] {
  return groups
    .map((group) => ({
      rule_type: group.rule_type,
      conditions: (group.conditions || [])
        .flatMap((condition) =>
          [...new Set((condition.dimension_values || []).map((value) => value.trim()).filter(Boolean))]
            .map((dimensionValue) => ({
              dimension_type: condition.dimension_type,
              dimension_value: dimensionValue,
            })),
        ),
    }))
    .filter((group) => group.conditions.length > 0)
}

export const normalizeTrafficScopeRuleGroups = fromUiToApiRuleGroups
