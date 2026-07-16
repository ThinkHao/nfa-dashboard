# 院校结算用户下拉性能优化设计

- 日期：2026-07-16
- 状态：已确认方案，待规格复核
- 涉及仓库：`nfa-dashboard`

## 1. 问题与根因

院校结算查询页面的用户下拉使用 `/api/v1/settlement/data/customer/owner-subjects`。当前服务实现先通过 `ListAll` 读取筛选范围内最多 100,000 条完整结算记录，再在 Go 内存中遍历客户、线路、节点、渠道四个归属字段并去重用户 ID，最后查询用户名称。

页面虽然在初始化时预加载用户，但该请求排在系统设置、地区/CP/学校和重点院校请求之后串行执行。用户较早点击下拉框时，重查询可能尚未开始或仍未完成。

## 2. 现网数据实测基线

只读测试范围：页面默认服务月份 `2026-05` 至 `2026-07`，不选择地区和 CP，使用当前有效结算槽位。

| 指标 | 当前实现 | 单次扫描去重查询 |
|---|---:|---:|
| 筛选范围结算行数 | 27,209 | 27,209 |
| 返回应用层行数 | 27,209 条完整记录 | 25 个用户 ID |
| 数据库中位耗时 | 约 2,197 ms | 约 240 ms |
| 数据库加速 | 1 倍 | 约 9.2 倍 |
| 返回行数缩减 | 1 倍 | 约 1,088 倍 |

数据库以外的接口序列化和网络耗时尚未计入。实现后的目标是接口常规响应约 `0.3～0.6 秒`；该值需通过端到端实测确认，不作为未经测量的完成结论。

## 3. 保持的业务语义

用户下拉仍只显示以下范围内实际出现过的归属用户：

- 当前必选服务月份；
- 已选择的地区；
- 已选择的 CP；
- 客户费、线路费、节点费、渠道费四个归属字段的并集；
- 当前有效结算槽位；
- 现有结算数据过滤规则允许的记录。

不改为“全部系统用户”，也不引入可能过期的应用缓存。

## 4. 后端设计

在 `SettlementDataRepository` 增加轻量方法：

```go
ListDistinctOwnerUserIDs(ctx context.Context, filter map[string]interface{}) ([]uint64, error)
```

查询继续使用 `applySettlementCustomerFilters` 与当前活动槽位关联，确保地区、CP、月份和过滤规则语义一致。通过四行常量派生表与 `CASE` 把四个归属列展开为单列，在数据库内执行 `DISTINCT`，只返回有效的正整数用户 ID。

概念 SQL：

```sql
SELECT DISTINCT
  CASE owner_slots.n
    WHEN 1 THEN customer_fee_owner_id
    WHEN 2 THEN network_line_fee_owner_id
    WHEN 3 THEN node_deduction_fee_owner_id
    ELSE channel_owner_user_id
  END AS owner_id
FROM <当前结算数据源>
JOIN (
  SELECT 1 AS n UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4
) owner_slots
WHERE <现有筛选条件>
HAVING owner_id IS NOT NULL AND owner_id > 0;
```

`ListUsedOwnerSubjects` 改为调用该方法，再用现有 `userRepo.FindByIDs` 解析用户显示名称。接口 URL、返回结构、权限和前端调用方式保持不变。

`ListUsedChannelOwners` 与 `ListUsedOwnerEntities` 不在本次页面链路中，暂不扩大修改范围。

## 5. 前端设计

页面仍在设置默认月份之后预加载用户，但不再等待地区/学校和重点院校加载完成。初始化顺序调整为：

1. 加载系统流量设置；
2. 设置默认月份；
3. 并行执行地区/CP/学校、重点院校和用户选项加载。

用户选项仍按现有 query key 缓存。地区、CP 或月份变化后，首次打开下拉框时按现有规则重新加载；不新增跨筛选条件缓存或过期策略。

## 6. 测试

### 后端

- 仓库测试确认查询只选择去重 `owner_id`，并应用地区、CP、学校和日期筛选。
- 服务测试确认 `ListUsedOwnerSubjects` 使用仓库返回的 ID，而不再调用全量结算列表。
- 四个归属字段中的重复 ID 最终只返回一个用户选项。
- 空 ID 集合直接返回空选项，不执行无意义的用户查询。

### 前端

- 把初始化并行编排提取为可测试函数，确认三个加载任务同时启动而不是串行等待。
- 保留现有 query key、防重复请求和筛选变化重新加载行为。

## 7. 验证与性能验收

```bash
cd backend
go test ./internal/repository ./internal/service
go test ./...

cd ../frontend/frontend
npm run test:unit -- --run
npm run type-check
npm run build
```

实现后对同一 `2026-05` 至 `2026-07` 范围复测：

- 数据库轻量查询结果仍为 25 个用户 ID；
- 数据库查询中位耗时不高于 400 ms；
- 与旧实现从完整行提取出的 ID 集合完全一致；
- 记录接口端到端耗时，目标常规不高于 600 ms。

## 8. 不做

- 不新增数据库索引或 SQL 迁移。
- 不加入 Redis、内存缓存或缓存失效策略。
- 不改变用户下拉的范围语义。
- 不批量优化其他 owner 列表接口。
- 不改变结算查询、金额计算或分页逻辑。
