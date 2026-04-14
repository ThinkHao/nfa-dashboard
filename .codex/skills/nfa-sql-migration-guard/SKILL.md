---
name: nfa-sql-migration-guard
description: Use when working on nfa-dashboard schema changes, numbered SQL files, offline bundle migrations, schema_migrations behavior, or release-time SQL validation. Trigger this when adding tables, columns, indexes, constraints, seed data, adopting stray SQL files, or deciding where a migration should live.
---

# NFA SQL Migration Guard

Keep incremental migrations in `sql/migrations` only.

## Project Rules

- Treat `sql/migrations` as the only source of truth for incremental migrations.
- Leave legacy numbered files in `sql/` alone unless explicitly adopting one into `sql/migrations`.
- Never create a new numbered SQL file directly under `sql/`.
- If a schema change adds runtime-required tables or columns, declare them with migration header comments and update `scripts/offline-deploy.sh` `assert_db_schema()` in the same task.
- If a change is query-only and does not change schema, do not create a migration just for consistency theater.

## Workflow

1. Inspect `sql/` and `sql/migrations/` before making migration decisions.
2. Run the guard script instead of hand-picking a version number:
   - `python scripts/sql_migration_guard.py create "<title>"`
   - `python scripts/sql_migration_guard.py adopt "<title>" --source <path-to-sql>`
3. Put the actual incremental SQL into the generated file under `sql/migrations/`.
4. For schema-changing migrations, keep one explicit contract block at the top:
   - `-- contract: none` when the schema change is not runtime-required
   - `-- contract: table=<table_name>` for runtime-required tables
   - `-- contract: column=<table_name>.<column_name>` for runtime-required columns
5. Run `python scripts/sql_migration_guard.py check` or `pwsh scripts/check-sql-migrations.ps1`.
6. If the new schema is required at runtime, update `scripts/offline-deploy.sh` prechecks before finishing.

## Decision Points

- Use `create` for a brand-new migration.
- Use `adopt` when someone already drafted SQL in the wrong place and it needs to be moved into `sql/migrations`.
- Use `check` before PR, release, or whenever you touched numbered SQL files.

## Common Mistakes

- Adding `sql/033_feature.sql` and assuming CI or offline deploy will find it.
- Reusing an existing migration number because the root `sql/` directory already contains the same prefix.
- Shipping runtime schema changes without extending offline schema prechecks.

Read [references/usage.md](references/usage.md) for command examples and expected outcomes.
