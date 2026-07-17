-- Migration: add settlement src region
-- contract: column=nfa_school_settlement.src_region
-- contract: column=settlement_customer.src_region
-- contract: column=settlement_customer_v.src_region
-- contract: column=settlement_customer_monthly.src_region
-- contract: column=settlement_customer_monthly_v.src_region

SET @ddl := IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='nfa_school_settlement' AND COLUMN_NAME='src_region')=0,
  'ALTER TABLE nfa_school_settlement ADD COLUMN src_region varchar(20) NULL COMMENT ''节点源区域快照'' AFTER region', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @ddl := IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_customer' AND COLUMN_NAME='src_region')=0,
  'ALTER TABLE settlement_customer ADD COLUMN src_region varchar(20) NULL COMMENT ''节点源区域快照'' AFTER region', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @ddl := IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_customer_v' AND COLUMN_NAME='src_region')=0,
  'ALTER TABLE settlement_customer_v ADD COLUMN src_region varchar(20) NULL COMMENT ''节点源区域快照'' AFTER region', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @ddl := IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_customer_monthly' AND COLUMN_NAME='src_region')=0,
  'ALTER TABLE settlement_customer_monthly ADD COLUMN src_region varchar(20) NULL COMMENT ''节点源区域快照'' AFTER region', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @ddl := IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_customer_monthly_v' AND COLUMN_NAME='src_region')=0,
  'ALTER TABLE settlement_customer_monthly_v ADD COLUMN src_region varchar(20) NULL COMMENT ''节点源区域快照'' AFTER region', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

UPDATE nfa_school_settlement ss
JOIN (
  SELECT school_id, region, cp, MAX(src_region) AS src_region
  FROM nfa_school
  WHERE src_region IS NOT NULL AND src_region <> ''
  GROUP BY school_id, region, cp
  HAVING COUNT(DISTINCT src_region) = 1
) s ON CONVERT(s.school_id USING utf8mb4) COLLATE utf8mb4_unicode_ci=ss.school_id
   AND CONVERT(s.region USING utf8mb4) COLLATE utf8mb4_unicode_ci=ss.region
   AND CONVERT(s.cp USING utf8mb4) COLLATE utf8mb4_unicode_ci=ss.cp
SET ss.src_region=CONVERT(s.src_region USING utf8mb4) COLLATE utf8mb4_unicode_ci
WHERE ss.src_region IS NULL;

UPDATE settlement_customer c
JOIN (
  SELECT school_name, region, cp, MAX(src_region) AS src_region
  FROM nfa_school
  WHERE src_region IS NOT NULL AND src_region <> ''
  GROUP BY school_name, region, cp
  HAVING COUNT(DISTINCT src_region) = 1
) s ON CONVERT(s.school_name USING utf8mb4) COLLATE utf8mb4_unicode_ci=c.school_name
   AND CONVERT(s.region USING utf8mb4) COLLATE utf8mb4_unicode_ci=c.region
   AND CONVERT(s.cp USING utf8mb4) COLLATE utf8mb4_unicode_ci=c.cp
SET c.src_region=CONVERT(s.src_region USING utf8mb4) COLLATE utf8mb4_unicode_ci
WHERE c.src_region IS NULL;

UPDATE settlement_customer_v v
JOIN settlement_customer c ON c.region=v.region AND c.cp=v.cp AND c.school_name=v.school_name AND c.service_date=v.service_date
SET v.src_region=c.src_region
WHERE v.src_region IS NULL AND c.src_region IS NOT NULL;

UPDATE settlement_customer_monthly m
JOIN (
  SELECT region, cp, school_name, DATE_FORMAT(service_date, '%Y-%m') AS service_month, MAX(src_region) AS src_region
  FROM settlement_customer
  WHERE src_region IS NOT NULL
  GROUP BY region, cp, school_name, DATE_FORMAT(service_date, '%Y-%m')
  HAVING COUNT(DISTINCT src_region)=1
) d ON CONVERT(d.region USING utf8mb4) COLLATE utf8mb4_0900_ai_ci=m.region
   AND CONVERT(d.cp USING utf8mb4) COLLATE utf8mb4_0900_ai_ci=m.cp
   AND CONVERT(d.school_name USING utf8mb4) COLLATE utf8mb4_0900_ai_ci=m.school_name
   AND CONVERT(d.service_month USING utf8mb4) COLLATE utf8mb4_0900_ai_ci=m.service_month
SET m.src_region=CONVERT(d.src_region USING utf8mb4) COLLATE utf8mb4_0900_ai_ci WHERE m.src_region IS NULL;

UPDATE settlement_customer_monthly_v m
JOIN (
  SELECT region, cp, school_name, service_month, slot, MAX(src_region) AS src_region
  FROM settlement_customer_v
  WHERE src_region IS NOT NULL
  GROUP BY region, cp, school_name, service_month, slot
  HAVING COUNT(DISTINCT src_region)=1
) d ON d.region=m.region AND d.cp=m.cp AND d.school_name=m.school_name AND d.service_month=m.service_month AND d.slot=m.slot
SET m.src_region=d.src_region WHERE m.src_region IS NULL;

SET @ddl := IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_customer' AND INDEX_NAME='idx_sc_date_src_region')=0,
  'ALTER TABLE settlement_customer ADD INDEX idx_sc_date_src_region (src_region, service_date)', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @ddl := IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_customer_v' AND INDEX_NAME='idx_scv_month_slot_src_region')=0,
  'ALTER TABLE settlement_customer_v ADD INDEX idx_scv_month_slot_src_region (src_region, service_month, slot, service_date)', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @ddl := IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_customer_monthly' AND INDEX_NAME='idx_scm_month_src_region')=0,
  'ALTER TABLE settlement_customer_monthly ADD INDEX idx_scm_month_src_region (src_region, service_month)', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @ddl := IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='settlement_customer_monthly_v' AND INDEX_NAME='idx_scmv_month_slot_src_region')=0,
  'ALTER TABLE settlement_customer_monthly_v ADD INDEX idx_scmv_month_slot_src_region (src_region, service_month, slot)', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;
