-- Migration: add node settlement groups
-- contract: table=edc_node_settlement_groups
-- contract: table=edc_node_settlement_group_members
-- contract: column=rate_node.billing_subject_type
-- contract: column=rate_node.billing_subject_id
-- contract: column=rate_node.billing_display_name
-- contract: column=rate_final_node.billing_subject_type
-- contract: column=rate_final_node.billing_subject_id
-- contract: column=rate_final_node.billing_display_name
-- contract: column=settlement_node_daily95.billing_subject_type
-- contract: column=settlement_node_daily95.billing_subject_id
-- contract: column=settlement_node_daily95.billing_display_name
-- contract: column=settlement_node_monthly95.billing_subject_type
-- contract: column=settlement_node_monthly95.billing_subject_id
-- contract: column=settlement_node_monthly95.billing_display_name

CREATE TABLE IF NOT EXISTS `edc_node_settlement_groups` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `group_name` VARCHAR(128) NOT NULL COMMENT '结算分组名称',
  `region` VARCHAR(32) NOT NULL COMMENT '区域',
  `cp` VARCHAR(32) NOT NULL COMMENT 'CP',
  `enabled` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
  `remark` VARCHAR(255) NULL COMMENT '备注',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_edc_node_settlement_groups_name` (`group_name`),
  KEY `idx_edc_node_settlement_groups_region_cp` (`region`,`cp`,`enabled`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='EDC节点结算分组';

CREATE TABLE IF NOT EXISTS `edc_node_settlement_group_members` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `group_id` BIGINT UNSIGNED NOT NULL COMMENT '结算分组ID',
  `entity_id` BIGINT UNSIGNED NOT NULL COMMENT 'EDC实体ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_edc_node_settlement_group_members_entity` (`entity_id`),
  UNIQUE KEY `uk_edc_node_settlement_group_members_group_entity` (`group_id`,`entity_id`),
  KEY `idx_edc_node_settlement_group_members_group` (`group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='EDC节点结算分组成员';

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_node' AND COLUMN_NAME='billing_subject_type')=0,
  'ALTER TABLE `rate_node` ADD COLUMN `billing_subject_type` VARCHAR(16) NOT NULL DEFAULT ''node'' COMMENT ''计费主体类型: node|group'' AFTER `display_name`, ADD COLUMN `billing_subject_id` BIGINT UNSIGNED NULL COMMENT ''计费主体ID'' AFTER `billing_subject_type`, ADD COLUMN `billing_display_name` VARCHAR(128) NULL COMMENT ''计费主体展示名'' AFTER `billing_subject_id`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_final_node' AND COLUMN_NAME='billing_subject_type')=0,
  'ALTER TABLE `rate_final_node` ADD COLUMN `billing_subject_type` VARCHAR(16) NOT NULL DEFAULT ''node'' COMMENT ''计费主体类型: node|group'' AFTER `display_name`, ADD COLUMN `billing_subject_id` BIGINT UNSIGNED NULL COMMENT ''计费主体ID'' AFTER `billing_subject_type`, ADD COLUMN `billing_display_name` VARCHAR(128) NULL COMMENT ''计费主体展示名'' AFTER `billing_subject_id`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_node_daily95' AND COLUMN_NAME='billing_subject_type')=0,
  'ALTER TABLE `settlement_node_daily95` ADD COLUMN `billing_subject_type` VARCHAR(16) NOT NULL DEFAULT ''node'' COMMENT ''计费主体类型: node|group'' AFTER `display_name`, ADD COLUMN `billing_subject_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT ''计费主体ID'' AFTER `billing_subject_type`, ADD COLUMN `billing_display_name` VARCHAR(128) NOT NULL DEFAULT '''' COMMENT ''计费主体展示名'' AFTER `billing_subject_id`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_node_monthly95' AND COLUMN_NAME='billing_subject_type')=0,
  'ALTER TABLE `settlement_node_monthly95` ADD COLUMN `billing_subject_type` VARCHAR(16) NOT NULL DEFAULT ''node'' COMMENT ''计费主体类型: node|group'' AFTER `display_name`, ADD COLUMN `billing_subject_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT ''计费主体ID'' AFTER `billing_subject_type`, ADD COLUMN `billing_display_name` VARCHAR(128) NOT NULL DEFAULT '''' COMMENT ''计费主体展示名'' AFTER `billing_subject_id`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

UPDATE `rate_node`
SET `billing_subject_type` = 'node',
    `billing_subject_id` = `entity_id`,
    `billing_display_name` = COALESCE(`display_name`, '')
WHERE `billing_subject_type` = '' OR `billing_subject_type` IS NULL OR `billing_subject_type` = 'node';

UPDATE `rate_final_node`
SET `billing_subject_type` = 'node',
    `billing_subject_id` = `entity_id`,
    `billing_display_name` = COALESCE(`display_name`, '')
WHERE `billing_subject_type` = '' OR `billing_subject_type` IS NULL OR `billing_subject_type` = 'node';

UPDATE `settlement_node_daily95`
SET `billing_subject_type` = 'node',
    `billing_subject_id` = `entity_id`,
    `billing_display_name` = `display_name`
WHERE `billing_subject_id` = 0 AND `entity_id` > 0;

UPDATE `settlement_node_monthly95`
SET `billing_subject_type` = 'node',
    `billing_subject_id` = `entity_id`,
    `billing_display_name` = `display_name`
WHERE `billing_subject_id` = 0 AND `entity_id` > 0;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_node' AND INDEX_NAME='uk_rate_node_entity_mode_base') > 0,
  'ALTER TABLE `rate_node` DROP INDEX `uk_rate_node_entity_mode_base`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_node' AND INDEX_NAME='uk_rate_node_subject_mode_base') = 0,
  'ALTER TABLE `rate_node` ADD UNIQUE KEY `uk_rate_node_subject_mode_base` (`billing_subject_type`,`billing_subject_id`,`settlement_mode`,`unit_base`)',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_final_node' AND INDEX_NAME='uk_final_node_entity_mode_base') > 0,
  'ALTER TABLE `rate_final_node` DROP INDEX `uk_final_node_entity_mode_base`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_final_node' AND INDEX_NAME='uk_final_node_subject_mode_base') = 0,
  'ALTER TABLE `rate_final_node` ADD UNIQUE KEY `uk_final_node_subject_mode_base` (`billing_subject_type`,`billing_subject_id`,`settlement_mode`,`unit_base`)',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_node_daily95' AND INDEX_NAME='uk_node_daily95_entity_date_mode_base') > 0,
  'ALTER TABLE `settlement_node_daily95` DROP INDEX `uk_node_daily95_entity_date_mode_base`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_node_daily95' AND INDEX_NAME='uk_node_daily95_subject_date_mode_base') = 0,
  'ALTER TABLE `settlement_node_daily95` ADD UNIQUE KEY `uk_node_daily95_subject_date_mode_base` (`billing_subject_type`,`billing_subject_id`,`settlement_time`,`settlement_mode`,`unit_base`)',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_node_monthly95' AND INDEX_NAME='uk_node_monthly95_entity_month_mode_base') > 0,
  'ALTER TABLE `settlement_node_monthly95` DROP INDEX `uk_node_monthly95_entity_month_mode_base`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_node_monthly95' AND INDEX_NAME='uk_node_monthly95_subject_month_mode_base') = 0,
  'ALTER TABLE `settlement_node_monthly95` ADD UNIQUE KEY `uk_node_monthly95_subject_month_mode_base` (`billing_subject_type`,`billing_subject_id`,`service_month`,`settlement_mode`,`unit_base`)',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

