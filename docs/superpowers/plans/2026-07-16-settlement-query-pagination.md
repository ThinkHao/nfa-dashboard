# 院校结算查询分页修复 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复院校结算查询明细视图把当前页行数误当总数的问题，并提供可复用的分页响应规范化函数。

**Architecture:** 在 `src/utils/pagination.ts` 中集中兼容标准 `{ items, total }` 与历史数组响应。院校结算页面仍保留当前页月份裁剪，但分页器总数始终采用规范化后的服务端总数。

**Tech Stack:** Vue 3、TypeScript、Vitest、Element Plus

---

### Task 1: 分页响应规范化函数（TDD）

**Files:**
- Create: `frontend/frontend/src/utils/pagination.ts`
- Create: `frontend/frontend/src/utils/__tests__/pagination.spec.ts`

- [ ] **Step 1: 写失败测试**

```ts
import { describe, expect, it } from 'vitest'
import { normalizePaginatedResponse } from '@/utils/pagination'

describe('normalizePaginatedResponse', () => {
  it('preserves the server total instead of using the current page length', () => {
    const items = Array.from({ length: 10 }, (_, index) => ({ id: index + 1 }))
    expect(normalizePaginatedResponse({ items, total: 67 })).toEqual({ items, total: 67 })
  })

  it('supports legacy array responses', () => {
    const items = [{ id: 1 }, { id: 2 }]
    expect(normalizePaginatedResponse(items)).toEqual({ items, total: 2 })
  })

  it('falls back to page length for an invalid total', () => {
    const items = [{ id: 1 }]
    expect(normalizePaginatedResponse({ items, total: 'invalid' })).toEqual({ items, total: 1 })
  })

  it('returns an empty page for unsupported responses', () => {
    expect(normalizePaginatedResponse(null)).toEqual({ items: [], total: 0 })
  })
})
```

- [ ] **Step 2: 运行测试并确认失败**

Run: `npm run test:unit -- src/utils/__tests__/pagination.spec.ts --run`
Expected: FAIL，无法解析 `@/utils/pagination`。

- [ ] **Step 3: 写最小实现**

```ts
export type NormalizedPaginatedResponse<T> = {
  items: T[]
  total: number
}

export function normalizePaginatedResponse<T>(response: unknown): NormalizedPaginatedResponse<T> {
  if (Array.isArray(response)) return { items: response as T[], total: response.length }
  if (response && typeof response === 'object') {
    const candidate = response as { items?: unknown; total?: unknown }
    if (Array.isArray(candidate.items)) {
      const total = Number(candidate.total)
      return {
        items: candidate.items as T[],
        total: Number.isFinite(total) && total >= 0 ? total : candidate.items.length,
      }
    }
  }
  return { items: [], total: 0 }
}
```

- [ ] **Step 4: 运行测试并确认通过**

Run: `npm run test:unit -- src/utils/__tests__/pagination.spec.ts --run`
Expected: 4 tests PASS。

- [ ] **Step 5: 提交**

```bash
git add frontend/frontend/src/utils/pagination.ts frontend/frontend/src/utils/__tests__/pagination.spec.ts
git commit -m "test(web): 覆盖分页响应总数解析"
```

### Task 2: 修复院校结算明细分页

**Files:**
- Modify: `frontend/frontend/src/views/SettlementUserQueryView.vue`

- [ ] **Step 1: 引入公共解析器**

```ts
import { normalizePaginatedResponse } from '@/utils/pagination'
```

- [ ] **Step 2: 替换 `fetchRows` 的响应分支**

```ts
    const page = normalizePaginatedResponse<any>(res)
    const clipped = clipRowsBySelectedMonths(page.items)
    rows.value = await enrichRowsWithStartDates(clipped, signal)
    pagination.total = page.total
```

删除原先数组/对象两个分支及两处 `pagination.total = clipped.length`。

- [ ] **Step 3: 运行相关测试与类型检查**

Run: `npm run test:unit -- src/utils/__tests__/pagination.spec.ts src/views/__tests__/settlement-query-filter-utils.spec.ts src/views/__tests__/settlement-user-query-utils.spec.ts --run`
Expected: 19 tests PASS。

Run: `npm run type-check`
Expected: exit 0。

- [ ] **Step 4: 提交**

```bash
git add frontend/frontend/src/views/SettlementUserQueryView.vue
git commit -m "fix(web): 保留院校结算分页总数"
```

### Task 3: 完整验证与审计复核

**Files:**
- Verify only

- [ ] **Step 1: 确认问题代码已消失**

Run: `rg -n "pagination\.total = clipped\.length" frontend/frontend/src/views/SettlementUserQueryView.vue`
Expected: 无匹配。

- [ ] **Step 2: 运行全量前端测试**

Run: `$env:TZ='UTC'; npm run test:unit -- --run`
Expected: 29 个测试文件、106 项测试全部通过（现有 102 项 + 新增 4 项）。

- [ ] **Step 3: 运行类型检查与生产构建**

Run: `npm run type-check`
Expected: exit 0。

Run: `npm run build`
Expected: exit 0；既有大 chunk 警告不视为失败。

- [ ] **Step 4: 检查差异范围**

Run: `git diff --check` 与 `git status --short`
Expected: 无 whitespace error；仅包含计划内文件。
