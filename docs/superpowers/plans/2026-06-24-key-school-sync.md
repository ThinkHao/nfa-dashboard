# 重点院校（is_key_school）字段双向同步 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把「重点院校」标记从源库 `nfa_ipgroup.check_status`（OR 聚合）同步到 `nfa_school.is_key_school`，并在 dashboard 院校列表（含筛选）与 NFA 结算视图中露出。

**Architecture:** 三阶段、跨两仓库。Phase A 改 `nfa-extractor`（Python）：聚合 SQL 增列、字段进 `data_hash`、INSERT/UPDATE 同步写列（哈希变化触发一次性全表自动回填）。Phase B 改 `nfa-dashboard` 后端（Go：迁移 + 模型 + 仓库/服务/控制器筛选）。Phase C 改 `nfa-dashboard` 前端（Vue：列表列+筛选、结算视图按 school_id 标注）。

**Tech Stack:** Python + pandas + SQLAlchemy（extractor）；Go + Gin + GORM + MySQL（后端）；Vue 3 + TS + Element Plus + Vitest（前端）。

**关键约束（来自 spec §2）：** 凡进 `data_hash` 的字段，必须同时进 INSERT/UPDATE 的列清单，否则 `is_key_school` 会被永久锁死在 0 且无报错。部署顺序：先 dashboard 迁移 → 再 extractor 上线（触发回填）→ 再 dashboard 后端/前端。

**前置核对（实现前先确认）：** 源库 `nfa_ipgroup` 确有 `check_status` 列（0/1）。若列名/取值不同，调整 Phase A Task A2 的 SQL。

参考 spec：`docs/superpowers/specs/2026-06-24-key-school-sync-design.md`

---

## 文件结构

**nfa-extractor 仓库**（`C:\Users\haoji\Desktop\Code\nfa-extractor`）
- Modify: `web/nfa_extractor.py` — 聚合查询、`_prepare_school_data`、`_batch_update_school_info`、`_update_school_info`
- Create: `tests/test_is_key_school.py` — DB-free 单元测试（`_prepare_school_data` 的字段+哈希保证）
- Modify: `test_aggregation.py` — 集成脚本增加 is_key_school 列校验输出

**nfa-dashboard 仓库**（`C:\Users\haoji\Desktop\Code\nfa-dashboard`）
- Create: `sql/migrations/048_add-nfa-school-is-key-school.sql`
- Modify: `sql/nfa_school.sql`, `sql/dist/install_full.sql` — 建表定义同步
- Modify: `backend/internal/model/school.go` — `IsKeySchool` 字段
- Modify: `backend/internal/repository/school_repository.go` — filter 支持 `is_key_school`
- Modify: `backend/internal/service/school_service.go` — 透传参数
- Modify: `backend/internal/controller/school_controller.go` — 解析 query + 纯函数 `normalizeIsKeySchoolFilter`
- Create: `backend/internal/controller/school_controller_is_key_school_test.go`
- Create: `frontend/frontend/src/views/key-school-utils.ts` — `buildKeySchoolSet` / `isKeySchool`
- Create: `frontend/frontend/src/views/__tests__/key-school-utils.spec.ts`
- Modify: `frontend/frontend/src/views/SchoolsView.vue` — 列 + 筛选
- Modify: `frontend/frontend/src/components/settlement/SettlementDataTab.vue` 与 `frontend/frontend/src/views/SettlementUserQueryView.vue` — 结算标注

> **测试模式说明（重要）：** extractor 仓库无纯单元测试框架（现有 `test_aggregation.py` 是需要真实 DB 的手动脚本）。因此 Phase A 对**纯逻辑**（字段进哈希）用 DB-free 单元测试（`object.__new__` 绕过 DB 连接），对 **SQL/写库**（DB-bound）用集成脚本验证并给出可执行命令——这是为匹配该仓库既有测试风格，不强行编造伪单测。dashboard 后端沿用既有纯函数单测风格；前端沿用 util 化 vitest 风格。

---

## Phase A — nfa-extractor

> 所有命令在 `C:\Users\haoji\Desktop\Code\nfa-extractor` 下运行。

### Task A1: `_prepare_school_data` 纳入 is_key_school 并进哈希（DB-free 单测）

