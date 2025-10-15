-- 016_alter_rate_customer_add_fee_mode.sql
-- 为 rate_customer 增加行级配置模式字段：auto（自动）/ configed（手工）

SET @ddl := IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'rate_customer'
       AND COLUMN_NAME = 'fee_mode') = 0,
  'ALTER TABLE `rate_customer` ADD COLUMN `fee_mode` ENUM(''auto'',''configed'') NOT NULL DEFAULT ''auto'' COMMENT ''配置模式：auto=自动, configed=手工'' AFTER `network_line_fee_owner_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- 可选：为历史脏数据将空字符串修正为默认
UPDATE `rate_customer` SET `fee_mode` = 'auto' WHERE `fee_mode` IS NULL OR `fee_mode` = '';
