from dataclasses import dataclass
from typing import Optional


KNOWN_CPS = {"bilibili", "ali", "jsy", "cnc", "bsy", "xinliu"}

# 来源：EDC运维信息表 / EDC节点，仅保留状态为“正常”的区域+业务组合。
LOCAL_NORMAL_PAIRS = {
    ("上海", "cnc"),
    ("北京", "cnc"),
    ("广东", "cnc"),
    ("湖北", "cnc"),
    ("山东", "bsy"),
    ("江苏", "bsy"),
    ("上海", "bsy"),
    ("湖北", "bsy"),
    ("北京", "bsy"),
    ("四川", "bsy"),
    ("广东", "bilibili"),
    ("上海", "bilibili"),
    ("山东", "bilibili"),
    ("江苏", "bilibili"),
    ("河南", "bilibili"),
    ("湖北", "bilibili"),
    ("四川", "bilibili"),
    ("天津", "bilibili"),
    ("北京", "bilibili"),
    ("湖南", "bilibili"),
    ("福建", "bilibili"),
    ("上海", "jsy"),
    ("广东", "jsy"),
    ("河南", "jsy"),
    ("北京", "jsy"),
    ("北京", "xinliu"),
    ("广东", "xinliu"),
    ("山东", "xinliu"),
    ("四川", "xinliu"),
}


@dataclass(frozen=True)
class Resolution:
    target: Optional[str]
    rule: str
    error: Optional[str] = None


def normalize_region(region: str) -> str:
    value = region.strip()
    replacements = {
        "广西壮族自治区": "广西",
        "内蒙古自治区": "内蒙古",
        "宁夏回族自治区": "宁夏",
        "新疆维吾尔自治区": "新疆",
        "西藏自治区": "西藏",
        "香港特别行政区": "香港",
        "澳门特别行政区": "澳门",
    }
    if value in replacements:
        return replacements[value]
    for suffix in ("省", "市"):
        if value.endswith(suffix):
            return value[: -len(suffix)]
    return value


def resolve_src_region(region: str, cp: str) -> Resolution:
    normalized_region = normalize_region(region)
    normalized_cp = cp.strip().lower()

    if normalized_cp not in KNOWN_CPS:
        return Resolution(None, "unknown_cp", f"无法识别业务: {cp}")
    if normalized_region == "广东" and normalized_cp == "ali":
        return Resolution("广东", "guangdong_ali")
    if normalized_cp == "ali":
        return Resolution("北京", "ali_other_beijing")
    if normalized_region == "江苏" and normalized_cp in {"jsy", "cnc"}:
        return Resolution("上海", "jiangsu_jsy_cnc_shanghai")
    if normalized_region == "吉林" and normalized_cp == "bilibili":
        return Resolution("北京", "jilin_bilibili_beijing")
    if (normalized_region, normalized_cp) in LOCAL_NORMAL_PAIRS:
        return Resolution(region, "local_normal_node")
    if normalized_cp == "bilibili":
        return Resolution("天津", "bilibili_fallback_tianjin")
    return Resolution("北京", "recognized_fallback_beijing")
