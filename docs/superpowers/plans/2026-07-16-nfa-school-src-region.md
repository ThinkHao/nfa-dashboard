# NFA School Source Region Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `nfa_school.src_region` and a guarded, preview-first tool that computes and optionally applies the approved source-region rules.

**Architecture:** A numbered idempotent migration adds the nullable runtime column and the Go model exposes it. A focused Python module owns pure rule evaluation plus a small MySQL adapter; its default command is read-only JSON preview, while execution requires two explicit flags, creates a timestamped backup table, updates only deterministic rows, and verifies the committed result.

**Tech Stack:** MySQL 8-compatible SQL, Go/GORM model, Python 3 standard library plus installed `pymysql`, `unittest`, repository SQL migration guard.

---

## File map

- Create `sql/migrations/<guard-generated>_add-nfa-school-src-region.sql`: idempotent column migration and runtime contract.
- Modify `backend/internal/model/school.go`: expose `src_region` in GORM and JSON.
- Modify `scripts/offline-deploy.sh`: fail deployment when the runtime column is absent.
- Create `scripts/nfa_school_src_region.py`: pure mapping/rule logic, preview report, MySQL read/backup/update/verify, CLI gates.
- Create `scripts/test_nfa_school_src_region.py`: unit tests for all rule precedence and execution-gate behavior.

### Task 1: Add the runtime schema contract

**Files:**
- Create: guard-generated file under `sql/migrations/`
- Modify: `backend/internal/model/school.go`
- Modify: `scripts/offline-deploy.sh`

- [ ] **Step 1: Generate the migration filename with the repository guard**

Run:

```powershell
python scripts/sql_migration_guard.py create "add nfa school src region"
```

Expected: one new migration after `048`, under `sql/migrations/`; do not create a numbered file under `sql/`.

- [ ] **Step 2: Write the idempotent migration**

Use this SQL body, retaining the filename generated in Step 1:

```sql
-- Migration: add nfa school source region
-- contract: column=nfa_school.src_region
-- src_region remains nullable until the separately reviewed backfill is executed.
SET @col_exists := (
  SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'nfa_school'
    AND COLUMN_NAME = 'src_region'
);
SET @ddl := IF(
  @col_exists = 0,
  'ALTER TABLE `nfa_school` ADD COLUMN `src_region` varchar(20) NULL COMMENT ''服务源区域'' AFTER `region`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
```

- [ ] **Step 3: Add the model field**

Insert after `Region` in `model.School`:

```go
SrcRegion      *string   `gorm:"column:src_region" json:"src_region"`
```

Pointer semantics preserve the distinction between unfilled `NULL` and a valid region.

- [ ] **Step 4: Extend offline schema validation**

Add to the `checks` array in `assert_db_schema()`:

```bash
"SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='nfa_school' AND COLUMN_NAME='src_region';|nfa_school.src_region 列缺失"
```

- [ ] **Step 5: Validate and commit the schema slice**

Run:

```powershell
python scripts/sql_migration_guard.py check
gofmt -w backend/internal/model/school.go
cd backend
go test ./internal/model ./internal/repository
```

Expected: guard passes; Go packages pass.

Commit:

```powershell
git add sql/migrations backend/internal/model/school.go scripts/offline-deploy.sh
git commit -m "feat: add school source region field"
```

### Task 2: Implement rule evaluation test-first

**Files:**
- Create: `scripts/nfa_school_src_region.py`
- Create: `scripts/test_nfa_school_src_region.py`

- [ ] **Step 1: Write failing rule tests**

Create tests that import `resolve_src_region` and assert this exact matrix:

```python
CASES = [
    ("陕西", "bilibili", "天津", "bilibili_fallback_tianjin"),
    ("陕西", "ali", "北京", "ali_other_beijing"),
    ("陕西", "jsy", "北京", "recognized_fallback_beijing"),
    ("广东", "ali", "广东", "guangdong_ali"),
    ("江苏", "jsy", "上海", "jiangsu_jsy_cnc_shanghai"),
    ("江苏", "cnc", "上海", "jiangsu_jsy_cnc_shanghai"),
    ("吉林", "bilibili", "北京", "jilin_bilibili_beijing"),
    ("山东省", "bsy", "山东省", "local_normal_node"),
]
```

