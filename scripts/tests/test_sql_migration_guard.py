import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


SCRIPT_PATH = Path(__file__).resolve().parents[1] / "sql_migration_guard.py"


class SqlMigrationGuardTests(unittest.TestCase):
    def make_repo(self) -> Path:
        repo = Path(tempfile.mkdtemp(prefix="sql-guard-"))
        (repo / "sql" / "migrations").mkdir(parents=True)
        (repo / "scripts").mkdir(parents=True)
        (repo / "scripts" / "offline-deploy.sh").write_text(
            "#!/usr/bin/env bash\nassert_db_schema() {\n  local checks=(\n  )\n}\n",
            encoding="utf-8",
        )
        return repo

    def run_guard(self, repo: Path, *args: str) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [sys.executable, str(SCRIPT_PATH), "--repo-root", str(repo), *args],
            text=True,
            capture_output=True,
            check=False,
        )

    def test_create_generates_next_migration_file(self) -> None:
        repo = self.make_repo()
        (repo / "sql" / "migrations" / "032_existing_feature.sql").write_text("-- existing\n", encoding="utf-8")

        result = self.run_guard(repo, "create", "add traffic scope groups")

        self.assertEqual(result.returncode, 0, result.stderr)
        created = repo / "sql" / "migrations" / "033_add-traffic-scope-groups.sql"
        self.assertTrue(created.exists())
        self.assertIn("offline-deploy.sh", result.stdout)
        self.assertIn("-- contract: none", created.read_text(encoding="utf-8"))

    def test_adopt_moves_source_sql_into_migrations_directory(self) -> None:
        repo = self.make_repo()
        source = repo / "sql" / "draft_scope_rules.sql"
        source.write_text("CREATE TABLE demo(id bigint);\n", encoding="utf-8")

        result = self.run_guard(repo, "adopt", "scope rules", "--source", str(source))

        self.assertEqual(result.returncode, 0, result.stderr)
        created = repo / "sql" / "migrations" / "001_scope-rules.sql"
        self.assertTrue(created.exists())
        self.assertFalse(source.exists())
        self.assertEqual(created.read_text(encoding="utf-8"), "CREATE TABLE demo(id bigint);\n")

    def test_check_fails_for_duplicate_migration_versions(self) -> None:
        repo = self.make_repo()
        migrations = repo / "sql" / "migrations"
        (migrations / "010_first.sql").write_text("-- one\n", encoding="utf-8")
        (migrations / "010_second.sql").write_text("-- two\n", encoding="utf-8")

        result = self.run_guard(repo, "check")

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Duplicate migration versions", result.stdout)

    def test_check_fails_for_unapproved_root_numeric_sql(self) -> None:
        repo = self.make_repo()
        (repo / "sql" / "migrations" / "001_initial.sql").write_text("-- ok\n", encoding="utf-8")
        (repo / "sql" / "033_should_not_live_here.sql").write_text("-- bad\n", encoding="utf-8")

        result = self.run_guard(repo, "check")

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Root sql directory contains numbered migrations", result.stdout)

    def test_check_fails_when_schema_change_has_no_contract(self) -> None:
        repo = self.make_repo()
        (repo / "sql" / "migrations" / "001_create_demo.sql").write_text(
            "CREATE TABLE demo(id bigint);\n",
            encoding="utf-8",
        )

        result = self.run_guard(repo, "check")

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("missing contract declaration", result.stdout)

    def test_check_fails_when_runtime_contract_is_not_prechecked(self) -> None:
        repo = self.make_repo()
        (repo / "sql" / "migrations" / "001_create_demo.sql").write_text(
            "-- contract: table=demo\n"
            "CREATE TABLE demo(id bigint);\n",
            encoding="utf-8",
        )

        result = self.run_guard(repo, "check")

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Missing offline precheck coverage", result.stdout)

    def test_check_passes_when_runtime_contract_is_prechecked(self) -> None:
        repo = self.make_repo()
        (repo / "scripts" / "offline-deploy.sh").write_text(
            "#!/usr/bin/env bash\n"
            "assert_db_schema() {\n"
            "  local checks=(\n"
            "    \"SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='demo';|demo 表缺失\"\n"
            "    \"SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='demo' AND COLUMN_NAME='name';|demo.name 列缺失\"\n"
            "  )\n"
            "}\n",
            encoding="utf-8",
        )
        (repo / "sql" / "migrations" / "001_create_demo.sql").write_text(
            "-- contract: table=demo\n"
            "-- contract: column=demo.name\n"
            "CREATE TABLE demo(id bigint, name varchar(32));\n",
            encoding="utf-8",
        )

        result = self.run_guard(repo, "check")

        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
