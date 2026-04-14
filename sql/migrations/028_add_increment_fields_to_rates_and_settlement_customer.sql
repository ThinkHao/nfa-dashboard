-- contract: column=rate_customer.increment_start_at
-- contract: column=rate_customer.stock_ratio
-- contract: column=rate_customer.increment_ratio
-- contract: column=rate_customer.daily_increment_value
-- contract: column=settlement_customer.stock_ratio
-- contract: column=settlement_customer.increment_ratio
-- contract: column=settlement_customer.daily_increment_value
-- 028_add_increment_fields_to_rates_and_settlement_customer.sql
-- 客户业务费率增加：增量起算日期、存量占比、增量占比、当日增量值
-- 结算明细增加：存量占比、增量占比、当日增量值（用于留痕）

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_customer' AND COLUMN_NAME='increment_start_at')=0,
  'ALTER TABLE `rate_customer` ADD COLUMN `increment_start_at` DATETIME NULL COMMENT ''增量起算日期'' AFTER `start_at`', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_customer' AND COLUMN_NAME='stock_ratio')=0,
  'ALTER TABLE `rate_customer` ADD COLUMN `stock_ratio` DECIMAL(10,6) NULL COMMENT ''存量占比(0-1)'' AFTER `increment_start_at`', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_customer' AND COLUMN_NAME='increment_ratio')=0,
  'ALTER TABLE `rate_customer` ADD COLUMN `increment_ratio` DECIMAL(10,6) NULL COMMENT ''增量占比(0-1)'' AFTER `stock_ratio`', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_customer' AND COLUMN_NAME='daily_increment_value')=0,
  'ALTER TABLE `rate_customer` ADD COLUMN `daily_increment_value` DECIMAL(20,6) NULL COMMENT ''当日增量值(=当日日95*增量占比)'' AFTER `increment_ratio`', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_customer' AND COLUMN_NAME='stock_ratio')=0,
  'ALTER TABLE `settlement_customer` ADD COLUMN `stock_ratio` DECIMAL(10,6) NULL COMMENT ''存量占比(0-1)快照'' AFTER `channel_owner_user_id`', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_customer' AND COLUMN_NAME='increment_ratio')=0,
  'ALTER TABLE `settlement_customer` ADD COLUMN `increment_ratio` DECIMAL(10,6) NULL COMMENT ''增量占比(0-1)快照'' AFTER `stock_ratio`', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_customer' AND COLUMN_NAME='daily_increment_value')=0,
  'ALTER TABLE `settlement_customer` ADD COLUMN `daily_increment_value` DECIMAL(20,6) NULL COMMENT ''当日增量值快照'' AFTER `increment_ratio`', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;