**Files:**
- Create: `tests/test_is_key_school.py`
- Modify: `web/nfa_extractor.py:271-286`（`_prepare_school_data`）

- [ ] **Step 1: Write the failing test**

Create `tests/test_is_key_school.py`:

```python
import os
import sys
import pandas as pd

sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'web'))
from nfa_extractor import NFAExtractor


class _StubRuleEngine:
    def get_primary_hash_uuid(self, cp, hash_uuids):
        return hash_uuids[0]


def _make_extractor_without_db():
    """绕过 __init__（会连数据库），只装上哈希计算需要的 rule_engine。"""
    ext = object.__new__(NFAExtractor)
    ext.rule_engine = _StubRuleEngine()
    return ext


def _base_series(is_key_school):
    return pd.Series({
        'school_id': 's1',
        'school_name': '示例学校',
        'region': '华北',
        'cp': '电信',
        'hash_uuids': 'a,b',
        'hash_count': 2,
        'is_key_school': is_key_school,
    })


def test_prepare_school_data_includes_is_key_school():
    ext = _make_extractor_without_db()
    data = ext._prepare_school_data(_base_series(1))
    assert data['is_key_school'] == 1


def test_is_key_school_flip_changes_data_hash():
    ext = _make_extractor_without_db()
    h1 = ext._prepare_school_data(_base_series(1))['data_hash']
    h0 = ext._prepare_school_data(_base_series(0))['data_hash']
    assert h1 != h0, "is_key_school 翻转必须改变 data_hash，否则同步不会触发"
```

- [ ] **Step 2: Run test to verify it fails**

Run: `python -m pytest tests/test_is_key_school.py -v`
Expected: FAIL（`KeyError: 'is_key_school'` 或 `data['is_key_school']` 不存在 / 两个哈希相等）。

- [ ] **Step 3: Write minimal implementation**

In `web/nfa_extractor.py`, `_prepare_school_data`, 在构造 `school_data` 字典时（`'hash_count': ...` 之后、`data_hash` 计算之前）加一行：

```python
        school_data = {
            'school_id': str(school_info['school_id']),
            'school_name': str(school_info['school_name']),
            'region': str(school_info['region']),
            'cp': str(school_info['cp']),
            'hash_uuids': str(school_info['hash_uuids']),
            'primary_hash_uuid': self.rule_engine.get_primary_hash_uuid(
                cp=str(school_info['cp']),
                hash_uuids=str(school_info['hash_uuids']).split(',')
            ),
            'hash_count': int(school_info['hash_count']),
            'is_key_school': int(school_info['is_key_school']),  # ← 新增；自动进 data_hash
        }
        school_data['data_hash'] = self._calculate_data_hash(school_data)
        return school_data
```

- [ ] **Step 4: Run test to verify it passes**

Run: `python -m pytest tests/test_is_key_school.py -v`
Expected: PASS（2 passed）。

- [ ] **Step 5: Commit**

```bash
git add tests/test_is_key_school.py web/nfa_extractor.py
git commit -m "feat(extractor): _prepare_school_data 纳入 is_key_school 并进 data_hash"
```

---

### Task A2: 聚合 SQL 增加 OR 口径的 is_key_school 列

**Files:**
- Modify: `web/nfa_extractor.py:134-154`（`get_aggregated_school_info` 主查询）
- Modify: `test_aggregation.py`（集成校验输出）

- [ ] **Step 1: 修改聚合查询**

在主查询 SELECT 列表中、`hash_count` 行之后增加 `is_key_school` 聚合列：

```python
            query = """
            SELECT 
                school_id,
                school_name,
                region,
                cp,
                GROUP_CONCAT(DISTINCT hash_uuid) as hash_uuids,
                SUM(CASE WHEN hash_uuid IS NOT NULL THEN 1 ELSE 0 END) as hash_count,
                MAX(CASE WHEN check_status = 1 THEN 1 ELSE 0 END) as is_key_school
            FROM nfa_ipgroup
            WHERE hash_uuid IS NOT NULL
                AND school_id IS NOT NULL 
                AND school_id != ''
                AND school_name IS NOT NULL 
                AND school_name != ''
                AND region IS NOT NULL 
                AND region != ''
                AND cp IS NOT NULL 
                AND cp != ''
            GROUP BY school_id, region, cp
            HAVING hash_count > 0
            """
```

