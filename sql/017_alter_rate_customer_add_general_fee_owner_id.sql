-- 添加节点通用费归属列，用于与最终客户费率的节点扣减费归属映射
ALTER TABLE `rate_customer`
  ADD COLUMN `general_fee_owner_id` BIGINT UNSIGNED NULL AFTER `general_fee`;

-- 可选：为按归属筛选提供索引
ALTER TABLE `rate_customer`
  ADD INDEX `idx_rate_customer_general_fee_owner_id` (`general_fee_owner_id`);
