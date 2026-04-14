# SQL Migrations Convention

## File Naming
- Incremental migrations must live under `sql/migrations/`.
- Use `NNN_description.sql` where `NNN` is a zero-padded, strictly increasing integer.
- One migration per file; never reuse an existing number.
- Historical numbered files in the root `sql/` directory are legacy compatibility artifacts. Do not add new numbered files there.

## Runtime Contract
- Any migration with schema-changing SQL (`CREATE TABLE`, `ALTER TABLE`, etc.) must declare a contract at the top of the file.
- Use `-- contract: none` when the change is not required for runtime startup checks.
- Use `-- contract: table=<table_name>` for runtime-required tables.
- Use `-- contract: column=<table_name>.<column_name>` for runtime-required columns.
- Contracted runtime objects must also be present in `scripts/offline-deploy.sh` `assert_db_schema()`.

## Execution Order
- Execute numeric migration files in ascending order.
- Ignore non-numeric bootstrap files (`nfa_*.sql`) during incremental upgrades.

## Validation
- Use `python scripts/sql_migration_guard.py create "<title>"` to create a new migration shell.
- Use `python scripts/sql_migration_guard.py adopt "<title>" --source <path>` to move an existing SQL draft into `sql/migrations`.
- Run `python scripts/sql_migration_guard.py check` or `scripts/check-sql-migrations.ps1` before PR and release to detect duplicate migration numbers or misplaced numbered SQL files.
- If a migration adds runtime-required tables or columns, update `scripts/offline-deploy.sh` `assert_db_schema()` in the same change.