> 用 `MAX(CASE WHEN check_status = 1 THEN 1 ELSE 0 END)` 实现 OR：组内任一 `check_status=1` → 1，全 0 / NULL → 0。

- [ ] **Step 2: 扩展集成校验脚本**

在 `test_aggregation.py` 的 `test_aggregation()` 里，`print(df.dtypes)` 之后增加：

```python
        # 校验 is_key_school 列存在且取值合法
        assert 'is_key_school' in df.columns, "聚合结果缺少 is_key_school 列"
        bad = df[~df['is_key_school'].isin([0, 1])]
        assert bad.empty, f"is_key_school 出现非 0/1 取值: {bad['is_key_school'].unique()}"
        print("\n=== 重点院校统计 ===")
        print(df['is_key_school'].value_counts())
```

- [ ] **Step 3: 集成验证（需连到含 nfa_ipgroup 的源库）**

Run: `python test_aggregation.py`
Expected: 打印「重点院校统计」，断言不触发；`is_key_school` 仅含 0/1。
（无源库环境时，此步在联调/预发环境执行；先 commit 代码。）

- [ ] **Step 4: Commit**

```bash
git add web/nfa_extractor.py test_aggregation.py
git commit -m "feat(extractor): 聚合查询按 OR 口径输出 is_key_school"
```

---

### Task A3: INSERT/UPDATE 写库语句同步写入 is_key_school（约束 2）

**Files:**
- Modify: `web/nfa_extractor.py:322-348`（`_batch_update_school_info` 的 UPDATE 与 INSERT）
- Modify: `web/nfa_extractor.py:371,377`（`_update_school_info` 的 UPDATE 与 INSERT）

- [ ] **Step 1: 改 `_batch_update_school_info` 的 UPDATE**

```python
                        update_query = """
                        UPDATE nfa_school
                        SET school_name = :school_name,
                            hash_uuids = :hash_uuids,
                            primary_hash_uuid = :primary_hash_uuid,
                            hash_count = :hash_count,
                            is_key_school = :is_key_school,
                            data_hash = :data_hash
                        WHERE id = :id
                        """
```

- [ ] **Step 2: 改 `_batch_update_school_info` 的 INSERT**

```python
                        insert_query = """
                        INSERT INTO nfa_school (
                            school_id, school_name, region, cp, hash_uuids, 
                            primary_hash_uuid, hash_count, is_key_school, data_hash
                        ) VALUES (
                            :school_id, :school_name, :region, :cp, :hash_uuids, 
                            :primary_hash_uuid, :hash_count, :is_key_school, :data_hash
                        )
                        """
```

- [ ] **Step 3: 改 `_update_school_info`（向后兼容路径）的两条语句**

UPDATE（`:371`）：

```python
                        update_query = "UPDATE nfa_school SET school_name = :school_name, hash_uuids = :hash_uuids, primary_hash_uuid = :primary_hash_uuid, hash_count = :hash_count, is_key_school = :is_key_school, data_hash = :data_hash WHERE id = :id"
```

INSERT（`:377`）：

```python
                    insert_query = "INSERT INTO nfa_school (school_id, school_name, region, cp, hash_uuids, primary_hash_uuid, hash_count, is_key_school, data_hash) VALUES (:school_id, :school_name, :region, :cp, :hash_uuids, :primary_hash_uuid, :hash_count, :is_key_school, :data_hash)"
```

- [ ] **Step 4: 静态自查（约束 2）**

确认 4 条 SQL（2 UPDATE + 2 INSERT）都含 `is_key_school`。`_prepare_school_data` 产出的字典已含该键（Task A1），故 SQLAlchemy 命名参数能绑定。

