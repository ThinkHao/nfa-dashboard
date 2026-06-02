-- Migration: add final node cp fee
-- contract: column=rate_final_node.cp_fee
-- contract: column=rate_final_node.cp_fee_owner_id

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_final_node' AND COLUMN_NAME='cp_fee')=0,
  'ALTER TABLE `rate_final_node` ADD COLUMN `cp_fee` DECIMAL(18,6) NULL AFTER `fee_type`, ADD COLUMN `cp_fee_owner_id` BIGINT UNSIGNED NULL AFTER `cp_fee`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

UPDATE `rate_final_node` rfn
JOIN `rate_node` rn
  ON (
    (rfn.entity_id IS NOT NULL AND rn.entity_id = rfn.entity_id)
    OR (rfn.entity_id IS NULL AND rn.entity_id IS NULL AND rn.region = rfn.region AND rn.cp = rfn.cp)
  )
  AND rn.settlement_mode = rfn.settlement_mode
  AND rn.unit_base = rfn.unit_base
SET rfn.cp_fee = rn.cp_fee,
    rfn.cp_fee_owner_id = rn.cp_fee_owner_id
WHERE (rfn.fee_type = 'auto' OR rfn.fee_type IS NULL OR rfn.fee_type = '');

