#!/usr/bin/env python3
"""Guard rails for project SQL migrations.

This script centralizes the repository rules around incremental migrations:
- New incremental migrations live under sql/migrations only.
- Root sql/ keeps legacy numbered files for compatibility, but new numbered files are forbidden.
- Migration numbers are strictly increasing three-digit prefixes.
"""

from __future__ import annotations

import argparse
import re
import shutil
import sys
from dataclasses import dataclass
from pathlib import Path


LEGACY_ROOT_NUMERIC_SQL = {
    "001_auth_users.sql",
    "002_auth_roles.sql",
    "003_auth_permissions.sql",
    "004_auth_user_roles.sql",
    "005_auth_role_permissions.sql",
    "006_operation_logs.sql",
    "007_seed_auth.sql",
    "008_settlement_rates.sql",
    "009_business_types.sql",
    "010_seed_business_types.sql",
    "011_init_final_customer_rates.sql",
    "012_customer_fields_and_rules.sql",
    "013_alter_rate_final_customer_add_sync_meta.sql",
    "014_create_rate_sync_config.sql",
    "015_create_rate_sync_audits.sql",
    "016_alter_rate_customer_add_fee_mode.sql",
    "016_user_schools.sql",
    "017_alter_rate_customer_add_general_fee_owner_id.sql",
    "017_alter_users_add_alias.sql",
    "018_create_settlement_formulas.sql",
    "019_add_settlement_formula_permissions.sql",
    "020_create_settlement_results.sql",
    "021_rate_discount_and_channel_fields.sql",
    "022_alter_settlement_customer_add_channel_and_trace.sql",
    "023_alter_settlement_customer_add_indexes.sql",
    "024_alter_settlement_config_add_flags.sql",
    "025_create_index_for_school_traffic.sql",
    "026_create_school_traffic_5m.sql",
    "027_create_settlement_customer_monthly.sql",
    "028_add_increment_fields_to_rates_and_settlement_customer.sql",
    "029_add_increment_fields_to_settlement_customer_monthly.sql",
    "030_create_customer_filter_rules.sql",
    "031_create_traffic_scope_rules.sql",
    "032_upgrade_traffic_scope_rule_groups.sql",
    "20250813_add_alias_to_users.sql",
}

MIGRATION_RE = re.compile(r"^(?P<version>\d{3})_(?P<name>.+)\.sql$")
ROOT_NUMERIC_RE = re.compile(r"^\d+_.+\.sql$")
CONTRACT_RE = re.compile(r"^--\s*contract:\s*(?P<value>.+?)\s*$", re.IGNORECASE)
SCHEMA_CHANGE_RE = re.compile(r"\bCREATE\s+(?!TEMPORARY\b)TABLE\b|\bALTER\s+TABLE\b", re.IGNORECASE)
COLUMN_PRECHECK_RE = re.compile(
    r"TABLE_NAME='(?P<table>[^']+)'.*?COLUMN_NAME='(?P<column>[^']+)'.*?\|",
    re.IGNORECASE | re.DOTALL,
)
TABLE_NAME_RE = re.compile(r"TABLE_NAME='(?P<table>[^']+)'", re.IGNORECASE)
CHECK_ENTRY_RE = re.compile(r'"(?P<entry>SELECT COUNT\(\*\).*?\|[^"]+)"', re.IGNORECASE)


@dataclass(frozen=True)
class Contract:
    tables: tuple[str, ...]
    columns: tuple[str, ...]
    explicit_none: bool = False


@dataclass(frozen=True)
class Layout:
    repo_root: Path
    sql_dir: Path
    migrations_dir: Path
    offline_deploy: Path


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Manage and validate nfa-dashboard SQL migrations."
    )
    parser.add_argument(
        "--repo-root",
        type=Path,
        default=Path(__file__).resolve().parents[1],
        help="Repository root. Defaults to the parent of scripts/.",
    )

    subparsers = parser.add_subparsers(dest="command", required=True)

    create_parser = subparsers.add_parser(
        "create",
        help="Create a new migration template under sql/migrations.",
    )
    create_parser.add_argument("title", help="Human-readable migration title.")

    adopt_parser = subparsers.add_parser(
        "adopt",
        help="Move an existing SQL file into sql/migrations using the next version.",
    )
    adopt_parser.add_argument("title", help="Human-readable migration title.")
    adopt_parser.add_argument(
        "--source",
        type=Path,
        required=True,
        help="Existing SQL file to move into sql/migrations.",
    )

    subparsers.add_parser(
        "check",
        help="Validate migration numbering and forbid new numbered SQL in the root sql directory.",
    )
    return parser.parse_args()


