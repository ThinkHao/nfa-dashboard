-- Migration: add settlement customer composite index
-- contract: none

-- 1) 复算匹配主过滤键：region/cp/school_name/service_date
SET @ddl := IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS
   WHERE TABLE_SCHEMA = DATABASE()
     AND TABLE_NAME = 'settlement_customer'
     AND INDEX_NAME = 'idx_settlement_customer_region_cp_school_date') = 0,
  'ALTER TABLE `settlement_customer`
     ADD INDEX `idx_settlement_customer_region_cp_school_date` (`region`,`cp`,`school_name`,`service_date`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- 2) 复算源数据范围扫描与排序：settlement_date + 业务维度 + id
SET @ddl := IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS
   WHERE TABLE_SCHEMA = DATABASE()
     AND TABLE_NAME = 'nfa_school_settlement'
     AND INDEX_NAME = 'idx_school_settlement_date_region_cp_school_id') = 0,
  'ALTER TABLE `nfa_school_settlement`
     ADD INDEX `idx_school_settlement_date_region_cp_school_id` (`settlement_date`,`region`,`cp`,`school_name`,`id`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

