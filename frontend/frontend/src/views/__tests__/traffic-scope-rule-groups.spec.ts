import { describe, expect, it } from 'vitest'

import {
  createEmptyTrafficScopeRuleGroup,
  fromApiToUiRuleGroups,
  fromUiToApiRuleGroups,
} from '../traffic-scope-rule-groups'

describe('traffic-scope-rule-groups', () => {
  it('creates an empty rule group with one default condition', () => {
    expect(createEmptyTrafficScopeRuleGroup()).toEqual({
      rule_type: 'allow',
      conditions: [
        {
          dimension_type: 'region',
          dimension_values: [],
        },
      ],
    })
  })

  it('merges same-dimension api conditions to multi-select ui conditions', () => {
    const uiGroups = fromApiToUiRuleGroups([
      {
        id: 11,
        user_id: 101,
        rule_type: 'allow',
        conditions: [
          { dimension_type: 'region', dimension_value: ' 华东 ' },
          { dimension_type: 'region', dimension_value: '华北' },
          { dimension_type: 'region', dimension_value: '华东' },
          { dimension_type: 'cp', dimension_value: 'CMCC' },
          { dimension_type: 'school', dimension_value: '   ' },
        ],
      },
    ])

    expect(uiGroups).toEqual([
      {
        id: 11,
        user_id: 101,
        rule_type: 'allow',
        conditions: [
          { dimension_type: 'region', dimension_values: ['华东', '华北'] },
          { dimension_type: 'cp', dimension_values: ['CMCC'] },
        ],
      },
    ])
  })

  it('expands multi-select ui conditions into api payload and drops empty values', () => {
    const payload = fromUiToApiRuleGroups([
      {
        rule_type: 'allow',
        conditions: [
          { dimension_type: 'region', dimension_values: [' 华东 ', '华北', '华东', '  '] },
          { dimension_type: 'cp', dimension_values: ['CMCC'] },
          { dimension_type: 'school', dimension_values: [] },
        ],
      },
      {
        rule_type: 'deny',
        conditions: [
          { dimension_type: 'school', dimension_values: ['school-x'] },
        ],
      },
      {
        rule_type: 'allow',
        conditions: [
          { dimension_type: 'region', dimension_values: ['   '] },
        ],
      },
    ])

    expect(payload).toEqual([
      {
        rule_type: 'allow',
        conditions: [
          { dimension_type: 'region', dimension_value: '华东' },
          { dimension_type: 'region', dimension_value: '华北' },
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
