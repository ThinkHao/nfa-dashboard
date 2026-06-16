# CLAUDE.md

## Project Overview

NFA Dashboard（Network Flow Analysis Dashboard）—— 学校流量监控与结算系统。前端 Vue 3 + TypeScript，后端 Go (Gin + GORM)，数据库 MySQL 5.7+。仅支持 Linux 部署。

## Project Structure

```
nfa-dashboard/
├── backend/              # Go 后端 (Gin + GORM, 手动 DI)
│   ├── main.go           # 入口
│   ├── config/           # Viper YAML 配置
│   └── internal/
│       ├── bootstrap/    # 路由注册 & DI (app.go)
│       ├── controller/   # 控制器层
│       ├── service/      # 业务逻辑层
│       ├── repository/   # 数据访问层
│       ├── model/        # GORM 模型
│       ├── middleware/    # Auth, Audit, CORS, Gzip, Logger, RateLimit, SecurityHeaders
│       ├── authz/        # 权限定义
│       └── scheduler/    # 结算任务调度
├── frontend/frontend/    # Vue 3 前端
│   └── src/
│       ├── api/          # Axios 实例 + 全量 API 客户端
│       ├── components/   # 可复用组件 (含 settlement/, ui/)
│       ├── composables/  # 组合式函数
│       ├── layouts/      # DefaultLayout, BlankLayout
│       ├── router/       # 单文件路由 + 权限守卫
│       ├── stores/       # Pinia (auth, theme, tagsView, tasks, counter)
│       ├── types/        # TypeScript 类型
│       ├── utils/        # 工具函数
│       └── views/        # 页面视图（NFA: SettlementUserQueryView 单用户结算查询；EDC: SettlementNodeQueryView 单节点结算查询）
├── cli/                  # Go CLI (独立 go.mod, HTTP API 客户端)
├── sql/migrations/       # 增量 SQL 迁移
├── scripts/              # 部署 & 迁移守卫脚本
├── compose/              # Docker Compose 配置
├── docs/                 # 文档
└── agents.md             # Agent/贡献者详细规则
```

## Tech Stack

| Layer | Technology |
|---|---|
| Frontend framework | Vue 3.5 (Composition API, `<script setup>`) |
| Build tool | Vite 6.2 |
| Language | TypeScript 5.8 |
| UI library | Element Plus 2.9 (Chinese locale) |
| State management | Pinia 3.0 |
| Routing | Vue Router 4.5 (HTML5 history) |
| HTTP client | Axios 1.9 (JWT interceptor, auto refresh, 401 retry) |
| Charts | ECharts 5.6 + vue-echarts 7.0 |
| Backend language | Go 1.24 |
| HTTP framework | Gin 1.8 |
| ORM | GORM 1.24 (MySQL) |
| Auth | JWT (golang-jwt/jwt/v5), access + refresh token |
| Config | Viper (YAML) |
| Testing | Vitest (frontend unit), Playwright (frontend e2e), go test (backend) |
| Linting | ESLint 9 + Prettier 3.5 |

## Key Commands

### Backend (run from `backend/`)

```bash
go run main.go                                    # 启动开发服务器 (localhost:8081)
go test ./internal/service                         # 服务层测试
go test ./internal/controller                      # 控制器层测试
go test ./...                                      # 全量测试
```

### Frontend (run from `frontend/frontend/`)

```bash
npm install                                        # 安装依赖
npm run dev                                        # 启动开发服务器 (localhost:5173)
npm run type-check                                 # TypeScript 类型检查
npm run test:unit                                  # 单元测试
npm run test:unit -- <spec-files>                  # 指定文件测试
npm run build                                      # 类型检查 + 生产构建
npm run lint                                       # ESLint 检查
npm run format                                     # Prettier 格式化
```

### SQL Migrations

```bash
python scripts/sql_migration_guard.py check        # 提交前检查
python scripts/sql_migration_guard.py create "title"   # 创建新迁移
python scripts/sql_migration_guard.py adopt "title" --source path/to/file.sql  # 接管已有 SQL
# 或
powershell -File scripts/check-sql-migrations.ps1
```

## Architecture Rules

### Backend Layering

严格分层：Controller → Service → Repository → Model。手动依赖注入（无 DI 框架），在 `backend/internal/bootstrap/app.go` 中组装。

API 两个版本：
- `/api/v1/` — Auth, Schools, Traffic, Settlement, System (users/roles/permissions/settings)
- `/api/v2/` — Schools, Traffic, EDC traffic, Settlement data（带 per-user traffic-scope 过滤）

### CLI (`cli/`)

- CLI 是 HTTP API 客户端，**禁止**直接连 MySQL 或绕过 Gin 中间件
- 所有调用必须走 `/api/v1` 或 `/api/v2` + JWT auth，保持 RBAC、traffic-scope、审计日志一致
- **禁止**在输出中打印 access/refresh token；使用环境变量 `NFA_DASHBOARD_BASE_URL`, `NFA_DASHBOARD_TOKEN`, `NFA_DASHBOARD_REFRESH_TOKEN`
- 写操作支持 `--dry-run`（显示 method, path, query, body 而不发送）
- 默认输出为人类/agent 可读摘要，JSON 保存到文件；仅 `--print-body` 时输出完整 JSON
- 流量数据转换：`raw_bytes * 8 / 60 / 1_000_000` = Mbps（不要用 `/300`）
- 比特率单位使用十进制 `1000`（`Mbps = bits/s / 1_000_000`），`traffic_byte_unit_base` 仅影响字节大小显示（B/KB/MB/GB），不影响 bps/Kbps/Mbps

