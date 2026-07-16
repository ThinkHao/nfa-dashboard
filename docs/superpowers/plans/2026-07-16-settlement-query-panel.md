# 院校结算查询面板 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将“单用户结算查询”改为服务月份必选、用户/地区/CP/学校均可选的“院校结算查询”，并保持现有金额与按月列展示口径。

**Architecture:** 复用现有结算 API，仅提取前端查询过滤纯函数统一生成参数和校验月份。页面、翻页、按月列和导出继续走同一参数生成函数；后端与数据库不改。

**Tech Stack:** Vue 3、TypeScript、Vitest、Element Plus、Vite

---

### Task 1: 查询过滤纯函数（TDD）

**Files:**
- Create: `frontend/frontend/src/views/settlement-query-filter-utils.ts`
- Create: `frontend/frontend/src/views/__tests__/settlement-query-filter-utils.spec.ts`

- [ ] **Step 1: 写失败测试**

测试 `buildSettlementQueryParams`：无用户时省略 `channel_owner_user_id`，有用户时带正确 ID；地区、CP、学校仅在非空时发送；月份转换为起止时间。测试 `validateSettlementQueryRange`：月份为空失败，超过 12 个月失败，有效月份成功，且不依赖用户。

- [ ] **Step 2: 运行测试并确认因模块不存在而失败**

Run: `npm run test:unit -- src/views/__tests__/settlement-query-filter-utils.spec.ts`
Expected: FAIL，无法解析 `settlement-query-filter-utils`。

- [ ] **Step 3: 写最小实现**

```ts
export type SettlementQueryFilters = {
  userId: number | null
  region: string
  cp: string
  schoolName: string
}

import { resolveMonthRangeDateTime } from './settlement-user-query-utils'

export function buildSettlementQueryParams(
  filters: SettlementQueryFilters,
  monthRange: [string, string] | null,
  page: number,
  pageSize: number,
): Record<string, string | number> {
  const params: Record<string, string | number> = { page, page_size: pageSize }
  const { start, end } = resolveMonthRangeDateTime(monthRange)
  if (start) params.start_service_date = start
  if (end) params.end_service_date = end
  if (filters.userId != null && filters.userId > 0) params.channel_owner_user_id = filters.userId
  if (filters.region) params.region = filters.region
  if (filters.cp) params.cp = filters.cp
  if (filters.schoolName) params.school_name = filters.schoolName
  return params
}

export function validateSettlementQueryRange(monthRange: [string, string] | null): string | null {
  if (!monthRange?.[0] || !monthRange?.[1]) return '请先选择服务月份范围'
  const [startYear, startMonth] = monthRange[0].split('-').map(Number)
  const [endYear, endMonth] = monthRange[1].split('-').map(Number)
  const months = (endYear - startYear) * 12 + endMonth - startMonth + 1
  if (months > 12) return '查询时间跨度最多 12 个月'
  return null
}
```

- [ ] **Step 4: 运行测试并确认通过**

Run: `npm run test:unit -- src/views/__tests__/settlement-query-filter-utils.spec.ts`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add frontend/frontend/src/views/settlement-query-filter-utils.ts frontend/frontend/src/views/__tests__/settlement-query-filter-utils.spec.ts
git commit -m "test(web): 覆盖院校结算可选筛选"
```

### Task 2: 页面接入可选用户筛选（TDD）

**Files:**
- Modify: `frontend/frontend/src/views/SettlementUserQueryView.vue`

- [ ] **Step 1: 将页面参数构造接到已通过测试的纯函数**

`buildParams(page, pageSize)` 调用 `buildSettlementQueryParams(filter, monthRange.value, page, pageSize)`。用户为空时请求中不再出现 `channel_owner_user_id`。

- [ ] **Step 2: 删除用户必选 UI 与校验**

移除用户表单项的 `required`；placeholder 改为“全部用户”。`validateBeforeQuery` 只消费 `validateSettlementQueryRange` 的结果并展示提示，不再出现“请先选择用户”。

- [ ] **Step 3: 运行过滤测试与既有按月列测试**

Run: `npm run test:unit -- src/views/__tests__/settlement-query-filter-utils.spec.ts src/views/__tests__/settlement-user-query-utils.spec.ts`
Expected: PASS，且既有按月列金额结构测试无回归。

- [ ] **Step 4: 提交**

```bash
git add frontend/frontend/src/views/SettlementUserQueryView.vue
git commit -m "feat(web): 院校结算支持可选用户筛选"
```

### Task 3: 同步用户可见命名

**Files:**
- Modify: `frontend/frontend/src/views/SettlementUserQueryView.vue`
- Modify: `frontend/frontend/src/router/index.ts`
- Modify: `frontend/frontend/src/views/home-workbench.ts`
- Modify: `frontend/frontend/src/views/SystemSettingsView.vue`
- Modify: `frontend/frontend/src/components/settlement/SettlementDataTab.vue`

- [ ] **Step 1: 更新页面、菜单与工作台名称**

把用户可见“单用户结算查询”改为“院校结算查询”。

- [ ] **Step 2: 更新系统设置说明与相关代码注释**

设置标题改为“院校结算查询页 95 值单位”，说明中的页面名称同步更新；相关基准注释同步改名。文件名、路由路径/name、导出前缀均保持不变。

- [ ] **Step 3: 静态扫描残留用户可见旧名称**

Run: `rg -n "单用户结算查询|单用户结算页" frontend/frontend/src`
Expected: 无匹配。

- [ ] **Step 4: 提交**

```bash
git add frontend/frontend/src/views/SettlementUserQueryView.vue frontend/frontend/src/router/index.ts frontend/frontend/src/views/home-workbench.ts frontend/frontend/src/views/SystemSettingsView.vue frontend/frontend/src/components/settlement/SettlementDataTab.vue
git commit -m "feat(web): 重命名院校结算查询面板"
```

### Task 4: 完整验证

**Files:**
- Verify only

- [ ] **Step 1: 运行针对性单测**

Run: `npm run test:unit -- src/views/__tests__/settlement-query-filter-utils.spec.ts src/views/__tests__/settlement-user-query-utils.spec.ts`
Expected: 全部通过。

- [ ] **Step 2: 运行前端全量单测**

Run: `npm run test:unit -- --run`
Expected: 全部通过，无失败测试。

- [ ] **Step 3: 运行类型检查**

Run: `npm run type-check`
Expected: exit 0。

- [ ] **Step 4: 运行生产构建**

Run: `npm run build`
Expected: exit 0；现有 Vite 大 chunk 警告可记录但不视为失败。

- [ ] **Step 5: 检查差异范围**

Run: `git diff HEAD~3 --check` 及 `git status --short`
Expected: 无 whitespace error；无意外文件。
