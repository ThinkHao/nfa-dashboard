# nfa-dashboard Agent Notes

## CLI And Permissions

- The project-level CLI lives in `cli/` and is an HTTP API client. Keep it independent from `backend/`, and do not make CLI write paths connect directly to MySQL or bypass Gin middleware.
- CLI calls must inherit the existing backend permission model by using `/api/v1` or `/api/v2` endpoints with JWT auth. This preserves `AuthRequired`, `PermissionRequired`, effective traffic-scope filtering, and operation-log auditing.
- When adding or changing backend endpoints in `backend/internal/bootstrap/app.go`, update a typed CLI command when the endpoint is agent-facing. If no typed command is added, verify the generic `api get|post|put|delete --path ...` path still covers the endpoint.
- Do not print access tokens or refresh tokens in CLI output. Prefer `NFA_DASHBOARD_BASE_URL`, `NFA_DASHBOARD_TOKEN`, and `NFA_DASHBOARD_REFRESH_TOKEN` for automation.
- Use `--dry-run` before agent-driven write calls when the payload is non-trivial. Dry-run output should show method, path, query, body, file, and download target without sending the request.
- CLI default output should be a concise summary for humans/agents, with JSON responses saved to disk. Use `--print-body` only when the caller explicitly needs the full JSON on stdout.
- If the CLI is meant to be used outside this repository, publish versioned release binaries for common platforms and include `checksums.txt`: Windows amd64, Linux amd64/arm64, and macOS amd64/arm64. Keep `nfa-dashboard-cli version` and the release workflow aligned when changing release behavior.
- For traffic data commands, preserve JSON export for agent reading and support optional SVG export for human inspection. Summaries should include point count, time bounds, average/max Mbps, and 95th percentile values.
- NFA traffic monitor raw points convert to Mbps with `raw_bytes * 8 / 60 / 1_000_000`. Do not use `/300` for the CLI traffic summary or SVG chart. Keep CLI bit-rate output aligned with `TrafficView.vue` / `formatBitRate`: bit-rate units are decimal `1000` (`Mbps = bits/s / 1_000_000`). `traffic_byte_unit_base` only affects byte-size displays such as B/KB/MB/GB, not bps/Kbps/Mbps/Gbps.
- For the single-user settlement page, use `settlement user-panel` instead of mixing `settlement channel-results list`, `settlement data monthly`, or daily detail commands by hand. This command must mirror `SettlementUserQueryView.vue`: monthly amount rows come from `/api/v1/settlement/data/customer/monthly`; monthly 95 column values come from `/api/v1/settlement/data/customer` grouped by school/day/month and converted with the single-user unit settings.

## SQL And Migrations

- For SQL schema or migration changes, run the repository migration guard before completion:
  - `python scripts/sql_migration_guard.py`
  - or `powershell -File scripts/check-sql-migrations.ps1`
- Keep numbered SQL files and `sql/migrations/` ordering consistent. Do not renumber existing migrations unless explicitly asked.

## Business Semantics

- Preserve RBAC and user-visible data scope. CLI conveniences must not widen a user beyond the backend's effective permissions or traffic-scope rules.
- Preserve settlement semantics when adding CLI helpers: daily 95 vs range 95, recv vs both direction, unit base `1000` vs `1024`, 5m traffic precision, and V4/V6 label distinctions are business-visible details.
- For production-like repair or bulk update workflows, default to preview-first and backup-first. Use CLI dry-run plus explicit payload review before sending mutating requests.
- For discrepancy analysis, provide evidence from code paths, API responses, DB recomputation, or operation logs rather than stopping at a plausible explanation.
- For EDC node 95 settlement, runtime rates come from `rate_final_node`, not only `rate_node`. When node daily/monthly 95 tasks fail, first check `nfa_settlement_task.error_message`, then verify traffic existence, business rates, and final-node rates independently.
- For EDC node daily/monthly 95 amount calculation, traffic unit prices are `元/G`. Do not multiply the Mbps 95 value directly by the fee; convert with `mbps_95 / unit_base * fee` so `1000` uses GB and `1024` uses GiB.
- For user-visible fee-owner fields, show the system user's alias/display name/username instead of raw numeric IDs. This applies to tables, summaries, tooltips, exports, and edit forms/dropdowns. Raw `*_owner_id` values may remain in payloads or hidden import/export compatibility fields, but visible UI should resolve them to names.
- For node daily95/monthly95 task creation from a date/month range, keep the UI as a convenience layer but create one backend task for the selected range. Do not expand a 30-day or 12-month range into many `nfa_settlement_task` rows. Keep the existing single-period payloads compatible, but prefer range payloads such as `{ start_date, end_date }` and `{ start_month, end_month }`.
- Do not make creation endpoints synchronously calculate settlement results. Creation should perform only cheap validation, create the task row, and run calculation asynchronously. For "has traffic" checks, prefer existence queries (`SELECT 1 ... LIMIT 1`) over full `COUNT(*)` scans.
- Preserve node 95 calculation behavior when optimizing task creation: daily node 95 still processes one day at a time, monthly node 95 still processes one month at a time, and range tasks should execute those existing per-period calculations sequentially inside one task.

## Frontend Notes

- Settlement task modals should guard long-running submissions: prevent duplicate clicks while `submitting`, disable cancel/close/ESC/mask close during submission, and clean stale Element Plus overlays with `cleanupStaleElementOverlays` after dialogs close.
- For long-running but asynchronous create actions, prefer improving backend preflight cost over only increasing Axios timeouts. Endpoint-specific timeouts are acceptable as a fallback, but should not hide expensive synchronous work.
- Use focused Vue unit tests for settlement task UI behavior. For range task creation, assert request count and payload shape so future changes do not accidentally reintroduce per-day/per-month frontend loops.

## Local Development And Verification

- 本地开发服务一律在 VS Code 的集成终端或调试配置中前台运行，不使用 `Start-Process`、后台进程、服务管理器或其他脱离 VS Code 的方式启动。
- dashboard 后端应在 VS Code 集成终端执行 `go run .`（工作目录为 `backend/`），前端应在 VS Code 集成终端执行 `npm run dev -- --host 127.0.0.1`（工作目录为 `frontend/frontend/`）。需要停止或重启时，回到对应 VS Code 终端操作。
- 如果无法使用 VS Code 集成终端，应先向用户说明并暂停，不要自行改用隐藏后台进程。
- 本地开发服务必须与远端及生产容器区分；启动、停止、重启前核对工作目录、端口和数据库配置，不操作 `192.168.9.104` 的生产实例。
- The local backend commonly runs on `http://localhost:8081`; the frontend dev server commonly runs on `http://localhost:5173`. If `go run .` fails with `listen tcp :8081: bind`, identify and stop the existing backend process from its VS Code terminal rather than starting another server on a random port without telling the user.
- Backend code changes do not affect the running local server until the `go run .` process is restarted from its VS Code terminal.
- For settlement-related frontend/backend changes, run targeted tests first, then broader checks when practical:
  - `cd backend && go test ./internal/service`
  - `cd backend && go test ./internal/controller`
  - `cd backend && go test ./...`
  - `cd frontend/frontend && npm run type-check`
  - `cd frontend/frontend && npm run test:unit -- <target spec files>`
  - `cd frontend/frontend && npm run build` when frontend wiring or shared APIs changed
- Existing Vite large chunk warnings are not by themselves a failure. Report them, but do not treat them as blocking unless the task is about bundle size.
