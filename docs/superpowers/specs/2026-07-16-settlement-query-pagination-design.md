# 院校结算查询分页修复设计

- 日期：2026-07-16
- 状态：已确认方案，待规格复核
- 涉及仓库：`nfa-dashboard`

## 1. 问题与根因

“院校结算查询”明细视图请求后端分页接口后，页面将 `pagination.total` 设置成当前页经过月份裁剪后的行数 `clipped.length`，没有采用后端响应的总记录数 `res.total`。

因此后端即使返回 `{ items: 10 条, total: 67 }`，页面也会把总数写成 10；切换到每页 20 条时则写成 20。Element Plus 分页器据此判断只有一页，无法进入后续页。

## 2. 目标

- 标准分页响应必须保留服务端 `total`。
- 兼容历史数组响应，不影响现有接口兼容性。
- 提供一个小型公共分页响应解析器，供后续分页页面复用。
- 只迁移当前存在问题的页面，不批量改写已经正确工作的分页页面。

## 3. 方案

新增通用纯函数 `normalizePaginatedResponse<T>(response)`，统一输出：

```ts
type NormalizedPaginatedResponse<T> = {
  items: T[]
  total: number
}
```

解析规则：

1. 响应为数组：`items` 为该数组，`total` 为数组长度。
2. 响应为对象且 `items` 为数组：`items` 使用对象字段；当 `total` 是有效非负数时使用该值，否则回退到 `items.length`。
3. 其他响应：返回 `{ items: [], total: 0 }`。

函数只负责规范化响应形状，不处理分页参数、业务过滤或请求发送。

## 4. 页面接入

`SettlementUserQueryView.vue` 的 `fetchRows` 使用公共解析器：

1. 解析接口响应得到当前页 `items` 和服务端 `total`。
2. 保留现有 `clipRowsBySelectedMonths(items)` 防御性裁剪。
3. 表格行仍使用裁剪后的当前页数据。
4. `pagination.total` 使用解析器返回的 `total`，不得再使用当前页行数。

月份请求边界本身已经限制了后端结果，因此正常情况下防御性裁剪不会剔除当前页记录。即使历史数组响应没有服务端总数，也继续以数组长度兼容旧行为。

## 5. 其他页面审计结论

- 客户费率、节点费率、折扣规则、系统用户、角色、权限、操作日志等标准分页页面已使用 `res.total`，不需要迁移。
- `SchoolsView` 对 `{ items, total }` 使用 `total`，数组响应才使用数组长度，行为正确。
- `TrafficView` 的 `total = withBps.length` 属于前端获取并过滤结果的客户端分页语义，不应机械替换。
- `SettlementDataTab` 的全量导出循环已经读取服务端 `total` 并据此翻页，行为正确。

本次不对这些页面做无收益的机械重构。公共解析器作为后续新增或修改分页页面的推荐入口。

## 6. 测试

先写失败测试，再实现解析器：

1. `{ items: 10 条, total: 67 }` 返回 `total: 67`，证明不会再被当前页长度覆盖。
2. 数组响应返回数组长度作为总数。
3. 缺失或非法 `total` 时回退到 `items.length`。
4. 非法响应返回空数组和 0。

页面接线后运行：

```bash
cd frontend/frontend
npm run test:unit -- src/utils/__tests__/pagination.spec.ts src/views/__tests__/settlement-query-filter-utils.spec.ts src/views/__tests__/settlement-user-query-utils.spec.ts
npm run type-check
npm run build
```

全量测试在 `TZ=UTC` 下运行，以规避仓库中既有 `UnifiedDateRange.spec.ts` 对运行时区敏感的断言。

## 7. 不做

- 不修改后端分页接口或 SQL。
- 不改按月列视图、金额聚合或导出逻辑。
- 不批量迁移所有分页页面。
- 不修复与本需求无关的 `UnifiedDateRange` 时区测试。
