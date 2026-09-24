# EDC/NFA 联合比较

比较组把一个或多个 EDC 实体映射到 NFA 的 `nfa_school.src_region + cp` 业务范围。`region` 仍表示院校归属地区，不参与映射匹配。映射成员使用 `valid_from`/`valid_to` 保留历史口径，查询时按时间桶判断成员是否生效。

当前示例口径：

| 比较组 | EDC 成员 | NFA 条件 |
| --- | --- | --- |
| BJ-Bilibili | `BJ-Bilibili` | `src_region=北京市`, `cp=bilibili` |
| SH-jinshan | `SH-jinshan-01`, `SH-jinshan-02` | `src_region=上海市`, `cp=jinshan` |

迁移后先只读确认实体 ID，再写入映射。不要按名称前缀自动加入成员：

```sql
SELECT id, edc_name, display_name, enabled, is_backup
FROM edc_entities
WHERE edc_name IN ('BJ-Bilibili', 'SH-jinshan-01', 'SH-jinshan-02')
ORDER BY edc_name, id;
```

确认实体 ID 后，以业务组 ID 写入 `edc_nfa_comparison_groups`，再写入 `edc_nfa_comparison_group_members`。历史更名或切换时关闭旧成员的 `valid_to`，新增成员设置 `valid_from`，不要覆盖历史记录。

比较接口按两侧原始采样间隔独立换算：EDC 使用 `service_size`（bytes/5m，`×8/300/1e6`），NFA 使用 `total_recv`（bytes/60s，`×8/60/1e6`）。接口响应会返回这两个间隔，避免把原始值直接当成同一单位比较。

查询先分别按 5 分钟时间桶聚合，再在内存按时间桶合并，避免两侧明细表直接多对多关联。返回中的 `status` 用于区分双侧都有采样、EDC 缺失、NFA 缺失和双侧缺失；零值采样不再被误判成缺失。

映射维护页位于“EDC/NFA 映射”，使用已有的 `traffic.scope.manage` 权限。页面提供新增、编辑、启停、成员多选和可选生效区间，保存前展示预览；后端事务替换组成员，并拒绝备份/禁用实体、重复成员和跨比较组的时间区间重叠。写入接口为：

- `GET /api/v2/edc-nfa/mappings`
- `GET /api/v2/edc-nfa/mapping-entities`
- `POST /api/v2/edc-nfa/mappings`
- `PUT /api/v2/edc-nfa/mappings/:id`
- `PUT /api/v2/edc-nfa/mappings/:id/enabled`

比较页继续使用启用中的映射；操作日志由统一审计中间件记录。比较明细支持前端分页，图表 tooltip 展示两位小数和 `NFA/EDC` 百分比。时间范围复用统一日期组件，日期选择默认从 `00:00:00` 到 `23:59:59`；查询使用可取消的项目级按钮。单次查询最多 366 天，后端按 7 天窗口分段、最多 8 个并发任务聚合，表格保留完整 5 分钟点，图表在点数超过 5000 时启用 LTTB 抽样。匹配按节点源区域、CP 及用户可见院校范围进行。
