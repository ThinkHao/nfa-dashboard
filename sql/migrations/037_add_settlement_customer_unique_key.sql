-- Migration: dedupe settlement_customer and add unique key for upsert
-- contract: none

-- 1) 清理重复数据：同键保留 updated_at 更晚（相同 updated_at 保留 id 更大）
DELETE sc_old
FROM settlement_customer sc_old
INNER JOIN settlement_customer sc_new
  ON sc_old.region = sc_new.region
 AND sc_old.cp = sc_new.cp
 AND sc_old.school_name = sc_new.school_name
 AND sc_old.service_date = sc_new.service_date
 AND (
      sc_old.updated_at < sc_new.updated_at
      OR (sc_old.updated_at = sc_new.updated_at AND sc_old.id < sc_new.id)
 );

-- 2) 增加唯一约束（幂等）
SET @ddl := IF(
  (
    SELECT COUNT(*)
    FROM INFORMATION_SCHEMA.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'settlement_customer'
      AND INDEX_NAME = 'uk_settlement_customer_region_cp_school_date'
  ) = 0,
  'ALTER TABLE `settlement_customer`
     ADD UNIQUE KEY `uk_settlement_customer_region_cp_school_date` (`region`,`cp`,`school_name`,`service_date`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
