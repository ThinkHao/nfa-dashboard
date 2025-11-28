-- 021_rate_discount_and_channel_fields.sql
-- 折损规则表 + 渠道费率/归属 + 客户费率起始时间等字段

START TRANSACTION;

-- 1. rate_customer 增加渠道费率、渠道费用归属用户、客户费率起始时间
ALTER TABLE `rate_customer`
    ADD COLUMN `channel_rate` DECIMAL(18,6) NULL AFTER `general_fee_owner_id`,
    ADD COLUMN `channel_owner_user_id` BIGINT NULL AFTER `channel_rate`,
    ADD COLUMN `start_at` DATE NULL AFTER `channel_owner_user_id`;

-- 2. rate_final_customer 增加渠道费率与渠道费用归属用户
ALTER TABLE `rate_final_customer`
    ADD COLUMN `channel_rate` DECIMAL(18,6) NULL AFTER `node_deduction_fee_owner_id`,
    ADD COLUMN `channel_owner_user_id` BIGINT NULL AFTER `channel_rate`;

-- 3. 客户费率折损规则主表
CREATE TABLE IF NOT EXISTS `rate_discount_rule` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
    `name` VARCHAR(128) NOT NULL COMMENT '规则名称',
    `scope_type` VARCHAR(32) NOT NULL DEFAULT 'global' COMMENT '作用范围类型: global/region/cp/school 等',
    `scope_key` VARCHAR(128) NULL COMMENT '作用范围键值，例如区域、CP 或 school_id',
    `fields` JSON NULL COMMENT '需要折损的费率字段列表，例如 ["customer_fee"]',
    `enabled` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
    `priority` INT NOT NULL DEFAULT 100 COMMENT '优先级，越小越优先',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_rate_discount_rule_scope` (`scope_type`, `scope_key`),
    KEY `idx_rate_discount_rule_priority` (`priority`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='客户费率折损规则主表';

-- 4. 客户费率折损规则明细表
CREATE TABLE IF NOT EXISTS `rate_discount_rule_item` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
    `rule_id` BIGINT UNSIGNED NOT NULL COMMENT '规则ID',
    `from_year` INT NOT NULL COMMENT '服务年限下界，含，例如 1 表示第 1 年起',
    `to_year` INT NULL COMMENT '服务年限上界，含，NULL 表示之后全部',
    `discount_rate` DECIMAL(8,4) NOT NULL COMMENT '折损比例，例如 1.0000, 0.7500',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_rate_discount_rule_item_rule` (`rule_id`),
    CONSTRAINT `fk_rate_discount_rule_item_rule` FOREIGN KEY (`rule_id`) REFERENCES `rate_discount_rule` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='客户费率折损规则明细表';

-- 5. 结算结果表增加服务时间与复算标记（根据现有 settlement_customer/settlement_results 结构适配）
-- 假设业务结算金额主表为 settlement_customer（客户结算金额）
ALTER TABLE `settlement_customer`
    ADD COLUMN `service_date` DATE NULL AFTER `settlement_time`,
    ADD COLUMN `recalculated` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否为复算结果' AFTER `service_date`,
    ADD COLUMN `last_recalc_time` DATETIME NULL COMMENT '最近复算时间' AFTER `recalculated`;

COMMIT;
