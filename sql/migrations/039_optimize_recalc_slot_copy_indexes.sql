-- Migration: optimize recalc key-path indexes for slot copy and source scan
-- contract: none

-- 1) Ensure slot key index exists for hit-key delete/copy/upsert path
SET @ddl := IF(
  (
    SELECT COUNT(*)
    FROM INFORMATION_SCHEMA.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'settlement_customer_v'
      AND INDEX_NAME IN ('uk_scv_month_slot_region_cp_school_date', 'idx_scv_month_slot_region_cp_school_date')
  ) = 0,
  'ALTER TABLE `settlement_customer_v`
     ADD INDEX `idx_scv_month_slot_region_cp_school_date` (`service_month`,`slot`,`region`,`cp`,`school_name`,`service_date`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- 2) Accelerate hit-key JOIN in slot copy/delete path
SET @ddl := IF(
  (
    SELECT COUNT(*)
    FROM INFORMATION_SCHEMA.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'settlement_customer_v'
      AND INDEX_NAME = 'idx_scv_region_cp_school_date_month_slot'
  ) = 0,
  'ALTER TABLE `settlement_customer_v`
     ADD INDEX `idx_scv_region_cp_school_date_month_slot` (`region`,`cp`,`school_name`,`service_date`,`service_month`,`slot`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- 3) Optimize monthly source range scan + scope filtering
SET @ddl := IF(
  (
    SELECT COUNT(*)
    FROM INFORMATION_SCHEMA.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'nfa_school_settlement'
      AND INDEX_NAME = 'idx_school_settlement_date_region_cp_school_time'
  ) = 0,
  'ALTER TABLE `nfa_school_settlement`
     ADD INDEX `idx_school_settlement_date_region_cp_school_time` (`settlement_date`,`region`,`cp`,`school_name`,`settlement_time`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;
