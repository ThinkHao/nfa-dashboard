-- Migration: relax rate node settlement mode key
-- contract: none

SET @ddl := IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS
   WHERE TABLE_SCHEMA = DATABASE()
     AND TABLE_NAME = 'rate_node'
     AND INDEX_NAME = 'uk_rate_node') > 0,
  'ALTER TABLE `rate_node` DROP INDEX `uk_rate_node`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS
   WHERE TABLE_SCHEMA = DATABASE()
     AND TABLE_NAME = 'rate_node'
     AND INDEX_NAME = 'uk_rate_node_entity_mode_base') = 0,
  'ALTER TABLE `rate_node` ADD UNIQUE KEY `uk_rate_node_entity_mode_base` (`entity_id`,`settlement_mode`,`unit_base`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

