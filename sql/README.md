# SQL Migrations Convention

## File Naming
- Use `NNN_description.sql` where `NNN` is a zero-padded, strictly increasing integer.
- One migration per file; never reuse an existing number.
- Historical files with duplicate numbers are kept for compatibility. New files must continue from the next unique number.

## Execution Order
- Execute numeric migration files in ascending order.
- Ignore non-numeric bootstrap files (`nfa_*.sql`) during incremental upgrades.

## Validation
- Run `scripts/check-sql-migrations.ps1` before release to detect duplicate numeric prefixes.
