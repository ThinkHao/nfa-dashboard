-- 一次性初始化脚本：从 rate_customer 初始化/同步 rate_final_customer（保护手工 config 记录）
-- 可幂等重复执行

START TRANSACTION;

INSERT INTO rate_final_customer
  (region, cp, school_name, fee_type,
   customer_fee, customer_fee_owner_id,
   network_line_fee, network_line_fee_owner_id,
   created_at, updated_at)
SELECT
  rc.region,
  rc.cp,
  COALESCE(rc.school_name, 'not_a_school') AS school_name,
  'auto' AS fee_type,
  rc.customer_fee,
  rc.customer_fee_owner_id,
  rc.network_line_fee,
  rc.network_line_fee_owner_id,
  NOW(), NOW()
FROM rate_customer rc
ON DUPLICATE KEY UPDATE
  fee_type = IF(rate_final_customer.fee_type = 'config', rate_final_customer.fee_type, VALUES(fee_type)),
  customer_fee = IF(rate_final_customer.fee_type = 'config', rate_final_customer.customer_fee, VALUES(customer_fee)),
  customer_fee_owner_id = IF(rate_final_customer.fee_type = 'config', rate_final_customer.customer_fee_owner_id, VALUES(customer_fee_owner_id)),
  network_line_fee = IF(rate_final_customer.fee_type = 'config', rate_final_customer.network_line_fee, VALUES(network_line_fee)),
  network_line_fee_owner_id = IF(rate_final_customer.fee_type = 'config', rate_final_customer.network_line_fee_owner_id, VALUES(network_line_fee_owner_id)),
  updated_at = NOW();

COMMIT;