def build_layout(repo_root: Path) -> Layout:
    repo_root = repo_root.resolve()
    return Layout(
        repo_root=repo_root,
        sql_dir=repo_root / "sql",
        migrations_dir=repo_root / "sql" / "migrations",
        offline_deploy=repo_root / "scripts" / "offline-deploy.sh",
    )


def ensure_layout(layout: Layout) -> None:
    if not layout.sql_dir.is_dir():
        raise SystemExit(f"sql directory not found: {layout.sql_dir}")
    layout.migrations_dir.mkdir(parents=True, exist_ok=True)


def slugify_title(title: str) -> str:
    slug = re.sub(r"[^a-z0-9]+", "-", title.strip().lower())
    slug = slug.strip("-")
    if not slug:
        raise SystemExit("Migration title must contain letters or digits.")
    return slug


def migration_files(layout: Layout) -> list[Path]:
    return sorted(
        path
        for path in layout.migrations_dir.iterdir()
        if path.is_file() and MIGRATION_RE.match(path.name)
    )


def next_migration_version(layout: Layout) -> int:
    versions = [int(MIGRATION_RE.match(path.name).group("version")) for path in migration_files(layout)]
    return (max(versions) if versions else 0) + 1


def destination_path(layout: Layout, title: str) -> Path:
    version = next_migration_version(layout)
    if version > 999:
        raise SystemExit("Migration version exceeded 999. Please extend the naming scheme.")
    return layout.migrations_dir / f"{version:03d}_{slugify_title(title)}.sql"


def create_template_body(title: str) -> str:
    heading = title.strip()
    return (
        f"-- Migration: {heading}\n"
        "-- contract: none\n"
        "-- Replace `none` with one or more contract lines when this migration adds runtime-required schema.\n"
        "-- contract: table=example_table\n"
        "-- contract: column=example_table.example_column\n"
        "-- Add idempotent SQL here.\n"
        "-- If this migration introduces runtime-required tables or columns,\n"
        "-- update scripts/offline-deploy.sh assert_db_schema() before release.\n\n"
    )


def relative_to_repo(layout: Layout, path: Path) -> str:
    try:
        return str(path.resolve().relative_to(layout.repo_root))
    except ValueError:
        return str(path.resolve())


def print_precheck_hint(layout: Layout) -> None:
    rel = relative_to_repo(layout, layout.offline_deploy)
    print(
        "Reminder: if this migration adds runtime-required tables or columns, "
        f"update {rel} assert_db_schema()."
    )


def cmd_create(layout: Layout, title: str) -> int:
    dest = destination_path(layout, title)
    dest.write_text(create_template_body(title), encoding="utf-8")
    print(f"Created migration: {relative_to_repo(layout, dest)}")
    print_precheck_hint(layout)
    return 0


def cmd_adopt(layout: Layout, title: str, source: Path) -> int:
    source = source.resolve()
    if not source.is_file():
        raise SystemExit(f"Source SQL file not found: {source}")
    dest = destination_path(layout, title)
    shutil.move(str(source), str(dest))
    print(f"Adopted migration into: {relative_to_repo(layout, dest)}")
    print_precheck_hint(layout)
    return 0


def find_duplicate_versions(layout: Layout) -> dict[str, list[str]]:
    duplicates: dict[str, list[str]] = {}
    buckets: dict[str, list[str]] = {}
    for path in migration_files(layout):
        version = MIGRATION_RE.match(path.name).group("version")
        buckets.setdefault(version, []).append(path.name)
    for version, names in buckets.items():
        if len(names) > 1:
            duplicates[version] = names
    return duplicates


