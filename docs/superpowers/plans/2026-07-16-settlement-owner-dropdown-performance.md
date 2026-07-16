# 院校结算用户下拉性能优化 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 保持月份/地区/CP 范围语义，将用户下拉从全量结算行读取改为数据库去重 ID 查询，并提前并行预加载。

**Architecture:** 仓库层在当前活动槽位和既有过滤条件上，将四个 owner 字段展开并 `DISTINCT`，服务层只按这些 ID 查询用户名称。前端把三个互不依赖的初始化加载放入 `Promise.all`，不引入缓存或新接口。

**Tech Stack:** Go、GORM、MySQL 5.7、Vue 3、TypeScript、Vitest

---

### Task 1: 服务改用轻量 owner ID 仓库方法（TDD）

**Files:**
- Modify: `backend/internal/repository/settlement_data_repository.go`
- Modify: `backend/internal/service/settlement_data_service.go`
- Create: `backend/internal/service/settlement_data_owner_subjects_test.go`

- [ ] **Step 1: 写失败服务测试**

测试 stub 通过嵌入现有接口满足无关方法，并记录新方法收到的 filter；用户仓库 stub 记录 `FindByIDs` 参数。断言：服务传入地区、CP、学校和日期；只查询仓库返回的 `[7, 9]`；显示名使用现有 alias 规则；空 ID 时不调用用户仓库。

```go
type ownerIDsRepoStub struct {
    repository.SettlementDataRepository
    ids []uint64
    filter map[string]interface{}
}

func (s *ownerIDsRepoStub) ListDistinctOwnerUserIDs(_ context.Context, filter map[string]interface{}) ([]uint64, error) {
    s.filter = filter
    return s.ids, nil
}
```

- [ ] **Step 2: 运行测试并确认走到旧全量路径而失败**

Run: `go test ./internal/service -run TestListUsedOwnerSubjectsUsesDistinctOwnerIDs -v`
Expected: FAIL；当前服务调用嵌入 stub 的旧 `ListSettlementCustomer` 路径，而不是测试提供的轻量 ID 方法。

- [ ] **Step 3: 扩展仓库接口、实现单次扫描查询并接线服务**

接口增加：

```go
ListDistinctOwnerUserIDs(ctx context.Context, filter map[string]interface{}) ([]uint64, error)
```

`ListUsedOwnerSubjects` 构造与 `ListAll` 相同的 filter map，然后调用新方法。ID 为空直接返回 `[]UsedOwnerSubject{}`；非空调用 `userRepo.FindByIDs(ids)` 并沿用 `displayUserName`。

使用 `settlement_customer`；若 slot 表可用则使用 `settlement_customer_v scv` 并调用 `withActiveSlot`。然后调用 `applySettlementCustomerFilters(qb, filter, alias)`。

```go
ownerExpr := fmt.Sprintf(`CASE owner_slots.n
    WHEN 1 THEN %s.customer_fee_owner_id
    WHEN 2 THEN %s.network_line_fee_owner_id
    WHEN 3 THEN %s.node_deduction_fee_owner_id
    ELSE %s.channel_owner_user_id
END`, alias, alias, alias, alias)

var rows []struct { OwnerID uint64 `gorm:"column:owner_id"` }
err := qb.
    Joins("JOIN (SELECT 1 AS n UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4) owner_slots ON 1=1").
    Select("DISTINCT " + ownerExpr + " AS owner_id").
    Where("(" + ownerExpr + ") IS NOT NULL AND (" + ownerExpr + ") > 0").
    Order("owner_id ASC").
    Scan(&rows).Error
```

将 rows 转为 `[]uint64` 返回。

- [ ] **Step 4: 运行服务回归测试并确认通过**

Run: `go test ./internal/service -run TestListUsedOwnerSubjectsUsesDistinctOwnerIDs -v`
Expected: PASS。

Run: `go test ./internal/repository ./internal/service`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add backend/internal/repository/settlement_data_repository.go backend/internal/service/settlement_data_service.go backend/internal/service/settlement_data_owner_subjects_test.go
git commit -m "perf(settlement): 数据库去重归属用户 ID"
```

### Task 2: 前端初始化并行加载（TDD）

**Files:**
- Create: `frontend/frontend/src/views/settlement-query-initial-loads.ts`
- Create: `frontend/frontend/src/views/__tests__/settlement-query-initial-loads.spec.ts`
- Modify: `frontend/frontend/src/views/SettlementUserQueryView.vue`

- [ ] **Step 1: 写失败测试**

三个 loader 各返回受控 Promise。调用 `runSettlementQueryInitialLoads` 后、释放任何 Promise 前，断言三个 loader 都已调用；随后释放并等待完成。

- [ ] **Step 2: 运行测试并确认模块不存在**

Run: `npm run test:unit -- src/views/__tests__/settlement-query-initial-loads.spec.ts --run`
Expected: FAIL，无法解析模块。

- [ ] **Step 3: 实现最小并行编排并接入页面**

```ts
export function runSettlementQueryInitialLoads(loaders: Array<() => Promise<unknown>>): Promise<unknown[]> {
  return Promise.all(loaders.map((load) => load()))
}
```

页面 `onMounted` 在 `trafficSettings.ensureLoaded()` 和 `setDefaultMonthRange()` 后调用：

```ts
await runSettlementQueryInitialLoads([loadRegionCpSchool, loadKeySchoolSet, loadOwnerUsers])
```

- [ ] **Step 4: 运行前端相关测试和类型检查**

Run: `npm run test:unit -- src/views/__tests__/settlement-query-initial-loads.spec.ts src/views/__tests__/settlement-query-filter-utils.spec.ts src/views/__tests__/settlement-user-query-utils.spec.ts --run`
Expected: PASS。

Run: `npm run type-check`
Expected: exit 0。

- [ ] **Step 5: 提交**

```bash
git add frontend/frontend/src/views/settlement-query-initial-loads.ts frontend/frontend/src/views/__tests__/settlement-query-initial-loads.spec.ts frontend/frontend/src/views/SettlementUserQueryView.vue
git commit -m "perf(web): 并行预加载结算用户选项"
```

### Task 3: 同口径性能与完整验证

**Files:**
- Verify only

- [ ] **Step 1: 对同一数据库范围比对 ID 集合**

使用只读脚本分别从旧完整记录和新 SQL 提取 `2026-05-01` 至 `2026-08-01` 的 owner ID，断言集合完全相等且均为 25 个。

- [ ] **Step 2: 复测数据库耗时**

同一连接预热后执行新查询至少 5 次，记录中位数。Expected: 不高于 400 ms。

- [ ] **Step 3: 后端完整验证**

Run: `go test ./internal/repository ./internal/service`
Expected: PASS。

Run: `go test ./...`
Expected: PASS。

- [ ] **Step 4: 前端完整验证**

Run: `$env:TZ='UTC'; npm run test:unit -- --run`
Expected: 全部 PASS。

Run: `npm run type-check` 与 `npm run build`
Expected: exit 0；既有大 chunk 警告不视为失败。

- [ ] **Step 5: 检查差异**

Run: `git diff --check` 与 `git status --short`
Expected: 无 whitespace error；仅计划内文件。
