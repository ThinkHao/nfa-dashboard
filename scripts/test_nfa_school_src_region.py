import unittest
from unittest.mock import patch

from scripts.nfa_school_src_region import (
    LOCAL_NORMAL_PAIRS,
    build_preview,
    execute_backfill,
    main,
    resolve_src_region,
)


class RuleTests(unittest.TestCase):
    def test_approved_rule_matrix(self):
        cases = [
            ("陕西", "bilibili", "天津", "bilibili_fallback_tianjin"),
            ("陕西", "ali", "北京", "ali_other_beijing"),
            ("陕西", "jsy", "北京", "recognized_fallback_beijing"),
            ("广东", "ali", "广东", "guangdong_ali"),
            ("江苏", "jsy", "上海", "jiangsu_jsy_cnc_shanghai"),
            ("江苏", "cnc", "上海", "jiangsu_jsy_cnc_shanghai"),
            ("吉林", "bilibili", "北京", "jilin_bilibili_beijing"),
            ("山东省", "bsy", "山东省", "local_normal_node"),
        ]

        for region, cp, target, rule in cases:
            with self.subTest(region=region, cp=cp):
                result = resolve_src_region(region, cp)
                self.assertEqual(target, result.target)
                self.assertEqual(rule, result.rule)
                self.assertIsNone(result.error)

    def test_unknown_cp_is_error(self):
        result = resolve_src_region("陕西", "unknown")

        self.assertIsNone(result.target)
        self.assertEqual("unknown_cp", result.rule)
        self.assertEqual("无法识别业务: unknown", result.error)

    def test_confirmed_database_cp_mappings(self):
        cases = [
            ("陕西", "jinshan", "北京", "recognized_fallback_beijing"),
            ("广东省", "baidu", "广东省", "local_normal_node"),
            ("陕西", "se", "北京", "recognized_fallback_beijing"),
        ]
        for region, cp, target, rule in cases:
            with self.subTest(region=region, cp=cp):
                result = resolve_src_region(region, cp)
                self.assertEqual(target, result.target)
                self.assertEqual(rule, result.rule)
                self.assertFalse(result.skipped)

    def test_explicitly_ignored_cps_are_skipped(self):
        for cp in ("dianbo", "zhibo", "NULL"):
            with self.subTest(cp=cp):
                result = resolve_src_region("陕西", cp)
                self.assertIsNone(result.target)
                self.assertEqual("ignored_cp", result.rule)
                self.assertTrue(result.skipped)
                self.assertIsNone(result.error)

    def test_offline_and_planned_nodes_are_not_local(self):
        self.assertNotIn(("吉林", "xinliu"), LOCAL_NORMAL_PAIRS)
        self.assertNotIn(("河南", "xinliu"), LOCAL_NORMAL_PAIRS)
        self.assertNotIn(("上海", "xinliu"), LOCAL_NORMAL_PAIRS)
        self.assertNotIn(("广东", "iqiyi"), LOCAL_NORMAL_PAIRS)


class PreviewTests(unittest.TestCase):
    def test_build_preview_buckets_and_details(self):
        rows = [
            {
                "id": 1,
                "school_id": "s1",
                "school_name": "陕西院校A",
                "region": "陕西",
                "cp": "bilibili",
                "src_region": None,
            },
            {
                "id": 2,
                "school_id": "s2",
                "school_name": "广东院校B",
                "region": "广东",
                "cp": "ali",
                "src_region": "广东",
            },
            {
                "id": 3,
                "school_id": "s3",
                "school_name": "未知院校C",
                "region": "陕西",
                "cp": "unknown",
                "src_region": None,
            },
        ]

        preview = build_preview(rows)

        self.assertEqual(3, preview["summary"]["total"])
        self.assertEqual(1, preview["summary"]["will_update"])
        self.assertEqual(1, preview["summary"]["unchanged"])
        self.assertEqual(1, preview["summary"]["errors"])
        self.assertEqual(1, len(preview["updates"]))
        self.assertEqual(1, len(preview["unchanged"]))
        self.assertEqual(1, len(preview["errors"]))
        required = {
            "id",
            "school_name",
            "region",
            "cp",
            "current_src_region",
            "target_src_region",
            "rule",
        }
        for detail in preview["details"]:
            self.assertTrue(required.issubset(detail))

    def test_ignored_cp_has_its_own_bucket(self):
        preview = build_preview(
            [
                {
                    "id": 9,
                    "school_id": "s9",
                    "school_name": "忽略院校",
                    "region": "陕西",
                    "cp": "NULL",
                    "src_region": None,
                }
            ]
        )

        self.assertEqual(1, preview["summary"]["skipped"])
        self.assertEqual(0, preview["summary"]["errors"])
        self.assertEqual(1, len(preview["skipped"]))


class FakeCursor:
    def __init__(self, connection):
        self.connection = connection
        self._rows = []

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc, traceback):
        return False

    def execute(self, sql, params=None):
        normalized = " ".join(sql.split())
        self.connection.calls.append(("execute", normalized, params))
        if normalized.startswith("SELECT id, school_id"):
            self._rows = self.connection.row_reads.pop(0)
        elif normalized.startswith("SELECT COUNT(*) AS row_count"):
            self._rows = [{"row_count": self.connection.backup_count}]

    def executemany(self, sql, params):
        normalized = " ".join(sql.split())
        values = list(params)
        self.connection.calls.append(("executemany", normalized, values))
        if self.connection.fail_updates:
            raise RuntimeError("update failed")

    def fetchall(self):
        return self._rows

    def fetchone(self):
        return self._rows[0]


class FakeConnection:
    def __init__(self, row_reads, backup_count=1, fail_updates=False):
        self.row_reads = list(row_reads)
        self.backup_count = backup_count
        self.fail_updates = fail_updates
        self.calls = []

    def cursor(self):
        return FakeCursor(self)

    def begin(self):
        self.calls.append(("begin",))

    def commit(self):
        self.calls.append(("commit",))

    def rollback(self):
        self.calls.append(("rollback",))


class ExecutionTests(unittest.TestCase):
    def setUp(self):
        self.before = {
            "id": 1,
            "school_id": "s1",
            "school_name": "陕西院校A",
            "region": "陕西",
            "cp": "bilibili",
            "src_region": None,
        }
        self.after = {**self.before, "src_region": "天津"}

    def test_execute_requires_confirm_before_connecting(self):
        with patch("scripts.nfa_school_src_region.open_connection") as connect:
            with self.assertRaises(SystemExit):
                main(["--execute"])
        connect.assert_not_called()

    def test_backup_precedes_update_transaction_and_commit(self):
        connection = FakeConnection([[self.before], [self.after]])

        result = execute_backfill(connection, timestamp="20260716_120000")

        operations = [call[0] for call in connection.calls]
        self.assertEqual(
            [
                "execute",
                "execute",
                "execute",
                "execute",
                "commit",
                "begin",
                "executemany",
                "execute",
                "commit",
            ],
            operations,
        )
        self.assertIn("CREATE TABLE", connection.calls[1][1])
        self.assertIn("INSERT INTO", connection.calls[2][1])
        self.assertEqual("nfa_school_src_region_backup_20260716_120000", result["backup_table"])
        self.assertEqual(1, result["updated"])

    def test_update_failure_rolls_back(self):
        connection = FakeConnection([[self.before]], fail_updates=True)

        with self.assertRaises(RuntimeError):
            execute_backfill(connection, timestamp="20260716_120000")

        self.assertIn(("rollback",), connection.calls)
        self.assertEqual(1, connection.calls.count(("commit",)))


if __name__ == "__main__":
    unittest.main()
