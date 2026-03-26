-- 为月度快照表补充增量相关字段，支持按月聚合展示（幂等）
SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_customer_monthly' AND COLUMN_NAME='stock_ratio')=0,
  'ALTER TABLE `settlement_customer_monthly` ADD COLUMN `stock_ratio` DECIMAL(10,6) NULL COMMENT ''存量占比(0-1)月均快照'' AFTER `channel_owner_user_id`', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_customer_monthly' AND COLUMN_NAME='increment_ratio')=0,
  'ALTER TABLE `settlement_customer_monthly` ADD COLUMN `increment_ratio` DECIMAL(10,6) NULL COMMENT ''增量占比(0-1)月均快照'' AFTER `stock_ratio`', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_customer_monthly' AND COLUMN_NAME='daily_increment_value')=0,
  'ALTER TABLE `settlement_customer_monthly` ADD COLUMN `daily_increment_value` DECIMAL(20,6) NULL COMMENT ''当日增量值月均快照'' AFTER `increment_ratio`', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;
