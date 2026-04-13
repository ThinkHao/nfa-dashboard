import type { CreatedImportUser, CustomerRateImportResponse, MissingImportUser } from '@/types/api'

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