Also assert that `resolve_src_region("陕西", "unknown")` returns an error result with no target, and that a down/offline-only pair is absent from the local-node set.

- [ ] **Step 2: Run the tests and confirm RED**

Run:

```powershell
python -m unittest scripts.test_nfa_school_src_region -v
```

Expected: import failure because the implementation module does not exist.

- [ ] **Step 3: Implement the minimal pure rule engine**

Define:

```python
KNOWN_CPS = {"bilibili", "ali", "jsy", "cnc", "bsy", "xinliu"}
LOCAL_NORMAL_PAIRS = {
    ("上海", "cnc"), ("北京", "cnc"), ("广东", "cnc"), ("湖北", "cnc"),
    ("山东", "bsy"), ("江苏", "bsy"), ("上海", "bsy"), ("湖北", "bsy"),
    ("北京", "bsy"), ("四川", "bsy"),
    ("广东", "bilibili"), ("上海", "bilibili"), ("山东", "bilibili"),
    ("江苏", "bilibili"), ("河南", "bilibili"), ("湖北", "bilibili"),
    ("四川", "bilibili"), ("天津", "bilibili"), ("北京", "bilibili"),
    ("湖南", "bilibili"), ("福建", "bilibili"),
    ("上海", "jsy"), ("广东", "jsy"), ("河南", "jsy"), ("北京", "jsy"),
    ("北京", "xinliu"), ("广东", "xinliu"), ("山东", "xinliu"), ("四川", "xinliu"),
}
```

Normalize only administrative suffixes for comparison. Return a small immutable result containing `target`, `rule`, and optional `error`; implement the seven approved rules in exact priority order. Preserve the input `region` when returning `local_normal_node`.

- [ ] **Step 4: Run the tests and confirm GREEN**

Run:

```powershell
python -m unittest scripts.test_nfa_school_src_region -v
```

Expected: all rule tests pass, including both Shaanxi assertions.

- [ ] **Step 5: Commit the rule slice**

```powershell
git add scripts/nfa_school_src_region.py scripts/test_nfa_school_src_region.py
git commit -m "feat: implement school source region rules"
```

### Task 3: Add deterministic preview reporting

**Files:**
- Modify: `scripts/nfa_school_src_region.py`
- Modify: `scripts/test_nfa_school_src_region.py`

- [ ] **Step 1: Write failing report tests**

Use three in-memory rows: one `NULL` Shaanxi Bilibili row, one already-correct Guangdong Ali row, and one unknown CP. Assert summary values `total=3`, `will_update=1`, `unchanged=1`, `errors=1`; assert every detail includes `id`, `school_name`, `region`, `cp`, `current_src_region`, `target_src_region`, and `rule`.

- [ ] **Step 2: Run the focused test and confirm RED**

```powershell
python -m unittest scripts.test_nfa_school_src_region.PreviewTests -v
```

Expected: failure because `build_preview` is missing.

- [ ] **Step 3: Implement preview construction and JSON output**

Implement `build_preview(rows)` with sorted details (`region`, `cp`, `school_name`, `id`), `rule_counts`, `updates`, `unchanged`, and `errors`. Add CLI options:

```text
--output PATH                  default outputs/nfa-school-src-region-preview.json
--execute                      default false
--confirm                      default false
--host/--port/--user/--password/--database
```

Database options default from `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASS`, and `DB_NAME`; never print the password. Default command executes only:

```sql
SELECT id, school_id, school_name, region, cp, src_region
FROM nfa_school
ORDER BY region, cp, school_name, id
```

Create the output directory only after successful read, serialize UTF-8 JSON with `ensure_ascii=False`, and print only a concise bucket summary plus output path.

- [ ] **Step 4: Run tests and a CLI help smoke test**

