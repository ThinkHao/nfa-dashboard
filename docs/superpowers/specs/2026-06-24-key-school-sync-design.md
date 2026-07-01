# 重点院校（is_key_school）字段双向同步设计

- 日期：2026-06-24
- 状态：待评审
- 涉及仓库：`nfa-extractor`（源数据抽取）+ `nfa-dashboard`（本项目）

## 1. 背景与目标

院校是否为「重点院校」这一属性起源于源库 `nfa_ipgroup` 表的 `check_status` 字段（`check_status=1` 表示重点院校）。目前 `nfa-extractor` 未把该字段同步到本项目，`nfa_school` 表也没有对应列。

**目标**：把「重点院校」标记从 `nfa_ipgroup.check_status` 一路同步到 `nfa_school`，并在 dashboard 的院校列表（含筛选）与结算视图中露出。

## 2. 关键约束（设计的核心）

`nfa_school.data_hash` 是 extractor 的变更检测哈希（`web/nfa_extractor.py:259` `_calculate_data_hash`，对 `school_data` 字典中除 `id/update_time/data_hash` 外的所有键取 MD5）。extractor 仅在「新算哈希 ≠ 库内旧哈希」时才写该行（`web/nfa_extractor.py:314`）。

由此得出三条硬约束：

1. **新字段必须进哈希**：否则 `check_status` 翻转而其他字段不变时哈希不变，extractor 跳过该行，标记永远同步不过来。
2. **进了哈希的字段必须同时进 UPDATE/INSERT 的列清单**（`web/nfa_extractor.py:322` UPDATE、`:338` INSERT）。否则会出现「哈希说变了、列却没写入」的永久漂移：第 1 轮哈希失配触发 UPDATE 但只写了新哈希没写新列 → 列停在 DEFAULT 0；之后哈希恒等 → 永远跳过 → `is_key_school` 被永久锁死在 0 且无报错。**口诀：凡进 data_hash 的字段，必进 UPDATE 的 SET 子句。**
3. **部署有序**：dashboard 加列迁移必须先于 extractor 新版本上线，否则 extractor 写不存在的列会直接报错。

## 3. 已确认的决策

| 决策点 | 选择 |
|---|---|
| 聚合口径（一个 school 聚合多个 ipgroup） | **OR**：任一 ipgroup `check_status=1` → 整校 `is_key_school=1` |
| dashboard 侧列名/类型 | `is_key_school tinyint(1) NOT NULL DEFAULT 0`（派生布尔，不照搬源列名 `check_status`，因这边是 OR 聚合后的院校级标记） |
| extractor 代码改动 | 由我方直接改 `nfa-extractor` 仓库 |
| 前端露出范围 | 院校列表展示 + 筛选；结算视图也标注 |

## 4. 数据流

```
nfa_ipgroup.check_status (源库, 逐 ipgroup)
        │  get_aggregated_school_info(): GROUP BY school_id,region,cp
        │  MAX(CASE WHEN check_status=1 THEN 1 ELSE 0 END) AS is_key_school   ← OR 聚合
        ▼
nfa-extractor _prepare_school_data() → school_data['is_key_school']  → 进 data_hash
        │  _batch_update_school_info() / _update_school_info(): INSERT/UPDATE 含 is_key_school
        ▼
nfa_school.is_key_school (本项目库, 逐 school_id+region+cp)
        │  School 模型 → school_repository → school_service → controller
        ▼
GET /api/v1/schools, /api/v2/schools  (字段随模型返回; 支持 is_key_school 筛选)
        ▼
SchoolsView.vue（列+筛选）/ 结算视图（按 school_id 标注）
```

## 5. nfa-extractor 侧改动（3 处，必须一起改）

文件：`web/nfa_extractor.py`

1. **聚合 SQL**（`get_aggregated_school_info`, `:134`）：在 SELECT 增加
   `MAX(CASE WHEN check_status = 1 THEN 1 ELSE 0 END) AS is_key_school`
   （用 `CASE` 而非裸 `MAX(check_status)`，对 NULL / 非 0/1 取值更稳健；`GROUP BY school_id, region, cp` 不变）。
   - 前置校验：确认源库 `nfa_ipgroup` 确有 `check_status` 列（评审/实现时核对源库 schema）。

2. **`_prepare_school_data`**（`:271`）：往 `school_data` 字典加
   `'is_key_school': int(school_info['is_key_school'])`
   置于 `data_hash` 计算之前（`school_data['data_hash'] = self._calculate_data_hash(school_data)` 那一行之上）。这样它**自动进哈希**，`_calculate_data_hash` 本身不用改。

3. **写库语句**（两条都要改）：
   - `_batch_update_school_info`（`:322` UPDATE、`:338` INSERT）
   - `_update_school_info`（`:371` UPDATE、`:377` INSERT，向后兼容路径）
   - UPDATE 增加 `is_key_school = :is_key_school`；INSERT 的列清单与 VALUES 占位增加 `is_key_school` / `:is_key_school`。

### 自动回填（无需单独脚本）

改动上线后第一次运行：库内每行的旧 `data_hash` 都不含 `is_key_school` 段，新算哈希对**所有行**都失配 → 每行各 UPDATE 一次 → `is_key_school` 全表写满。之后哈希恢复一致，回到增量 no-op；仅当源库 `check_status` 真变化时对应行才再次失配更新。**前提是约束 2 成立**（UPDATE 真的写了该列），否则这唯一一次全表刷新会被白白消费。