def find_unapproved_root_numeric_sql(layout: Layout) -> list[str]:
    offenders = []
    for path in sorted(layout.sql_dir.iterdir()):
        if not path.is_file():
            continue
        if not ROOT_NUMERIC_RE.match(path.name):
            continue
        if path.name not in LEGACY_ROOT_NUMERIC_SQL:
            offenders.append(path.name)
    return offenders


def parse_contracts(path: Path) -> Contract | None:
    tables: list[str] = []
    columns: list[str] = []
    explicit_none = False

    for raw_line in path.read_text(encoding="utf-8").splitlines():
        line = raw_line.strip()
        if not line:
            continue
        if not line.startswith("--"):
            break

        match = CONTRACT_RE.match(line)
        if not match:
            continue

        value = match.group("value").strip()
        if value.lower() == "none":
            explicit_none = True
            continue
        if value.lower().startswith("table="):
            tables.append(value.split("=", 1)[1].strip())
            continue
        if value.lower().startswith("column="):
            columns.append(value.split("=", 1)[1].strip())
            continue
        raise SystemExit(f"Unsupported contract declaration in {path.name}: {value}")

    if not tables and not columns and not explicit_none:
        return None
    if explicit_none and (tables or columns):
        raise SystemExit(f"Invalid contract declarations in {path.name}: `none` cannot be mixed with table/column contracts.")
    return Contract(tables=tuple(tables), columns=tuple(columns), explicit_none=explicit_none)


def needs_contract(path: Path) -> bool:
    return bool(SCHEMA_CHANGE_RE.search(path.read_text(encoding="utf-8")))


def parse_prechecks(layout: Layout) -> tuple[set[str], set[str]]:
    content = layout.offline_deploy.read_text(encoding="utf-8")
    table_checks: set[str] = set()
    column_checks: set[str] = set()
    for entry_match in CHECK_ENTRY_RE.finditer(content):
        entry = entry_match.group("entry")
        column_match = COLUMN_PRECHECK_RE.search(entry)
        if column_match:
            column_checks.add(f"{column_match.group('table')}.{column_match.group('column')}")
            continue
        table_match = TABLE_NAME_RE.search(entry)
        if table_match:
            table_checks.add(table_match.group("table"))
    return table_checks, column_checks


def validate_contracts(layout: Layout) -> list[str]:
    errors: list[str] = []
    table_checks, column_checks = parse_prechecks(layout)

    for path in migration_files(layout):
        contract = parse_contracts(path)
        if needs_contract(path) and contract is None:
            errors.append(
                f"{path.name} has schema changes but is missing contract declaration. "
                "Add `-- contract: none` or explicit runtime contracts at the top of the file."
            )
            continue
        if contract is None or contract.explicit_none:
            continue

        missing_tables = [table for table in contract.tables if table not in table_checks]
        missing_columns = [column for column in contract.columns if column not in column_checks]
        missing = [f"table={table}" for table in missing_tables] + [f"column={column}" for column in missing_columns]
        if missing:
            errors.append(
                f"Missing offline precheck coverage for {path.name}: {', '.join(missing)}"
            )

    return errors


def cmd_check(layout: Layout) -> int:
    errors: list[str] = []

    duplicates = find_duplicate_versions(layout)
    if duplicates:
        formatted = ", ".join(
            f"{version}: {', '.join(names)}" for version, names in sorted(duplicates.items())
        )
        errors.append(f"Duplicate migration versions found in sql/migrations: {formatted}")

    offenders = find_unapproved_root_numeric_sql(layout)
    if offenders:
        errors.append(
            "Root sql directory contains numbered migrations outside the approved legacy allowlist: "
            + ", ".join(offenders)
        )

    errors.extend(validate_contracts(layout))

    if errors:
        print("SQL migration guard failed:")
        for error in errors:
            print(f"- {error}")
        return 1

    print("OK: SQL migration guard passed.")
    return 0


def main() -> int:
    args = parse_args()
    layout = build_layout(args.repo_root)
    ensure_layout(layout)

    if args.command == "create":
        return cmd_create(layout, args.title)
    if args.command == "adopt":
        return cmd_adopt(layout, args.title, args.source)
    if args.command == "check":
        return cmd_check(layout)
    raise SystemExit(f"Unsupported command: {args.command}")


if __name__ == "__main__":
    sys.exit(main())
