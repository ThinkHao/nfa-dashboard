-- Migration: add EDC type and source/destination region dimensions
-- contract: column=edc_entities.entity_type
-- contract: column=edc_entities.src_region
-- contract: column=edc_entities.dst_region
-- contract: column=edc_traffic_5m.entity_type
-- contract: column=edc_traffic_5m.src_region
-- contract: column=edc_traffic_5m.dst_region

SET @ddl := IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='edc_entities' AND COLUMN_NAME='entity_type')=0,
  'ALTER TABLE edc_entities ADD COLUMN entity_type varchar(20) NULL COMMENT ''节点或传输'' AFTER cp', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @ddl := IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='edc_entities' AND COLUMN_NAME='src_region')=0,
  'ALTER TABLE edc_entities ADD COLUMN src_region varchar(20) NULL COMMENT ''流量源区域'' AFTER entity_type', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @ddl := IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='edc_entities' AND COLUMN_NAME='dst_region')=0,
  'ALTER TABLE edc_entities ADD COLUMN dst_region varchar(20) NULL COMMENT ''流量目区域'' AFTER src_region', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='edc_traffic_5m' AND COLUMN_NAME='entity_type')=0,
  'ALTER TABLE edc_traffic_5m ADD COLUMN entity_type varchar(20) NULL COMMENT ''类型快照'' AFTER cp', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @ddl := IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='edc_traffic_5m' AND COLUMN_NAME='src_region')=0,
  'ALTER TABLE edc_traffic_5m ADD COLUMN src_region varchar(20) NULL COMMENT ''源区域快照'' AFTER entity_type', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @ddl := IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='edc_traffic_5m' AND COLUMN_NAME='dst_region')=0,
  'ALTER TABLE edc_traffic_5m ADD COLUMN dst_region varchar(20) NULL COMMENT ''目区域快照'' AFTER src_region', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='edc_entities' AND INDEX_NAME='idx_edc_entity_dimensions')=0,
  'ALTER TABLE edc_entities ADD KEY idx_edc_entity_dimensions (enabled, is_backup, entity_type, src_region, dst_region)', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @ddl := IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='edc_traffic_5m' AND INDEX_NAME='idx_edc_traffic_dimensions_bucket')=0,
  'ALTER TABLE edc_traffic_5m ADD KEY idx_edc_traffic_dimensions_bucket (entity_type, src_region, dst_region, bucket_5m)', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- Existing traffic rows are intentionally not backfilled here. They require a
-- separately previewed mapping decision because source/destination regions are
-- manual dimensions and historical values should not be guessed.
