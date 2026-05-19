---
name: nfa-dashboard-cli
description: Use when operating nfa-dashboard through its project CLI: querying traffic, settlement, rates, users, roles, permissions, traffic scopes, settings, or audit logs; exporting JSON/SVG/CSV/XLSX; or calling backend HTTP APIs through the generic raw API command.
---

# nfa-dashboard CLI

Use this skill to operate `nfa-dashboard` through the project CLI. The CLI is an HTTP client, so backend JWT auth, RBAC permissions, traffic-scope filtering, and operation-log auditing remain in force.

## Core Rules

- Get the CLI in this order:
  1. If inside the `nfa-dashboard` repo, run from `cli/` with `go run .`.
  2. If `nfa-dashboard-cli` is already on `PATH`, use it after checking `nfa-dashboard-cli version`.
  3. If outside the repo and no binary exists, download a versioned release asset for the current platform and verify it with `checksums.txt`.
- Do not connect directly to MySQL for CLI write workflows. Use backend `/api/v1` or `/api/v2` through the CLI.
- Do not print access tokens or refresh tokens. Prefer `NFA_DASHBOARD_BASE_URL`, `NFA_DASHBOARD_TOKEN`, and `NFA_DASHBOARD_REFRESH_TOKEN`.
- For login, prefer `auth login --password-env ENV_NAME`. For self-signed HTTPS targets, pass `--insecure-skip-verify` explicitly.
- For non-trivial write calls, run the same command with `--dry-run` first and inspect method, path, query, body, file, and download target.
- For production operations, prefer a user-specified or fixed version. Do not silently download an unknown `latest` asset.

## CLI Availability

Release assets should use these names:

- `nfa-dashboard-cli-windows-amd64.exe`
- `nfa-dashboard-cli-linux-amd64`
- `nfa-dashboard-cli-linux-arm64`
- `nfa-dashboard-cli-darwin-arm64`
- `nfa-dashboard-cli-darwin-amd64`
- `checksums.txt`

After obtaining a binary, run `nfa-dashboard-cli version` and `nfa-dashboard-cli --help` before using it. On Linux/macOS, mark the file executable if needed.

## Discovery Workflow

1. Start with the built-in help:
   - `go run . --help`
   - `nfa-dashboard-cli --help`
   - `go run . <group> --help`
   - `nfa-dashboard-cli <group> --help`
   - `go run . settlement user-panel --help`
2. Prefer typed commands when available: `auth`, `traffic`, `settlement`, `rates`, `system`, and `logs`.
3. If no typed command exists, use raw API coverage:
   - `go run . api get --path /api/v1/...`
   - `go run . api post --path /api/v1/... --body "{}" --dry-run`
4. Let default JSON output save the response to disk and read the stdout summary. Use `--print-body` only when full JSON must be on stdout.

## Common Tasks

- Traffic trend: use `traffic data` with `--query region=...`, `--query school_name=...`, `--query cp=...`, `--query start_time=...`, `--query end_time=...`, and `--svg` when a human-readable chart is needed.
- Traffic units: NFA raw points convert to Mbps with `raw_bytes * 8 / 60 / 1_000_000`; bit-rate units are decimal 1000, matching the web traffic page.
- Single-user settlement page: use `settlement user-panel`. Do not reconstruct the result manually from channel/monthly/daily endpoints unless the user explicitly asks for investigation.
- Owner lookup before single-user settlement: do not use `system users list` to infer `channel_owner_user_id`. Query the settlement owner dropdown source instead:
  - `settlement owner-subjects --query region=... --query cp=... --query start_service_date="YYYY-MM-DD 00:00:00" --query end_service_date="YYYY-MM-DD 23:59:59"`
  - If using an older CLI without that typed command, use `api get --path /api/v1/settlement/data/customer/owner-subjects` with the same query filters.
  - Match the returned user subject by `label` such as `刘旭阳`, then pass its `id` as `channel_owner_user_id`.
  - Do not stop to ask the user for an ID until this endpoint has been tried with the same filters.
- Permission and scope checks: use `system permissions`, `system roles`, and `system traffic-scopes` commands.
- Audit verification: use `logs list` or `logs export` to check whether backend operation logs captured a CLI action.

## Reporting

When reporting results, include the command intent, target environment, key summary values, and saved JSON/SVG/download paths. If a command fails, preserve the HTTP status, backend message, and `missing` permission value if present.
