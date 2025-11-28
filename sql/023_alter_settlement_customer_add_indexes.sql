-- 023_alter_settlement_customer_add_indexes.sql
-- 为 settlement_customer 增加查询与复算常用索引

START TRANSACTION;

ALTER TABLE `settlement_customer`
  ADD INDEX `idx_service_date` (`service_date`),
  ADD INDEX `idx_channel_owner` (`channel_owner_user_id`),
  ADD INDEX `idx_discount_rule` (`discount_rule_id`);

COMMIT;
