import { describe, expect, it } from 'vitest'

import {
  buildCreatedUsersExportRows,
  buildMissingUsersPreview,
  normalizePendingImportTasks,
  removePendingImportTask,
  shouldPromptAutoCreateUsers,
  upsertPendingImportTask,
} from '../customer-rate-import'

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

  it('normalizes pending tasks by removing invalid/expired and de-duplicating', () => {
    const now = 1_700_000_000_000
    const day = 24 * 60 * 60 * 1000
    const tasks = normalizePendingImportTasks([
      { taskId: 101, createdAt: now - 1_000 },
      { taskId: 101, createdAt: now - 500 }, // duplicate keep latest
      { taskId: -1, createdAt: now - 2_000 }, // invalid id
      { taskId: 202, createdAt: now - day - 1 }, // expired
      { taskId: 303, createdAt: Number.NaN }, // invalid time
      { taskId: 404, createdAt: now - 2_000 },
    ], now)

    expect(tasks).toEqual([
      { taskId: 101, createdAt: now - 500 },
      { taskId: 404, createdAt: now - 2_000 },
    ])
  })

  it('upserts pending task at head and keeps max 20 items', () => {
    const now = 1_700_000_000_000
    const seed = Array.from({ length: 25 }).map((_, i) => ({ taskId: i + 1, createdAt: now - i }))
    const tasks = upsertPendingImportTask(seed, 7, now + 1000)
    expect(tasks[0]).toEqual({ taskId: 7, createdAt: now + 1000 })
    expect(tasks).toHaveLength(20)
    expect(tasks.filter(x => x.taskId === 7)).toHaveLength(1)
  })

  it('removes pending task by id', () => {
    const now = 1_700_000_000_000
    const tasks = removePendingImportTask([
      { taskId: 11, createdAt: now - 1_000 },
      { taskId: 22, createdAt: now - 2_000 },
    ], 11, now)

    expect(tasks).toEqual([{ taskId: 22, createdAt: now - 2_000 }])
  })
})