```powershell
python -m unittest scripts.test_nfa_school_src_region -v
python scripts/nfa_school_src_region.py --help
```

Expected: tests pass; help documents that preview is the default.

- [ ] **Step 5: Commit preview support**

```powershell
git add scripts/nfa_school_src_region.py scripts/test_nfa_school_src_region.py
git commit -m "feat: add source region backfill preview"
```

### Task 4: Add backup-first execution gates

**Files:**
- Modify: `scripts/nfa_school_src_region.py`
- Modify: `scripts/test_nfa_school_src_region.py`

- [ ] **Step 1: Write failing execution-gate tests**

Assert `--execute` without `--confirm` exits nonzero before opening a connection. With a fake connection, assert the execution call order is: begin, `CREATE TABLE ... LIKE nfa_school`, `INSERT ... SELECT * FROM nfa_school`, parameterized `UPDATE nfa_school SET src_region=%s WHERE id=%s`, verification select, commit. Assert any exception calls rollback.

- [ ] **Step 2: Run and confirm RED**

```powershell
python -m unittest scripts.test_nfa_school_src_region.ExecutionTests -v
```

Expected: failures because execution helpers are missing.

- [ ] **Step 3: Implement guarded execution**

Require both flags. Generate a safe backup identifier matching `nfa_school_src_region_backup_YYYYMMDD_HHMMSS`; reject any identifier outside `[A-Za-z0-9_]+`. Re-read rows inside the transaction, rebuild the preview, abort if errors exist, create/copy the backup, execute parameterized updates, verify every deterministic row, and commit. Return the backup table name and counts without credentials.

- [ ] **Step 4: Run all Python tests**

```powershell
python -m unittest scripts.test_nfa_school_src_region -v
```

Expected: rule, preview, gate, backup-order, commit, and rollback tests all pass.

- [ ] **Step 5: Commit execution support**

```powershell
git add scripts/nfa_school_src_region.py scripts/test_nfa_school_src_region.py
git commit -m "feat: guard source region backfill execution"
```

### Task 5: Generate the real read-only preview and verify delivery

**Files:**
- Create: `outputs/nfa-school-src-region-preview.json` (delivery artifact; do not commit if `outputs/` is ignored)
- Modify: `docs/superpowers/plans/2026-07-16-nfa-school-src-region.md` (checkboxes only)

- [ ] **Step 1: Apply only the schema migration to the authorized target database**

Use the repository deployment/migration mechanism for the selected environment. Do not run the backfill execution mode. Verify:

```sql
SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA=DATABASE()
  AND TABLE_NAME='nfa_school'
  AND COLUMN_NAME='src_region';
```

Expected: one `varchar(20)` nullable column.

- [ ] **Step 2: Generate the real preview**

Run with environment variables or explicit non-secret flags; provide the password through `DB_PASS`, not stdout:

```powershell
python scripts/nfa_school_src_region.py --output outputs/nfa-school-src-region-preview.json
```

Expected: read-only completion, JSON artifact created, no backup table, no updated rows.

- [ ] **Step 3: Verify preview assertions**

Inspect JSON programmatically and assert:

```python
assert errors == 0
assert every Shaanxi bilibili target is Tianjin
assert every Shaanxi non-bilibili recognized target is Beijing
assert sum(rule_counts.values()) + errors == total
assert will_update + unchanged + errors == total
```

If unknown CPs exist, do not execute and present them as an explicit pending list instead of weakening `errors == 0`.

- [ ] **Step 4: Run repository verification**

```powershell
python scripts/sql_migration_guard.py check
python -m unittest scripts.test_nfa_school_src_region -v
cd backend
go test ./internal/model ./internal/repository
cd ..
git diff --check
git status --short
```

Expected: all targeted checks pass; only known user files and intended changes/artifacts remain.

- [ ] **Step 5: Stop for user confirmation**

Report schema result, preview totals, each rule bucket, Shaanxi examples, unknown/pending CPs, artifact path, and the exact guarded execution command. Do not run `--execute --confirm` until the user explicitly approves this concrete preview.