Run: `python -m pytest tests/test_is_key_school.py -v`（确保 A1 仍 PASS）
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add web/nfa_extractor.py
git commit -m "fix(extractor): nfa_school 写库语句同步写入 is_key_school 列"
```

> 部署后首次运行：旧行 `data_hash` 不含 is_key_school，全表哈希失配 → 每行各 UPDATE 一次 → 自动回填。约束 2 已满足，回填会真正写入列值。

---

## Phase B — nfa-dashboard 后端

> 所有命令在 `C:\Users\haoji\Desktop\Code\nfa-dashboard` 下；Go 命令在 `backend/`。
> **顺序要求：** Task B1 迁移必须先于 Phase A 上线到目标库（见部署顺序）。

### Task B1: 加列迁移 + 建表定义同步

**Files:**
- Create: `sql/migrations/048_add-nfa-school-is-key-school.sql`
- Modify: `sql/nfa_school.sql`
- Modify: `sql/dist/install_full.sql`（`nfa_school` 段）

- [ ] **Step 1: 创建迁移**

Run: `python scripts/sql_migration_guard.py create "add nfa school is key school"`
（生成 `sql/migrations/048_add-nfa-school-is-key-school.sql` 骨架。）

- [ ] **Step 2: 写迁移内容**

```sql
-- contract: add column nfa_school.is_key_school (重点院校标记, 源 nfa_ipgroup.check_status OR 聚合)
ALTER TABLE `nfa_school`
  ADD COLUMN `is_key_school` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否重点院校(源 nfa_ipgroup.check_status OR 聚合)';
ALTER TABLE `nfa_school`
  ADD KEY `idx_is_key_school` (`is_key_school`);
```

- [ ] **Step 3: 同步建表定义**

在 `sql/nfa_school.sql` 的 `data_hash` 列之后、`PRIMARY KEY` 之前加列，并在索引区加 `KEY idx_is_key_school`：

```sql
  `data_hash` char(32) NOT NULL COMMENT '数据hash值，用于快速比较数据是否变化',
  `is_key_school` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否重点院校(源 nfa_ipgroup.check_status OR 聚合)',
  PRIMARY KEY (`id`),
```
```sql
  KEY `idx_data_hash` (`data_hash`),
  KEY `idx_is_key_school` (`is_key_school`)
```

在 `sql/dist/install_full.sql` 的 `nfa_school` 建表段做同样改动。

- [ ] **Step 4: 跑迁移守卫**

Run: `python scripts/sql_migration_guard.py check`
Expected: 通过（无报错）。

- [ ] **Step 5: Commit**

```bash
git add sql/migrations/048_add-nfa-school-is-key-school.sql sql/nfa_school.sql sql/dist/install_full.sql
git commit -m "feat(db): nfa_school 增加 is_key_school 列(迁移 048)"
```

---

### Task B2: School 模型增加 IsKeySchool 字段

**Files:**
- Modify: `backend/internal/model/school.go:8-19`

- [ ] **Step 1: 加字段**

在 `School` 结构体 `DataHash` 之后加：

```go
	DataHash        string    `gorm:"column:data_hash;not null" json:"data_hash"`
	IsKeySchool     int       `gorm:"column:is_key_school;not null;default:0" json:"is_key_school"`
```

- [ ] **Step 2: 编译验证**

Run（在 `backend/`）: `go build ./...`
Expected: 编译通过。

- [ ] **Step 3: Commit**

```bash
git add backend/internal/model/school.go
git commit -m "feat(model): School 增加 IsKeySchool 字段"
```

---

### Task B3: 控制器解析 is_key_school（纯函数 + 单测）

**Files:**
- Modify: `backend/internal/controller/school_controller.go`
- Create: `backend/internal/controller/school_controller_is_key_school_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/controller/school_controller_is_key_school_test.go`:

```go
package controller

import "testing"

