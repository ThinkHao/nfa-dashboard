import unittest

from scripts.nfa_school_src_region import LOCAL_NORMAL_PAIRS, resolve_src_region


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


if __name__ == "__main__":
    unittest.main()