### SQL & Migrations

- 增量迁移唯一入口：`sql/migrations/`
- **禁止**在根目录 `sql/` 下新增 `NNN_*.sql`
- 修改迁移文件后**必须**运行迁移守卫脚本
- 不要重新编号已有迁移（除非明确要求）
- 迁移含 schema 变更时，在文件头声明 `-- contract:`

## Business Semantics

### RBAC & Data Scope

- 保持 RBAC 和用户可见数据范围，CLI 不得扩大用户权限
- 所有变更操作记录审计日志

### Settlement (结算)

- **Daily 95 vs Range 95**：两种不同的 95 百分位计算模式
- **Recv vs Both direction**：流量方向语义
- **Unit base 1000 vs 1024**：GB vs GiB，影响价格计算
- EDC 节点 95 结算的运行时费率取自 `rate_final_node`，不仅 `rate_node`
- EDC 节点日/月 95 金额计算：流量单价为 `元/G`，转换公式 `mbps_95 / unit_base * fee`（1000 用 GB，1024 用 GiB）
- **创建端点禁止同步计算结果**：仅做廉价校验 → 创建任务行 → 异步计算
- "是否有流量" 检查用 `SELECT 1 ... LIMIT 1` 而非 `COUNT(*)`
- 节点日/月 95 任务：日 95 逐天处理，月 95 逐月处理，范围任务在单个任务内顺序执行已有逐期计算
- 用户可见的 fee-owner 字段显示系统用户的 alias/display name/username，而非原始数字 ID
- 95 百分位排名口径必须前后端对齐（见 `settlement-node-query-utils.ts` 与后端计算），避免单节点查询与结算任务结果出现 off-by-one 偏差

### Settlement Pages (结算页面结构)

- **两条数据链路**：NFA（院校，`nfa-extractor` → `school_settlement`，仅**日95**）与 EDC（节点，`edc-extractor` → `settlement_node_daily95`/`settlement_node_monthly95`，**日95 + 月95**）相互独立。`流量监控`(TrafficView) 用 `data_source` 开关统一承载两源。
- **结算中心（SettlementView）只保留 4 个 Tab**：结算数据、结算任务、结算配置、结算公式。
  - `结算数据`(SettlementDataTab) 是 NFA/EDC × 日/月 的**唯一明细视图**，靠 `数据源` + `聚合粒度` 两个开关切换，自带导出与（NFA）复算 / 重建月度快照。
  - **禁止**再新增按数据源或按粒度拆分的独立"明细"Tab/页面。历史上的 院校日95明细 / 节点日95明细 / 节点月95明细 与 `结算数据` 读同一张表、同一后端 handler（`ListNodeDaily`/`ListNodeMonthly`），已合并；对应后端 `daily-details`、`node-daily-details`、`node-monthly-details` 路由随之下线。
- **消费 / 对账面板**（独立路由，权限 `settlement.data.read`，带 traffic-scope 过滤、按月列对账视图、导出）：
  - `单用户结算查询`(SettlementUserQueryView) = NFA 院校；
  - `单节点结算查询`(SettlementNodeQueryView) = EDC 节点 / 结算分组。
- 运营视图（`结算数据`，需 `settlement.calculate`）与消费面板（需 `settlement.data.read`）权限与职责不同，**不要互相合并**。

### Node Settlement Groups (节点结算分组)

- EDC 节点结算支持把多个采集节点（`edc_entity`）聚合为一个**计费主体**结算，分组表为 `edc_node_settlement_groups` / `edc_node_settlement_group_members`（迁移 `046`）
- `billing_subject_type`（默认 `node`，可为 `group`）+ `billing_subject_id` + `billing_display_name` 贯穿 `rate_node`、`rate_final_node`、`settlement_node_daily95`、`settlement_node_monthly95`
- 分组管理走 `/api/v1/.../rates/node-groups`（GET/POST/PUT/DELETE，权限 `rates.node.read` / `rates.node.write`），前端在 `NodeRatesView.vue` 维护
- 一个 entity 仅能属于一个分组（`uk_..._members_entity` 唯一约束）

### Node Daily95/Monthly95 Task Creation

- 日期/月份范围在 UI 层方便操作，但后端只创建**一个**任务行
- **禁止**将 30 天或 12 个月展开为多个 `nfa_settlement_task` 行
- 优先使用 range payload：`{ start_date, end_date }` / `{ start_month, end_month }`

## Frontend Notes

- 结算任务模态框必须防止重复点击：提交中时禁用 cancel/close/ESC/mask close
- 使用 `cleanupStaleElementOverlays` 在对话框关闭后清理残留 overlay
- 长耗时异步创建操作：优先优化后端预检成本，而非仅增加 Axios 超时时间
- 范围任务创建的单元测试应断言请求数量和 payload 结构，防止未来回退为逐天/逐月前端循环

## Conventions

- 前端组件使用 Composition API `<script setup>` 语法
- 路径别名 `@/*` → `./src/*`
- 前端手动 chunk 分割：echarts, element-plus, vue/pinia/router, vendor
- Vite large chunk warning 不算构建失败，除非任务明确关注包体积
- `strict: false`, `noImplicitAny: false` 在 tsconfig 中
- 后端端口默认 8081，前端 dev server 默认 5173
- 如果 `go run main.go` 报 `:8081 bind` 错误，先找出并停掉已有进程，不要另起随机端口