func TestNormalizeIsKeySchoolFilter(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   string
		wantOk bool
	}{
		{name: "empty", input: "", want: "", wantOk: false},
		{name: "one", input: "1", want: "1", wantOk: true},
		{name: "zero", input: "0", want: "0", wantOk: true},
		{name: "true", input: "true", want: "1", wantOk: true},
		{name: "false", input: "false", want: "0", wantOk: true},
		{name: "spaces", input: " 1 ", want: "1", wantOk: true},
		{name: "invalid", input: "2", want: "", wantOk: false},
		{name: "garbage", input: "yes", want: "", wantOk: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := normalizeIsKeySchoolFilter(tt.input)
			if got != tt.want || ok != tt.wantOk {
				t.Fatalf("normalizeIsKeySchoolFilter(%q) = (%q,%v), want (%q,%v)", tt.input, got, ok, tt.want, tt.wantOk)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run（在 `backend/`）: `go test ./internal/controller -run TestNormalizeIsKeySchoolFilter -v`
Expected: FAIL（`undefined: normalizeIsKeySchoolFilter`）。

- [ ] **Step 3: 实现纯函数 + 接线两个控制器**

在 `school_controller.go`（与 `parseTrafficTimeParam` 同区）加：

```go
// normalizeIsKeySchoolFilter 解析 is_key_school 查询参数为 "0"/"1"；空或非法返回 ok=false（即不过滤）
func normalizeIsKeySchoolFilter(raw string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true":
		return "1", true
	case "0", "false":
		return "0", true
	default:
		return "", false
	}
}
```

在 `GetAllSchools`（`:352`）解析参数并传给 service（`normalizeIsKeySchoolFilter` 在非法/空时返回 `""`，可直接作为「不过滤」传入，无需额外辅助函数）：

```go
	cp := ctx.Query("cp")
	isKeySchool, _ := normalizeIsKeySchoolFilter(ctx.Query("is_key_school"))
```
调用处改为：`c.schoolService.GetAllSchools(schoolName, region, cp, isKeySchool, limit, offset)`。

在 `GetAllSchoolsV2`（`:222`）同样 `isKeySchool, _ := normalizeIsKeySchoolFilter(ctx.Query("is_key_school"))`，传给 `GetAllSchoolsWithScope`（见 B4 的新增参数位）。

> service 签名变更见 Task B4；本步先让控制器侧编译可后置到 B4 完成后统一 `go build`。建议按 B4→B3 顺序实现，或在本步同时完成 B4 的签名改动。

- [ ] **Step 4: Run test to verify it passes**

Run（在 `backend/`）: `go test ./internal/controller -run TestNormalizeIsKeySchoolFilter -v`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add backend/internal/controller/school_controller.go backend/internal/controller/school_controller_is_key_school_test.go
git commit -m "feat(controller): /schools 解析 is_key_school 筛选参数"
```

---

### Task B4: 服务 + 仓库透传并应用 is_key_school 过滤

**Files:**
- Modify: `backend/internal/service/school_service.go`
- Modify: `backend/internal/repository/school_repository.go:42-90`

- [ ] **Step 1: service 增加参数**

`GetAllSchools` 与 `GetAllSchoolsWithScope` 增加 `isKeySchool string` 参数（空串=不过滤）。在构造 filter 处：

```go
	if isKeySchool != "" {
		filter["is_key_school"] = isKeySchool
	}
```
（`GetAllSchools` 接口签名相应改为 `GetAllSchools(schoolName, region, cp, isKeySchool string, limit, offset int)`；`GetAllSchoolsWithScope` 末位 limit/offset 前插入 `isKeySchool string`。同步更新 `SchoolService` 接口定义。）

- [ ] **Step 2: 仓库应用过滤**

在 `GetAllSchools` 的 filter 循环里，对 `is_key_school` 做精确匹配。在 `switch key` 的字符串分支增加 case：

```go
				case "school_id", "primary_hash_uuid", "data_hash", "is_key_school":
					query = query.Where(key+" = ?", strValue)
```
（即把 `is_key_school` 并入既有精确匹配分支；因 service 以字符串 `"0"/"1"` 传入，走 `strValue` 路径即可。）

- [ ] **Step 3: 编译 + 全量测试**

Run（在 `backend/`）: `go build ./... && go test ./internal/controller ./internal/service -v`
Expected: 编译通过；控制器/服务测试 PASS（含 B3 的 `TestNormalizeIsKeySchoolFilter`）。

- [ ] **Step 4: 手动验证查询（可选，需 DB）**

启动后端 `go run main.go`，登录后：
`GET /api/v1/schools?is_key_school=1&limit=5` → 返回项 `is_key_school` 全为 1；
`GET /api/v1/schools?is_key_school=0&limit=5` → 全为 0；
不带参数 → 混合返回。

- [ ] **Step 5: Commit**

```bash
git add backend/internal/service/school_service.go backend/internal/repository/school_repository.go
git commit -m "feat(school): 服务/仓库支持 is_key_school 过滤"
```

---

## Phase C — nfa-dashboard 前端

> 命令在 `frontend/frontend/` 下。

### Task C1: 重点院校 util（纯函数 + vitest）

**Files:**
- Create: `frontend/frontend/src/views/key-school-utils.ts`
- Create: `frontend/frontend/src/views/__tests__/key-school-utils.spec.ts`

- [ ] **Step 1: Write the failing test**

Create `frontend/frontend/src/views/__tests__/key-school-utils.spec.ts`:

```ts
import { describe, it, expect } from 'vitest'
import { buildKeySchoolSet, isKeySchool } from '../key-school-utils'

describe('key-school-utils', () => {
  it('buildKeySchoolSet 收集 is_key_school===1 的 school_id（OR by school_id）', () => {
    const rows = [
      { school_id: 's1', is_key_school: 1 },
      { school_id: 's2', is_key_school: 0 },
      { school_id: 's3', is_key_school: 1 },
      { school_id: 's3', is_key_school: 0 }, // 同 id 另一 cp，非重点
    ]
    const set = buildKeySchoolSet(rows)
    expect(set.has('s1')).toBe(true)
    expect(set.has('s2')).toBe(false)
    expect(set.has('s3')).toBe(true) // OR：任一行为 1 即重点
  })

  it('isKeySchool 按 school_id 命中', () => {
    const set = new Set(['s1', 's3'])
    expect(isKeySchool(set, 's1')).toBe(true)
    expect(isKeySchool(set, 's2')).toBe(false)
    expect(isKeySchool(set, undefined)).toBe(false)
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm run test:unit -- src/views/__tests__/key-school-utils.spec.ts`
Expected: FAIL（无法解析模块 `../key-school-utils`）。

- [ ] **Step 3: 实现 util**

Create `frontend/frontend/src/views/key-school-utils.ts`:

```ts
export interface KeySchoolRow {
  school_id?: string | null
  is_key_school?: number | string | null
}

/** 收集重点院校 school_id 集合（按 school_id OR 聚合）。 */
export function buildKeySchoolSet(rows: KeySchoolRow[] | null | undefined): Set<string> {
  const set = new Set<string>()
  for (const r of rows || []) {
    const id = r?.school_id
    if (id == null || id === '') continue
    if (Number(r?.is_key_school) === 1) set.add(String(id))
  }
  return set
}

/** 某 school_id 是否重点院校。 */
export function isKeySchool(set: Set<string>, schoolId: string | null | undefined): boolean {
  if (schoolId == null || schoolId === '') return false
  return set.has(String(schoolId))
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npm run test:unit -- src/views/__tests__/key-school-utils.spec.ts`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add frontend/frontend/src/views/key-school-utils.ts frontend/frontend/src/views/__tests__/key-school-utils.spec.ts
git commit -m "feat(web): 新增重点院校 util(buildKeySchoolSet/isKeySchool)"
```

---

### Task C2: 院校列表展示 + 筛选

**Files:**
- Modify: `frontend/frontend/src/views/SchoolsView.vue`

- [ ] **Step 1: 加筛选字段到查询表单**

`queryForm`（`:36`）增加：

```ts
const queryForm = reactive({
  school_name: '',
  region: '',
  cp: '',
  is_key_school: '' as '' | '0' | '1'
})
```

- [ ] **Step 2: loadSchools 透传参数**

`loadSchools` 的 `params`（`:83`）改为剔除空值的 is_key_school（避免发送空串）：

```ts
    const params: Record<string, any> = {
      school_name: queryForm.school_name,
      region: queryForm.region,
      cp: queryForm.cp,
      limit: pageSize.value,
      offset: (currentPage.value - 1) * pageSize.value
    }
    if (queryForm.is_key_school !== '') params.is_key_school = queryForm.is_key_school
```

- [ ] **Step 3: 筛选 UI + 表格列**

在筛选表单（CP 之后）加：

```vue
        <ElFormItem label="重点院校">
          <ElSelect v-model="queryForm.is_key_school" placeholder="全部" clearable class="field-sm">
            <ElOption label="是" value="1" />
            <ElOption label="否" value="0" />
          </ElSelect>
        </ElFormItem>
```

在表格 `region` 列之后加列：

```vue
        <ElTableColumn label="重点院校" width="100">
          <template #default="scope">
            <ElTag v-if="Number(scope.row.is_key_school) === 1" type="danger" size="small">重点</ElTag>
            <span v-else>-</span>
          </template>
        </ElTableColumn>
```

`handleReset`（`:142`）增加 `queryForm.is_key_school = ''`。`ElTag` 需在 import 列表补上（`:12-23`）。

- [ ] **Step 4: 类型检查 + 构建**

Run: `npm run type-check`
Expected: 通过。

- [ ] **Step 5: Commit**

```bash
git add frontend/frontend/src/views/SchoolsView.vue
git commit -m "feat(web): 院校列表展示重点院校标签并支持筛选"
```

---

### Task C3: 结算视图按 school_id 标注重点院校

**Files:**
- Modify: `frontend/frontend/src/components/settlement/SettlementDataTab.vue`
- Modify: `frontend/frontend/src/views/SettlementUserQueryView.vue`

> 数据源说明：两视图行数据仅含 `school_id`、不含 `is_key_school`，故进入视图时额外拉一次重点院校集合再标注（spec §6.6，前端关联方案）。仅 NFA 侧，EDC 不动。

- [ ] **Step 1: 加载重点院校集合**

在两个视图各引入 util 与一个 ref，并在已有的初始化/查询流程后拉取一次（用现有 api 客户端；`SettlementDataTab` 用 `api.v2.getSchools`，`SettlementUserQueryView` 同）：

```ts
import { buildKeySchoolSet, isKeySchool } from '@/views/key-school-utils'

const keySchoolSet = ref<Set<string>>(new Set())

async function loadKeySchoolSet() {
  try {
    const res: any = await (api as any).v2.getSchools({ is_key_school: 1, limit: 100000 })
    const items = Array.isArray(res) ? res : (res?.items ?? [])
    keySchoolSet.value = buildKeySchoolSet(items)
  } catch {
    keySchoolSet.value = new Set()
  }
}
```

在视图 `onMounted`（或现有首次加载处）调用 `loadKeySchoolSet()`。

- [ ] **Step 2: 表格列加标签**

在两视图列出 `school_id` 的表格里，新增/复用一列展示标签（NFA 行有 `school_id` 字段）：

```vue
        <ElTableColumn label="重点院校" width="90">
          <template #default="scope">
            <ElTag v-if="isKeySchool(keySchoolSet, scope.row.school_id)" type="danger" size="small">重点</ElTag>
            <span v-else>-</span>
          </template>
        </ElTableColumn>
```

（`isKeySchool`、`keySchoolSet` 需在 `<script setup>` 暴露给模板——已是顶层绑定即可；`ElTag` 按需 import。）

- [ ] **Step 3: 类型检查**

Run: `npm run type-check`
Expected: 通过。

- [ ] **Step 4: 全量前端单测**

Run: `npm run test:unit`
Expected: 全绿（含 key-school-utils.spec.ts 与既有结算测试无回归）。

- [ ] **Step 5: Commit**

```bash
git add frontend/frontend/src/components/settlement/SettlementDataTab.vue frontend/frontend/src/views/SettlementUserQueryView.vue
git commit -m "feat(web): 结算视图按 school_id 标注重点院校"
```

---

## 部署顺序（spec §8）

1. 目标库执行 dashboard 迁移 `048`（加列，DEFAULT 0）。
2. 上线 `nfa-extractor` 新版本（Phase A）→ 首次运行触发全表自动回填。
3. 上线 dashboard 后端 + 前端（Phase B/C）。

## 验收清单

- [ ] extractor：`python -m pytest tests/test_is_key_school.py -v` 全 PASS。
- [ ] extractor（联调）：`python test_aggregation.py` 输出重点院校统计、is_key_school ∈ {0,1}。
- [ ] 后端：`go build ./... && go test ./internal/controller ./internal/service` PASS。
- [ ] 后端：`/api/v1/schools?is_key_school=1` 返回项 is_key_school 全 1。
- [ ] 前端：`npm run type-check` 与 `npm run test:unit` 全绿。
- [ ] 前端：院校列表显示「重点」标签与筛选可用；NFA 结算视图标注重点院校。
- [ ] 回填核对：extractor 跑一轮后，抽样源库 check_status=1 的院校，确认 `nfa_school.is_key_school=1`。
