-- Migration: edc node settlement schedule config
-- contract: column=nfa_settlement_config.node_daily_enabled
-- contract: column=nfa_settlement_config.node_daily_time
-- contract: column=nfa_settlement_config.node_monthly_enabled
-- contract: column=nfa_settlement_config.node_monthly_day
-- contract: column=nfa_settlement_config.node_monthly_time

SET @ddl := IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
   WHERE TABLE_SCHEMA = DATABASE()
     AND TABLE_NAME = 'nfa_settlement_config'
     AND COLUMN_NAME = 'node_daily_enabled') = 0,
  'ALTER TABLE `nfa_settlement_config`
     ADD COLUMN `node_daily_enabled` TINYINT(1) NOT NULL DEFAULT 0 COMMENT ''是否启用EDC节点每日自动结算'' AFTER `weekly_enabled`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
   WHERE TABLE_SCHEMA = DATABASE()
     AND TABLE_NAME = 'nfa_settlement_config'
     AND COLUMN_NAME = 'node_daily_time') = 0,
  'ALTER TABLE `nfa_settlement_config`
     ADD COLUMN `node_daily_time` VARCHAR(5) NOT NULL DEFAULT ''03:00'' COMMENT ''EDC节点每日结算时间'' AFTER `node_daily_enabled`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
   WHERE TABLE_SCHEMA = DATABASE()
     AND TABLE_NAME = 'nfa_settlement_config'
     AND COLUMN_NAME = 'node_monthly_enabled') = 0,
  'ALTER TABLE `nfa_settlement_config`
     ADD COLUMN `node_monthly_enabled` TINYINT(1) NOT NULL DEFAULT 0 COMMENT ''是否启用EDC节点每月自动结算'' AFTER `node_daily_time`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
   WHERE TABLE_SCHEMA = DATABASE()
     AND TABLE_NAME = 'nfa_settlement_config'
     AND COLUMN_NAME = 'node_monthly_day') = 0,
  'ALTER TABLE `nfa_settlement_config`
     ADD COLUMN `node_monthly_day` INT NOT NULL DEFAULT 1 COMMENT ''EDC节点月结算触发日(1-31)'' AFTER `node_monthly_enabled`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
   WHERE TABLE_SCHEMA = DATABASE()
     AND TABLE_NAME = 'nfa_settlement_config'
     AND COLUMN_NAME = 'node_monthly_time') = 0,
  'ALTER TABLE `nfa_settlement_config`
     ADD COLUMN `node_monthly_time` VARCHAR(5) NOT NULL DEFAULT ''04:00'' COMMENT ''EDC节点月结算时间'' AFTER `node_monthly_day`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;
-- Replace `none` with one or more contract lines when this migration adds runtime-required schema.
-- contract: table=example_table
-- contract: column=example_table.example_column
-- Add idempotent SQL here.
-- If this migration introduces runtime-required tables or columns,
-- update scripts/offline-deploy.sh assert_db_schema() before release.

