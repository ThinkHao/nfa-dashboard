-- contract: table=settlement_customer_v
-- contract: table=settlement_customer_monthly_v
-- contract: table=settlement_month_slot_pointer

CREATE TABLE IF NOT EXISTS `settlement_customer_v` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `region` VARCHAR(32) NOT NULL,
  `cp` VARCHAR(32) NOT NULL,
  `school_name` VARCHAR(128) NOT NULL,
  `service_month` VARCHAR(7) NOT NULL,
  `slot` TINYINT NOT NULL DEFAULT 0,
  `settlement_value` DECIMAL(18,6) NOT NULL,
  `settlement_time` DATETIME NOT NULL,
  `service_date` DATE NULL,
  `recalculated` TINYINT(1) NOT NULL DEFAULT 0,
  `last_recalc_time` DATETIME NULL,
  `customer_fee` DECIMAL(18,6) NULL,
  `customer_bill` DECIMAL(18,2) NULL,
  `customer_fee_owner_id` BIGINT UNSIGNED NULL,
  `network_line_fee` DECIMAL(18,6) NULL,
  `network_line_bill` DECIMAL(18,2) NULL,
  `network_line_fee_owner_id` BIGINT UNSIGNED NULL,
  `node_deduction_fee` DECIMAL(18,6) NULL,
  `node_deduction_bill` DECIMAL(18,2) NULL,
  `node_deduction_fee_owner_id` BIGINT UNSIGNED NULL,
  `channel_rate` DECIMAL(18,6) NULL,
  `channel_bill` DECIMAL(18,2) NULL,
  `channel_owner_user_id` BIGINT UNSIGNED NULL,
  `stock_ratio` DECIMAL(10,6) NULL,
  `increment_ratio` DECIMAL(10,6) NULL,
  `daily_increment_value` DECIMAL(20,6) NULL,
  `discount_rule_id` BIGINT UNSIGNED NULL,
  `service_year_index` INT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_scv_month_slot_region_cp_school_date` (`service_month`,`slot`,`region`,`cp`,`school_name`,`service_date`),
  KEY `idx_scv_region_cp_school_date_month_slot` (`region`,`cp`,`school_name`,`service_date`,`service_month`,`slot`),
  KEY `idx_scv_month_slot_service_date_id` (`service_month`,`slot`,`service_date`,`id`),
  KEY `idx_scv_region` (`region`),
  KEY `idx_scv_cp` (`cp`),
  KEY `idx_scv_school` (`school_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='客户结算明细双槽位表';

CREATE TABLE IF NOT EXISTS `settlement_customer_monthly_v` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `region` VARCHAR(32) NOT NULL,
  `cp` VARCHAR(32) NOT NULL,
  `school_name` VARCHAR(128) NOT NULL,
  `service_month` VARCHAR(7) NOT NULL,
  `slot` TINYINT NOT NULL DEFAULT 0,
  `settlement_value` DECIMAL(18,6) NOT NULL DEFAULT 0,
  `stock_ratio` DECIMAL(10,6) NULL,
  `increment_ratio` DECIMAL(10,6) NULL,
  `daily_increment_value` DECIMAL(20,6) NULL,
  `customer_fee` DECIMAL(18,6) NULL,
  `customer_bill` DECIMAL(18,2) NULL,
  `customer_fee_owner_id` BIGINT UNSIGNED NULL,
  `network_line_fee` DECIMAL(18,6) NULL,
  `network_line_bill` DECIMAL(18,2) NULL,
  `network_line_fee_owner_id` BIGINT UNSIGNED NULL,
  `node_deduction_fee` DECIMAL(18,6) NULL,
  `node_deduction_bill` DECIMAL(18,2) NULL,
  `node_deduction_fee_owner_id` BIGINT UNSIGNED NULL,
  `channel_rate` DECIMAL(18,6) NULL,
  `channel_bill` DECIMAL(18,2) NULL,
  `channel_owner_user_id` BIGINT UNSIGNED NULL,
  `recalculated` TINYINT(1) NOT NULL DEFAULT 0,
  `last_recalc_time` DATETIME NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_scmv_month_slot_region_cp_school` (`service_month`,`slot`,`region`,`cp`,`school_name`),
  KEY `idx_scmv_month_slot` (`service_month`,`slot`),
  KEY `idx_scmv_region_cp` (`region`,`cp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='客户结算月快照双槽位表';

