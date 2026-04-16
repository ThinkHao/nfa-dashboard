import type { CreatedImportUser, CustomerRateImportResponse, MissingImportUser } from '@/types/api'

export const CUSTOMER_RATE_IMPORT_PENDING_KEY = 'customerRateImport.pending.v1'
export const CUSTOMER_RATE_IMPORT_PENDING_MAX_AGE_MS = 24 * 60 * 60 * 1000
export const CUSTOMER_RATE_IMPORT_PENDING_MAX_ITEMS = 20

export interface PendingImportTaskEntry {
  taskId: number;
  createdAt: number;
}

export function shouldPromptAutoCreateUsers(res: CustomerRateImportResponse): boolean {
  return res.stage === 'needs_user_creation'
    && !!res.can_auto_create_users
    && !!res.resumable_token
    && Array.isArray(res.missing_users)
    && res.missing_users.length > 0
}

export function buildMissingUsersPreview(users: MissingImportUser[], limit = 10): string {
  return users
    .slice(0, limit)
    .map((item) => {
      const fields = Array.isArray(item.fields) && item.fields.length > 0 ? `字段：${item.fields.join('、')}` : '字段：未知'
      const lines = Array.isArray(item.lines) && item.lines.length > 0 ? `行号：${item.lines.join('、')}` : '行号：未知'
      return `${item.alias} -> ${item.suggested_username}（${fields}；${lines}）`
    })
    .join('\n')
}

export function buildCreatedUsersExportRows(users: CreatedImportUser[]): Array<[string, string, string]> {
  return users.map((item) => [item.alias, item.username, item.password])
}

export function normalizePendingImportTasks(
  value: unknown,
  now = Date.now(),
  maxAgeMs = CUSTOMER_RATE_IMPORT_PENDING_MAX_AGE_MS,
  maxItems = CUSTOMER_RATE_IMPORT_PENDING_MAX_ITEMS,
): PendingImportTaskEntry[] {
  const arr = Array.isArray(value) ? value : []
  const cutoff = now - Math.max(0, maxAgeMs)
  const byTaskId = new Map<number, PendingImportTaskEntry>()
  for (const item of arr) {
    const taskId = Number((item as any)?.taskId)
    const createdAt = Number((item as any)?.createdAt)
    if (!Number.isInteger(taskId) || taskId <= 0) continue
    if (!Number.isFinite(createdAt)) continue
    if (createdAt < cutoff) continue
    const prev = byTaskId.get(taskId)
    if (!prev || createdAt > prev.createdAt) {
      byTaskId.set(taskId, { taskId, createdAt })
    }
  }
  return Array.from(byTaskId.values())
    .sort((a, b) => b.createdAt - a.createdAt)
    .slice(0, Math.max(1, maxItems))
}

export function upsertPendingImportTask(
  items: PendingImportTaskEntry[],
  taskId: number,
  now = Date.now(),
): PendingImportTaskEntry[] {
  const next = normalizePendingImportTasks(items, now)
    .filter((item) => item.taskId !== taskId)
  next.unshift({ taskId, createdAt: now })
  return normalizePendingImportTasks(next, now)
}

export function removePendingImportTask(
  items: PendingImportTaskEntry[],
  taskId: number,
  now = Date.now(),
): PendingImportTaskEntry[] {
  return normalizePendingImportTasks(items, now).filter((item) => item.taskId !== taskId)
}
