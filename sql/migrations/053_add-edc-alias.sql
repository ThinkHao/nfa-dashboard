-- Migration: add a human-managed EDC recognition alias
-- contract: column=edc_entities.alias
-- contract: column=edc_traffic_5m.alias

SET @ddl := IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='edc_entities' AND COLUMN_NAME='alias')=0,
  'ALTER TABLE edc_entities ADD COLUMN alias varchar(255) NULL COMMENT ''人工识别名称'' AFTER display_name', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='edc_traffic_5m' AND COLUMN_NAME='alias')=0,
  'ALTER TABLE edc_traffic_5m ADD COLUMN alias varchar(255) NULL COMMENT ''人工识别名称快照'' AFTER dst_region', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- Existing traffic rows intentionally remain NULL. New extractor snapshots use
-- the current alias, while the dashboard falls back to display_name.
