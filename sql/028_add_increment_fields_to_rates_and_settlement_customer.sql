-- 028_add_increment_fields_to_rates_and_settlement_customer.sql
-- 客户业务费率增加：增量起算日期、存量占比、增量占比、当日增量值
-- 结算明细增加：存量占比、增量占比、当日增量值（用于留痕）

START TRANSACTION;

ALTER TABLE `rate_customer`
    ADD COLUMN `increment_start_at` DATETIME NULL COMMENT '增量起算日期' AFTER `start_at`,
    ADD COLUMN `stock_ratio` DECIMAL(10,6) NULL COMMENT '存量占比(0-1)' AFTER `increment_start_at`,
    ADD COLUMN `increment_ratio` DECIMAL(10,6) NULL COMMENT '增量占比(0-1)' AFTER `stock_ratio`,
    ADD COLUMN `daily_increment_value` DECIMAL(20,6) NULL COMMENT '当日增量值(=当日日95*增量占比)' AFTER `increment_ratio`;

ALTER TABLE `settlement_customer`
    ADD COLUMN `stock_ratio` DECIMAL(10,6) NULL COMMENT '存量占比(0-1)快照' AFTER `channel_owner_user_id`,
    ADD COLUMN `increment_ratio` DECIMAL(10,6) NULL COMMENT '增量占比(0-1)快照' AFTER `stock_ratio`,
    ADD COLUMN `daily_increment_value` DECIMAL(20,6) NULL COMMENT '当日增量值快照' AFTER `increment_ratio`;

COMMIT;
