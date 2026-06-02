-- Migration: edc node settlement
-- contract: table=rate_final_node
-- contract: column=rate_node.entity_id
-- contract: column=rate_node.unit_base
-- contract: column=rate_node.settlement_mode
-- contract: column=settlement_node_daily95.entity_id
-- contract: column=settlement_node_monthly95.entity_id

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_node' AND COLUMN_NAME='entity_id')=0,
  'ALTER TABLE `rate_node` ADD COLUMN `entity_id` BIGINT UNSIGNED NULL COMMENT ''EDC实体ID，NULL表示区域+CP兜底'' AFTER `id`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_node' AND COLUMN_NAME='display_name')=0,
  'ALTER TABLE `rate_node` ADD COLUMN `display_name` VARCHAR(128) NULL COMMENT ''节点展示名快照'' AFTER `entity_id`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_node' AND COLUMN_NAME='unit_base')=0,
  'ALTER TABLE `rate_node` ADD COLUMN `unit_base` INT NOT NULL DEFAULT 1000 COMMENT ''结算进制: 1000或1024'' AFTER `settlement_type`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_node' AND COLUMN_NAME='settlement_mode')=0,
  'ALTER TABLE `rate_node` ADD COLUMN `settlement_mode` VARCHAR(24) NOT NULL DEFAULT ''daily_95_avg'' COMMENT ''结算模式: daily_95_avg|range_95'' AFTER `unit_base`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

UPDATE `rate_node`
SET `settlement_mode` = CASE WHEN `settlement_type` IN ('monthly95','range_95') THEN 'range_95' ELSE 'daily_95_avg' END
WHERE `settlement_mode` IS NULL OR `settlement_mode` = '';

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_node' AND INDEX_NAME='idx_rate_node_entity')=0,
  'ALTER TABLE `rate_node` ADD INDEX `idx_rate_node_entity` (`entity_id`)',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_node' AND INDEX_NAME='idx_rate_node_region_cp_mode_base')=0,
  'ALTER TABLE `rate_node` ADD INDEX `idx_rate_node_region_cp_mode_base` (`region`,`cp`,`settlement_mode`,`unit_base`)',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

