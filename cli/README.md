# nfa-dashboard CLI

HTTP-based CLI for agent and operator workflows. It calls the existing backend API so JWT auth, RBAC permissions, traffic-scope filtering, and operation-log auditing stay in force.

## Build And Test

```powershell
cd cli
go test ./...
go build -o nfa-dashboard-cli.exe .
.\nfa-dashboard-cli.exe version
```

Release builds inject version metadata with Go ldflags:

```powershell
go build -ldflags "-s -w -X main.version=v1.0.0 -X main.commit=<sha> -X main.date=<utc-iso-date>" -o nfa-dashboard-cli.exe .
```

The CLI release workflow publishes common platform binaries plus `checksums.txt`:

- `nfa-dashboard-cli-windows-amd64.exe`
- `nfa-dashboard-cli-linux-amd64`
- `nfa-dashboard-cli-linux-arm64`
- `nfa-dashboard-cli-darwin-arm64`
- `nfa-dashboard-cli-darwin-amd64`

## Auth

```powershell
.\nfa-dashboard-cli.exe --base-url http://localhost:8081 auth login --username admin --password "..."
.\nfa-dashboard-cli.exe auth profile
```

For automation, prefer environment variables:

```powershell
$env:NFA_DASHBOARD_BASE_URL="http://localhost:8081"
$env:NFA_DASHBOARD_TOKEN="..."
$env:NFA_DASHBOARD_REFRESH_TOKEN="..."
```

If the target database password differs from seed SQL comments, login returns a credential-specific 401 with a reminder to verify the live account. Non-interactive callers can pass `--password-env ENV_NAME` instead of putting the password directly in the command line.

For self-signed HTTPS targets, pass `--insecure-skip-verify` explicitly:

```powershell
.\nfa-dashboard-cli.exe --base-url https://192.168.9.104:8090 --insecure-skip-verify auth login --username admin
```

## Generic API Coverage

Use raw API commands for any endpoint that does not yet have a typed command:

```powershell
.\nfa-dashboard-cli.exe api get --path /api/v1/system/permissions
.\nfa-dashboard-cli.exe api post --path /api/v1/system/permissions/sync --body "{}" --dry-run
```

By default, JSON responses are saved to a file under the temp directory and stdout prints a concise summary with key counts and file paths. Use `--json-file PATH` or `--out-dir DIR` to control where JSON is written. Use `--print-body` when an agent needs the full response on stdout.

## Typed Examples

```powershell
.\nfa-dashboard-cli.exe traffic data --query region=湖北 --query granularity=5m --svg
.\nfa-dashboard-cli.exe edc data --query region=北京市 --query cp=bilibili --svg
.\nfa-dashboard-cli.exe edc entities --query region=北京市 --query limit=20
.\nfa-dashboard-cli.exe settlement tasks list --query limit=20
.\nfa-dashboard-cli.exe settlement tasks create-node-daily95 --body '{"start_date":"2026-04-01","end_date":"2026-04-30"}' --dry-run
.\nfa-dashboard-cli.exe settlement data node --query region=北京市 --query cp=bilibili --query start_date=2026-04-01 --query end_date=2026-04-30
.\nfa-dashboard-cli.exe settlement user-panel --query channel_owner_user_id=5 --query region=北京市 --query cp=bilibili --query start_service_date="2026-04-01 00:00:00" --query end_service_date="2026-04-30 23:59:59"
.\nfa-dashboard-cli.exe settlement node-panel --query region=北京市 --query cp=bilibili --query start_date=2026-04-01 --query end_date=2026-04-30
.\nfa-dashboard-cli.exe rates customer export --download .\customer-rates.csv
.\nfa-dashboard-cli.exe rates customer import-task continue --id 42
.\nfa-dashboard-cli.exe system roles list --output table
.\nfa-dashboard-cli.exe logs export --download .\operation-logs.csv
```

All commands support `--output json|table|csv`; `json` is the summary-and-save default, while `table` and `csv` print transformed response data to stdout. Traffic data (`traffic data` for NFA, `edc data` for EDC) can also write an SVG chart with `--svg` or `--svg-file PATH`; the summary includes point count, time bounds, average, max, and 95th percentile Mbps values. NFA traffic monitor points are converted with `raw_bytes * 8 / 60 / 1_000_000`, matching the web traffic page's decimal bit-rate formatter. The `traffic_byte_unit_base` setting is for byte-size display such as B/KB/MB/GB, not Mbps/Gbps.

NFA (school) and EDC (node) traffic are two independent data links. `traffic *` queries NFA via `/api/v2/traffic`; `edc *` queries EDC via `/api/v2/edc/traffic`. EDC points expose `service_size` (服务流速) and `cache_size` (回源流速) instead of NFA's `total_recv`/`total_send`; the CLI aliases them so `edc data` reuses the same two-series chart, summary, and `*8/60/1_000_000` conversion.

`settlement user-panel` mirrors the frontend single-user settlement page. It fetches both monthly amount rows and daily rows, then writes a combined JSON file with `summary`, `panel_rows`, `monthly_rows`, and `daily_rows`. The default summary includes monthly row count, daily row count, single-user 95 unit/base, monthly 95 total, and amount totals.

`settlement node-panel` mirrors the frontend 单节点结算查询 (EDC) page. It fetches `settlement data node` (日95) plus `settlement data node-monthly` (月95) and aggregates per node+month, preferring the monthly `mbps_95` and falling back to the daily average; amounts come from `total_bill`. Node `mbps_95` is already in Mbps, so no unit conversion is applied. Node 95 tasks (`settlement tasks create-node-daily95` / `create-node-monthly95`) take a range payload and create exactly one task row for the whole range.