## 6. nfa-dashboard 侧改动

### 6.1 SQL 迁移（`sql/migrations/048_*.sql`）

```sql
-- contract: add nfa_school.is_key_school
ALTER TABLE nfa_school
  ADD COLUMN is_key_school tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否重点院校(源 nfa_ipgroup.check_status OR 聚合)';
ALTER TABLE nfa_school ADD KEY idx_is_key_school (is_key_school);
```

- 用 `python scripts/sql_migration_guard.py create "add nfa_school is_key_school"` 创建，改完跑 `... check`。
- 同步更新 `sql/nfa_school.sql` 与 `sql/dist/install_full.sql` 中的 `nfa_school` 建表定义。

### 6.2 模型（`backend/internal/model/school.go`）

`School` 结构体加：
```go
IsKeySchool int `gorm:"column:is_key_school;not null;default:0" json:"is_key_school"`
```
（用 `int`/`int8`，与既有 `HashCount int` 风格一致；前端按 0/1 判断。）

### 6.3 仓库（`backend/internal/repository/school_repository.go`）

`GetAllSchools` 的 filter 循环增加对 `is_key_school` 的处理：当 filter 含 `is_key_school` 且为有效 0/1 时 `query.Where("is_key_school = ?", v)`。注意现有循环按字符串分支处理 value，需要为整型/布尔值新增一个分支（或在 service 层统一转成字符串 `"0"/"1"` 再走精确匹配分支）。

### 6.4 服务 + 控制器

- `school_service.go`：`GetAllSchools` 与 `GetAllSchoolsWithScope` 增加 `isKeySchool *int`（或 `string`）参数，非空时写入 filter。
- `school_controller.go`：`GetAllSchools`（`:352`）与 `GetAllSchoolsV2`（`:222`）解析 `ctx.Query("is_key_school")`，透传给 service。空串=不过滤。
- 路由（`bootstrap/app.go:128/145`）无需新增，权限沿用 `school.read`。
- 返回字段无需改动：`is_key_school` 随 `model.School` 自动序列化。

### 6.5 前端院校列表（`frontend/frontend/src/views/SchoolsView.vue`）

- 表格（`:234`）新增一列「重点院校」：`is_key_school === 1` 渲染 `<ElTag type="danger">重点</ElTag>`，否则空/`-`。
- 筛选表单（`:197`）新增「是否重点院校」`ElSelect`（全部 / 是 / 否），绑定到 `queryForm.is_key_school`，随 `loadSchools` 的 params 一起发送（`:83`）。空值不发送该参数。
- API 客户端：确认 `api.v2.getSchools` 透传任意 query 参数（当前直接展开 `params`，应已支持）。如有显式类型（`frontend/frontend/src/types/api.ts`），补 `is_key_school?` 字段。

### 6.6 结算视图标注

涉及 `SettlementDataTab.vue`（NFA 院校日95）与 `SettlementUserQueryView.vue`（单用户结算查询，NFA）。两者行数据均带 `school_id`。

- **方案（推荐）**：前端在进入视图时拉取一次「重点院校 school_id 集合」（调用 `getSchools({ is_key_school: 1, limit: <足够大> })`，取 `school_id` 去重成 `Set`），在表格里对命中 `school_id` 的行加「重点」标签。理由：与结算后端 handler 解耦，不改结算 SQL。
- **粒度说明**：`nfa_school` 按 `(school_id, region, cp)` 唯一，而 NFA 结算以 `school_id` 为主体。约定按 `school_id` 做 OR：只要该 `school_id` 对应任一 `nfa_school` 行 `is_key_school=1`，结算视图即标注为重点。此约定需在实现时写进注释。
- 不改动 EDC（节点）侧视图——重点院校是 NFA 概念。

## 7. 测试

- **extractor**（`nfa-extractor/tests`, `test_aggregation.py`）：
  - 聚合：同一 school 下混合 `check_status`（0 与 1）→ 期望 `is_key_school=1`（OR）；全 0 → 0。
  - 哈希：构造仅 `is_key_school` 不同的两份 `school_data`，断言 `_calculate_data_hash` 结果不同（防回归：保证字段确实进了哈希）。
  - 写库：断言 INSERT/UPDATE 的参数字典含 `is_key_school`（防约束 2 回归）。
- **dashboard 后端**（`go test ./internal/...`）：
  - 仓库：`is_key_school=1/0` 过滤返回正确子集；不传时不过滤。
  - 控制器：`/schools?is_key_school=1` 透传到 service；空串不过滤。
- **dashboard 前端**（`npm run test:unit`）：
  - SchoolsView：`is_key_school=1` 行渲染「重点」标签；筛选项变化时请求带 `is_key_school` 参数。
  - 结算视图：命中重点集合的行显示标签。

## 8. 部署顺序

1. dashboard 迁移 `048`（加列，DEFAULT 0）。
2. 上线 `nfa-extractor` 新版本 → 首次运行触发全表自动回填。
3. 上线 dashboard 后端 + 前端（读取 + 展示 + 筛选）。

## 9. 不做（YAGNI）

- 不新建逐 ipgroup 的 `check_status` 镜像表（B 方案）——只需院校级标记。
- 不在 EDC 节点侧引入重点概念。
- 不为重点院校加独立权限——沿用 `school.read` / `settlement.data.read`。
