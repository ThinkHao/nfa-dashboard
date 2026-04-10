import { describe, expect, it } from 'vitest'

import { createEmptyTrafficScopeRuleGroup, normalizeTrafficScopeRuleGroups } from '../traffic-scope-rule-groups'

describe('traffic-scope-rule-groups', () => {
  it('creates an empty rule group with one default condition', () => {
    expect(createEmptyTrafficScopeRuleGroup()).toEqual({
      rule_type: 'allow',
      conditions: [
        {
          dimension_type: 'region',
          dimension_value: '',
        },
      ],
    })
  })

  it('normalizes nested groups into groups[].conditions[] payload', () => {
    const payload = normalizeTrafficScopeRuleGroups([
      {
        rule_type: 'allow',
        conditions: [
          { dimension_type: 'region', dimension_value: ' 华东 ' },
          { dimension_type: 'cp', dimension_value: 'CMCC' },
          { dimension_type: 'school', dimension_value: '   ' },
        ],
      },
      {
        rule_type: 'deny',
        conditions: [
          { dimension_type: 'school', dimension_value: 'school-x' },
        ],
      },
      {
        rule_type: 'allow',
        conditions: [
          { dimension_type: 'region', dimension_value: '   ' },
        ],
      },
    ])

    expect(payload).toEqual([
      {
        rule_type: 'allow',
        conditions: [
          { dimension_type: 'region', dimension_value: '华东' },
          { dimension_type: 'cp', dimension_value: 'CMCC' },
        ],
      },
      {
        rule_type: 'deny',
        conditions: [
          { dimension_type: 'school', dimension_value: 'school-x' },
        ],
      },
    ])
  })
})
