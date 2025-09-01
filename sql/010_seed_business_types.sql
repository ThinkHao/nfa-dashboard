-- 业务类型表数据种子：确保可用的业务对象类型
-- 幂等：使用 ON DUPLICATE KEY UPDATE，重复执行安全

INSERT INTO business_types (code, name, description, enabled) VALUES
  ('customer', '客户', '客户费用归属', 1),
  ('line_provider', '线路提供商', '线路费用归属', 1),
  ('node', '节点', '节点/EDC费用归属', 1),
  ('sales', '销售', '提成/激励归属', 1)
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  description = VALUES(description),
  enabled = VALUES(enabled);
