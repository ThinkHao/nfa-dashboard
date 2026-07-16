import unittest

from scripts.nfa_school_src_region import (
    LOCAL_NORMAL_PAIRS,
    build_preview,
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


if __name__ == "__main__":
    unittest.main()
