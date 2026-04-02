-- 审计：rate_customer 四个归属字段中，不存在于 users 表的非法用户ID
-- 使用方式：直接执行本脚本，结果集为空即通过

SELECT
  rc.id,
  rc.region,
  rc.cp,
  rc.school_name,
  'customer_fee_owner_id' AS owner_field,
  rc.customer_fee_owner_id AS invalid_user_id
FROM rate_customer rc
LEFT JOIN users u ON u.id = rc.customer_fee_owner_id
WHERE rc.customer_fee_owner_id IS NOT NULL
  AND u.id IS NULL

UNION ALL

SELECT
  rc.id,
  rc.region,
  rc.cp,
  rc.school_name,
  'network_line_fee_owner_id' AS owner_field,
  rc.network_line_fee_owner_id AS invalid_user_id
FROM rate_customer rc
LEFT JOIN users u ON u.id = rc.network_line_fee_owner_id
WHERE rc.network_line_fee_owner_id IS NOT NULL
  AND u.id IS NULL

UNION ALL

SELECT
  rc.id,
  rc.region,
  rc.cp,
  rc.school_name,
  'general_fee_owner_id' AS owner_field,
  rc.general_fee_owner_id AS invalid_user_id
FROM rate_customer rc
LEFT JOIN users u ON u.id = rc.general_fee_owner_id
WHERE rc.general_fee_owner_id IS NOT NULL
  AND u.id IS NULL

UNION ALL

SELECT
  rc.id,
  rc.region,
  rc.cp,
  rc.school_name,
  'channel_owner_user_id' AS owner_field,
  rc.channel_owner_user_id AS invalid_user_id
FROM rate_customer rc
LEFT JOIN users u ON u.id = rc.channel_owner_user_id
WHERE rc.channel_owner_user_id IS NOT NULL
  AND u.id IS NULL

ORDER BY id, owner_field;
