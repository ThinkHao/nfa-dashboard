import argparse
import json
import os
from collections import Counter
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import Any, Iterable, Optional


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


def build_preview(rows: Iterable[dict[str, Any]]) -> dict[str, Any]:
    details = []
    updates = []
    unchanged = []
    errors = []
    rule_counts: Counter[str] = Counter()

    for row in rows:
        resolution = resolve_src_region(str(row["region"]), str(row["cp"]))
        detail = {
            "id": row["id"],
            "school_id": row["school_id"],
            "school_name": row["school_name"],
            "region": row["region"],
            "cp": row["cp"],
            "current_src_region": row.get("src_region"),
            "target_src_region": resolution.target,
            "rule": resolution.rule,
            "error": resolution.error,
        }
        details.append(detail)
        if resolution.error:
            errors.append(detail)
            continue
        rule_counts[resolution.rule] += 1
        if row.get("src_region") == resolution.target:
            unchanged.append(detail)
        else:
            updates.append(detail)

    details.sort(
        key=lambda item: (
            str(item["region"]),
            str(item["cp"]),
            str(item["school_name"]),
            int(item["id"]),
        )
    )
    return {
        "generated_at": datetime.now().astimezone().isoformat(timespec="seconds"),
        "summary": {
            "total": len(details),
            "will_update": len(updates),
            "unchanged": len(unchanged),
            "errors": len(errors),
        },
        "rule_counts": dict(sorted(rule_counts.items())),
        "updates": updates,
        "unchanged": unchanged,
        "errors": errors,
        "details": details,
    }


def read_rows(connection: Any) -> list[dict[str, Any]]:
    with connection.cursor() as cursor:
        cursor.execute(
            """
            SELECT id, school_id, school_name, region, cp, src_region
            FROM nfa_school
            ORDER BY region, cp, school_name, id
            """
        )
        return list(cursor.fetchall())


def open_connection(args: argparse.Namespace) -> Any:
    try:
        import pymysql
    except ImportError as exc:
        raise RuntimeError("缺少 pymysql，请先安装数据库驱动") from exc

    return pymysql.connect(
        host=args.host,
        port=args.port,
        user=args.user,
        password=args.password,
        database=args.database,
        charset="utf8mb4",
        cursorclass=pymysql.cursors.DictCursor,
        autocommit=False,
    )


def parse_args(argv: Optional[list[str]] = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="预览或执行 nfa_school.src_region 回填；默认仅生成只读预览。"
    )
    parser.add_argument(
        "--output",
        default="outputs/nfa-school-src-region-preview.json",
        help="预览 JSON 输出路径",
    )
    parser.add_argument("--execute", action="store_true", help="执行数据库回填")
    parser.add_argument("--confirm", action="store_true", help="确认执行数据库回填")
    parser.add_argument("--host", default=os.getenv("DB_HOST", "localhost"))
    parser.add_argument("--port", type=int, default=int(os.getenv("DB_PORT", "3306")))
    parser.add_argument("--user", default=os.getenv("DB_USER", "root"))
    parser.add_argument("--password", default=os.getenv("DB_PASS", ""))
    parser.add_argument("--database", default=os.getenv("DB_NAME", "nfa"))
    return parser.parse_args(argv)


def write_preview(preview: dict[str, Any], output: str) -> Path:
    path = Path(output)
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(
        json.dumps(preview, ensure_ascii=False, indent=2) + "\n", encoding="utf-8"
    )
    return path


def main(argv: Optional[list[str]] = None) -> int:
    args = parse_args(argv)
    if args.execute:
        raise RuntimeError("执行模式尚未实现")

    connection = open_connection(args)
    try:
        preview = build_preview(read_rows(connection))
    finally:
        connection.close()
    output_path = write_preview(preview, args.output)
    summary = preview["summary"]
    print(
        "预览完成: "
        f"total={summary['total']} will_update={summary['will_update']} "
        f"unchanged={summary['unchanged']} errors={summary['errors']}"
    )
    print(f"预览文件: {output_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
