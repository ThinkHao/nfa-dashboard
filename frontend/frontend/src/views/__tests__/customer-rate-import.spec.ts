import { describe, expect, it } from 'vitest'

import { buildCreatedUsersExportRows, buildMissingUsersPreview, shouldPromptAutoCreateUsers } from '../customer-rate-import'

describe('customer-rate-import helpers', () => {
  it('detects when import response should prompt auto-create', () => {
    expect(shouldPromptAutoCreateUsers({
      affected: 0,
      errors: [],
      stage: 'needs_user_creation',
      can_auto_create_users: true,
      resumable_token: 'token-1',
      missing_users: [{ alias: '陈金荣', suggested_username: 'chenjr' }],
    })).toBe(true)
  })

  it('builds readable preview for missing users', () => {
    expect(buildMissingUsersPreview([
      { alias: '陈金荣', suggested_username: 'chenjr', fields: ['客户费归属'], lines: [2, 4] },
    ])).toContain('陈金荣 -> chenjr')
  })

  it('builds export rows for created users', () => {
    expect(buildCreatedUsersExportRows([
      { alias: '陈金荣', username: 'chenjr', password: 'Pwd123456' },
    ])).toEqual([
      ['陈金荣', 'chenjr', 'Pwd123456'],
    ])
  })
})
