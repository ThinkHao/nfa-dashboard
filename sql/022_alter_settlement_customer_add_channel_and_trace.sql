-- 022_alter_settlement_customer_add_channel_and_trace.sql
-- 在已有的 settlement_customer 表上补充“渠道”相关字段与折损追踪字段，供结算数据明细表与复算使用

START TRANSACTION;

ALTER TABLE `settlement_customer`
    -- 渠道相关：费率/金额/归属系统用户
    ADD COLUMN `channel_rate` DECIMAL(18,6) NULL COMMENT '渠道费率(支出)' AFTER `network_line_fee_owner_id`,
    ADD COLUMN `channel_bill` DECIMAL(20,6) NULL COMMENT '渠道结算金额' AFTER `channel_rate`,
    ADD COLUMN `channel_owner_user_id` BIGINT UNSIGNED NULL COMMENT '渠道费用归属系统用户ID' AFTER `channel_bill`,
    -- 折损追踪：便于审计具体采用的折损规则与年序
    ADD COLUMN `discount_rule_id` BIGINT UNSIGNED NULL COMMENT '折损规则ID快照' AFTER `channel_owner_user_id`,
    ADD COLUMN `service_year_index` INT NULL COMMENT '服务年序(1=首年)' AFTER `discount_rule_id`;

COMMIT;
