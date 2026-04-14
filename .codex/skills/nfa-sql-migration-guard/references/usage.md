# Usage

## Create a new migration shell

```bash
python scripts/sql_migration_guard.py create "add traffic scope groups"
```

Expected result:

- Creates `sql/migrations/<next>_add-traffic-scope-groups.sql`
- Includes a `-- contract: none` placeholder at the top
- Prints a reminder to review `scripts/offline-deploy.sh` prechecks

## Adopt an existing SQL file

```bash
python scripts/sql_migration_guard.py adopt "traffic scope bootstrap" --source sql/draft_scope_rules.sql
```

Expected result:

- Moves the source file into `sql/migrations/<next>_traffic-scope-bootstrap.sql`
- Removes the original misplaced file

## Validate repository state

```bash
python scripts/sql_migration_guard.py check
pwsh scripts/check-sql-migrations.ps1
```

Failure cases:

- Duplicate `NNN_*.sql` numbers inside `sql/migrations`
- New numbered SQL files under the root `sql/` directory that are not part of the approved legacy allowlist
- Schema-changing migration files without an explicit contract header
- Runtime contract declarations that are not covered by `scripts/offline-deploy.sh` prechecks
