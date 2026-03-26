# Engineering Guidelines

## CI Gates
- Frontend pipeline must run: `npm run build`, `npm run test:unit -- --run`, `npm run lint`.
- Backend pipeline must run: `go test ./...` before build/release.
- Keep CI runtime versions aligned with lockfiles/modules (`frontend/package.json`, `backend/go.mod`).

## Layering
- `controller` only handles HTTP protocol concerns.
- `service` owns orchestration and business rules.
- `repository` is the only layer that talks to persistence.

## Deployment
- Use prebuilt artifacts in release bundles.
- Do not edit source code or run `npm install` on production target hosts during `update`.
- Prefer immutable release packages.

## SQL Migrations
- Follow numeric incremental naming (`NNN_description.sql`).
- Run `scripts/check-sql-migrations.ps1` in release checks.