CREATE TABLE IF NOT EXISTS `rate_final_node` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `entity_id` BIGINT UNSIGNED NULL COMMENT 'EDC实体ID，NULL表示区域+CP兜底',
  `display_name` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '节点展示名',
  `region` VARCHAR(32) NOT NULL,
  `cp` VARCHAR(32) NOT NULL,
  `settlement_mode` VARCHAR(24) NOT NULL DEFAULT 'daily_95_avg',
  `unit_base` INT NOT NULL DEFAULT 1000,
  `final_fee` DECIMAL(18,6) NULL COMMENT '最终节点流量单价',
  `fee_type` VARCHAR(16) NOT NULL DEFAULT 'auto',
  `node_construction_fee` DECIMAL(18,6) NULL,
  `node_construction_fee_owner_id` BIGINT UNSIGNED NULL,
  `rack_fee` DECIMAL(18,6) NULL COMMENT '每月固定机柜费',
  `rack_fee_owner_id` BIGINT UNSIGNED NULL,
  `other_fee` DECIMAL(18,6) NULL,
  `other_fee_owner_id` BIGINT UNSIGNED NULL,
  `last_sync_time` DATETIME NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_final_node_entity` (`entity_id`),
  KEY `idx_final_node_region_cp` (`region`,`cp`),
  KEY `idx_final_node_mode_base` (`settlement_mode`,`unit_base`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='最终节点费率（EDC）';

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_node_daily95' AND COLUMN_NAME='entity_id')=0,
  'ALTER TABLE `settlement_node_daily95` ADD COLUMN `entity_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `id`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_node_daily95' AND COLUMN_NAME='display_name')=0,
  'ALTER TABLE `settlement_node_daily95` ADD COLUMN `display_name` VARCHAR(128) NOT NULL DEFAULT '''' AFTER `entity_id`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_node_daily95' AND COLUMN_NAME='service_month')=0,
  'ALTER TABLE `settlement_node_daily95` ADD COLUMN `service_month` VARCHAR(7) NOT NULL DEFAULT '''' AFTER `cp`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_node_daily95' AND COLUMN_NAME='settlement_mode')=0,
  'ALTER TABLE `settlement_node_daily95` ADD COLUMN `settlement_mode` VARCHAR(24) NOT NULL DEFAULT ''daily_95_avg'' AFTER `service_month`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_node_daily95' AND COLUMN_NAME='unit_base')=0,
  'ALTER TABLE `settlement_node_daily95` ADD COLUMN `unit_base` INT NOT NULL DEFAULT 1000 AFTER `settlement_mode`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_node_daily95' AND COLUMN_NAME='raw_95')=0,
  'ALTER TABLE `settlement_node_daily95` ADD COLUMN `raw_95` DECIMAL(24,6) NOT NULL DEFAULT 0 AFTER `unit_base`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_node_daily95' AND COLUMN_NAME='mbps_95')=0,
  'ALTER TABLE `settlement_node_daily95` ADD COLUMN `mbps_95` DECIMAL(24,6) NOT NULL DEFAULT 0 AFTER `raw_95`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_node_daily95' AND COLUMN_NAME='traffic_bill')=0,
  'ALTER TABLE `settlement_node_daily95` ADD COLUMN `traffic_bill` DECIMAL(18,6) NULL AFTER `daily95_bill`, ADD COLUMN `total_bill` DECIMAL(18,6) NULL AFTER `traffic_bill`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_node_daily95' AND INDEX_NAME='uk_node_daily95_entity_date_mode_base')=0,
  'ALTER TABLE `settlement_node_daily95` ADD UNIQUE KEY `uk_node_daily95_entity_date_mode_base` (`entity_id`,`settlement_time`,`settlement_mode`,`unit_base`)',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_node_monthly95' AND COLUMN_NAME='entity_id')=0,
  'ALTER TABLE `settlement_node_monthly95` ADD COLUMN `entity_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `id`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_node_monthly95' AND COLUMN_NAME='display_name')=0,
  'ALTER TABLE `settlement_node_monthly95` ADD COLUMN `display_name` VARCHAR(128) NOT NULL DEFAULT '''' AFTER `entity_id`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_node_monthly95' AND COLUMN_NAME='service_month')=0,
  'ALTER TABLE `settlement_node_monthly95` ADD COLUMN `service_month` VARCHAR(7) NOT NULL DEFAULT '''' AFTER `cp`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_node_monthly95' AND COLUMN_NAME='settlement_mode')=0,
  'ALTER TABLE `settlement_node_monthly95` ADD COLUMN `settlement_mode` VARCHAR(24) NOT NULL DEFAULT ''daily_95_avg'' AFTER `service_month`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_node_monthly95' AND COLUMN_NAME='unit_base')=0,
  'ALTER TABLE `settlement_node_monthly95` ADD COLUMN `unit_base` INT NOT NULL DEFAULT 1000 AFTER `settlement_mode`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_node_monthly95' AND COLUMN_NAME='raw_95')=0,
  'ALTER TABLE `settlement_node_monthly95` ADD COLUMN `raw_95` DECIMAL(24,6) NOT NULL DEFAULT 0 AFTER `unit_base`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_node_monthly95' AND COLUMN_NAME='mbps_95')=0,
  'ALTER TABLE `settlement_node_monthly95` ADD COLUMN `mbps_95` DECIMAL(24,6) NOT NULL DEFAULT 0 AFTER `raw_95`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_node_monthly95' AND COLUMN_NAME='traffic_bill')=0,
  'ALTER TABLE `settlement_node_monthly95` ADD COLUMN `traffic_bill` DECIMAL(18,6) NULL AFTER `monthly95_bill`, ADD COLUMN `total_bill` DECIMAL(18,6) NULL AFTER `traffic_bill`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_node_monthly95' AND INDEX_NAME='uk_node_monthly95_entity_month_mode_base')=0,
  'ALTER TABLE `settlement_node_monthly95` ADD UNIQUE KEY `uk_node_monthly95_entity_month_mode_base` (`entity_id`,`service_month`,`settlement_mode`,`unit_base`)',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;
-- Replace `none` with one or more contract lines when this migration adds runtime-required schema.
-- contract: table=example_table
-- contract: column=example_table.example_column
-- Add idempotent SQL here.
-- If this migration introduces runtime-required tables or columns,
-- update scripts/offline-deploy.sh assert_db_schema() before release.