CREATE TABLE IF NOT EXISTS `settlement_month_slot_pointer` (
  `service_month` VARCHAR(7) NOT NULL,
  `active_slot` TINYINT NOT NULL DEFAULT 0,
  `task_id` BIGINT NULL,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`service_month`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='客户结算月槽位发布指针';

-- 初始化指针：已有月份默认 slot=0
INSERT INTO `settlement_month_slot_pointer` (`service_month`, `active_slot`, `task_id`, `updated_at`)
SELECT m.service_month, 0, NULL, NOW()
FROM (
  SELECT DISTINCT DATE_FORMAT(service_date, '%Y-%m') AS service_month
  FROM settlement_customer
  WHERE service_date IS NOT NULL
) m
LEFT JOIN settlement_month_slot_pointer p ON p.service_month = m.service_month
WHERE p.service_month IS NULL;

-- 初始化 slot 表（仅在目标键不存在时写入）
INSERT INTO `settlement_customer_v` (
  `region`,`cp`,`school_name`,`service_month`,`slot`,
  `settlement_value`,`settlement_time`,`service_date`,`recalculated`,`last_recalc_time`,
  `customer_fee`,`customer_bill`,`customer_fee_owner_id`,
  `network_line_fee`,`network_line_bill`,`network_line_fee_owner_id`,
  `node_deduction_fee`,`node_deduction_bill`,`node_deduction_fee_owner_id`,
  `channel_rate`,`channel_bill`,`channel_owner_user_id`,
  `stock_ratio`,`increment_ratio`,`daily_increment_value`,`discount_rule_id`,`service_year_index`,
  `created_at`,`updated_at`
)
SELECT
  sc.`region`, sc.`cp`, sc.`school_name`, DATE_FORMAT(sc.`service_date`, '%Y-%m') AS `service_month`, 0,
  sc.`settlement_value`, sc.`settlement_time`, sc.`service_date`, sc.`recalculated`, sc.`last_recalc_time`,
  sc.`customer_fee`, sc.`customer_bill`, sc.`customer_fee_owner_id`,
  sc.`network_line_fee`, sc.`network_line_bill`, sc.`network_line_fee_owner_id`,
  sc.`node_deduction_fee`, sc.`node_deduction_bill`, sc.`node_deduction_fee_owner_id`,
  sc.`channel_rate`, sc.`channel_bill`, sc.`channel_owner_user_id`,
  sc.`stock_ratio`, sc.`increment_ratio`, sc.`daily_increment_value`, sc.`discount_rule_id`, sc.`service_year_index`,
  sc.`created_at`, sc.`updated_at`
FROM settlement_customer sc
WHERE sc.service_date IS NOT NULL
ON DUPLICATE KEY UPDATE
  settlement_value = VALUES(settlement_value),
  settlement_time = VALUES(settlement_time),
  recalculated = VALUES(recalculated),
  last_recalc_time = VALUES(last_recalc_time),
  customer_fee = VALUES(customer_fee),
  customer_bill = VALUES(customer_bill),
  customer_fee_owner_id = VALUES(customer_fee_owner_id),
  network_line_fee = VALUES(network_line_fee),
  network_line_bill = VALUES(network_line_bill),
  network_line_fee_owner_id = VALUES(network_line_fee_owner_id),
  node_deduction_fee = VALUES(node_deduction_fee),
  node_deduction_bill = VALUES(node_deduction_bill),
  node_deduction_fee_owner_id = VALUES(node_deduction_fee_owner_id),
  channel_rate = VALUES(channel_rate),
  channel_bill = VALUES(channel_bill),
  channel_owner_user_id = VALUES(channel_owner_user_id),
  stock_ratio = VALUES(stock_ratio),
  increment_ratio = VALUES(increment_ratio),
  daily_increment_value = VALUES(daily_increment_value),
  discount_rule_id = VALUES(discount_rule_id),
  service_year_index = VALUES(service_year_index),
  updated_at = VALUES(updated_at);

INSERT INTO `settlement_customer_monthly_v` (
  `region`,`cp`,`school_name`,`service_month`,`slot`,
  `settlement_value`,`stock_ratio`,`increment_ratio`,`daily_increment_value`,
  `customer_fee`,`customer_bill`,`customer_fee_owner_id`,
  `network_line_fee`,`network_line_bill`,`network_line_fee_owner_id`,
  `node_deduction_fee`,`node_deduction_bill`,`node_deduction_fee_owner_id`,
  `channel_rate`,`channel_bill`,`channel_owner_user_id`,
  `recalculated`,`last_recalc_time`,`created_at`,`updated_at`
)
SELECT
  m.`region`, m.`cp`, m.`school_name`, m.`service_month`, 0,
  m.`settlement_value`, m.`stock_ratio`, m.`increment_ratio`, m.`daily_increment_value`,
  m.`customer_fee`, m.`customer_bill`, m.`customer_fee_owner_id`,
  m.`network_line_fee`, m.`network_line_bill`, m.`network_line_fee_owner_id`,
  m.`node_deduction_fee`, m.`node_deduction_bill`, m.`node_deduction_fee_owner_id`,
  m.`channel_rate`, m.`channel_bill`, m.`channel_owner_user_id`,
  m.`recalculated`, m.`last_recalc_time`, m.`created_at`, m.`updated_at`
FROM settlement_customer_monthly m
ON DUPLICATE KEY UPDATE
  settlement_value = VALUES(settlement_value),
  stock_ratio = VALUES(stock_ratio),
  increment_ratio = VALUES(increment_ratio),
  daily_increment_value = VALUES(daily_increment_value),
  customer_fee = VALUES(customer_fee),
  customer_bill = VALUES(customer_bill),
  customer_fee_owner_id = VALUES(customer_fee_owner_id),
  network_line_fee = VALUES(network_line_fee),
  network_line_bill = VALUES(network_line_bill),
  network_line_fee_owner_id = VALUES(network_line_fee_owner_id),
  node_deduction_fee = VALUES(node_deduction_fee),
  node_deduction_bill = VALUES(node_deduction_bill),
  node_deduction_fee_owner_id = VALUES(node_deduction_fee_owner_id),
  channel_rate = VALUES(channel_rate),
  channel_bill = VALUES(channel_bill),
  channel_owner_user_id = VALUES(channel_owner_user_id),
  recalculated = VALUES(recalculated),
  last_recalc_time = VALUES(last_recalc_time),
  updated_at = VALUES(updated_at);
