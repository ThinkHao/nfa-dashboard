-- Migration: edc backup filtering
-- contract: column=edc_entities.is_backup

SET @ddl := IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
   WHERE TABLE_SCHEMA = DATABASE()
     AND TABLE_NAME = 'edc_entities'
     AND COLUMN_NAME = 'is_backup') = 0,
  'ALTER TABLE `edc_entities`
     ADD COLUMN `is_backup` TINYINT(1) NOT NULL DEFAULT 0 COMMENT ''是否为备份采集映射'' AFTER `cp`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS
   WHERE TABLE_SCHEMA = DATABASE()
     AND TABLE_NAME = 'edc_entities'
     AND INDEX_NAME = 'idx_edc_entity_enabled_backup') = 0,
  'ALTER TABLE `edc_entities`
     ADD KEY `idx_edc_entity_enabled_backup` (`enabled`, `is_backup`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

