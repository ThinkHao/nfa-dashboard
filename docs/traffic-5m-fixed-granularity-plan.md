# 流量监控 5 分钟固定粒度改造方案

## 1. 目标与约束

1. 任意时间范围都必须返回 5 分钟一个点，禁止降采样到 15m/1h/day。
2. 过滤维度支持自由组合：`region`、`cp`、`school_name`（可只选其一或任意组合）。
3. 在保证灵活查询的同时维持可接受响应时间。

## 2. 核心设计

1. 新增 5 分钟聚合事实表 `nfa_school_traffic_5m`，图表查询只读该表。
2. 后端使用“查询模板路由”，不再全局固定 `FORCE INDEX` 单一索引。
3. 后端按请求时间轴补齐缺失点（5 分钟步长），保证点数完整。
4. 前端过滤支持“下拉选择 + 手动输入”。

## 3. 数据模型

建议表结构见 `sql/026_create_school_traffic_5m.sql`，核心字段：

1. `bucket_5m`: 5 分钟桶时间（主时间轴）。
2. `region`、`cp`、`school_id`、`school_name`: 业务维度。
3. `total_recv`、`total_send`: 该桶流量汇总（字节）。
4. `record_count`: 该桶汇总时包含的明细行数（用于校验）。

主键使用 `(bucket_5m, region, cp, school_id)`，保证幂等 `UPSERT`。

## 4. 写入与回填

1. 增量任务每 1~5 分钟执行一次，按 `[last_cursor, now-5m)` 做窗口聚合。
2. 使用 `INSERT ... SELECT ... GROUP BY ... ON DUPLICATE KEY UPDATE` 幂等写入。
3. 首次上线做历史回填（建议按天分批），完成后切流。

5 分钟桶计算公式：

```sql
FROM_UNIXTIME(UNIX_TIMESTAMP(create_time) - MOD(UNIX_TIMESTAMP(create_time), 300))
```

## 5. 查询接口契约（建议）

沿用 `GET /api/v2/traffic`，新增参数：

1. `group_by`: `total | cp | school`，默认 `total`。
2. `fill`: `zero | null`，默认 `zero`。
3. `strict_5m`: `1 | 0`，默认 `1`，为 `1` 时强制补齐完整时间轴。

返回结构建议：

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "granularity": "5m",
    "start_time": "2026-03-01T00:00:00+08:00",
    "end_time": "2026-03-10T00:00:00+08:00",
    "points_expected": 2593,
    "series": [
      {
        "key": "CMCC",
        "label": "CMCC",
        "points": [
          ["2026-03-01T00:00:00+08:00", 12345, 23456]
        ]
      }
    ]
  }
}
```

说明：`points_expected = floor((end-start)/300s)+1`，用于前后端校验“未丢点”。

## 6. 查询模板（不降粒度）

1. 区域 + CP 总体趋势：`GROUP BY bucket_5m`
2. 区域 + 学校查看不同 CP：`GROUP BY bucket_5m, cp`
3. 学校总体趋势：`GROUP BY bucket_5m`
4. 区域总体趋势：`GROUP BY bucket_5m`

全部模板都必须限定 `bucket_5m BETWEEN ? AND ?`，且不做时间粒度转换。

## 7. 点位补齐策略

后端收到聚合结果后执行补齐：

1. 先生成 `[start, end]` 的 5 分钟时间轴数组。
2. 再按 `series_key + bucket_5m` 建映射。
3. 缺失点按 `fill` 补 `0` 或 `null`。

这样可以避免依赖 MySQL 递归 CTE，在 MySQL 5.7/8.0 都可稳定工作。

## 8. 前端改造点

1. `region/cp/school_name` 改为 `filterable + allow-create`。
2. 学校字段支持 remote 搜索（防止一次性加载大量院校）。
3. 图表不再自行降点，直接使用后端完整 5 分钟序列。
4. 展示 `points_expected` 与实际点数，异常时提示“数据缺口”。

## 9. 性能保障

1. 查聚合表，不查明细表。
2. 查询按时间窗口分页拉取（例如 7 天/片），前端拼接，粒度不变。
3. 响应压缩（gzip/br）并收敛返回字段（仅时间与流量）。

## 10. 上线步骤

1. 发布 DDL（新增聚合表与索引）。
2. 发布聚合作业与回填任务。
3. 双读对账（老接口 vs 新聚合接口）至少 3 天。
4. 切换流量页到新查询路径。
5. 下线旧的单一 `FORCE INDEX` 查询逻辑。

## 11. 验收标准

1. 任意时间范围响应均为 5 分钟粒度。
2. 图表点数严格等于 `points_expected`。
3. 支持 `region+cp`、`region+school(+cp)` 等自由组合查询。
4. 慢查询比例明显下降，接口 P95 可控。
