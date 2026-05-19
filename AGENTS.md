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
