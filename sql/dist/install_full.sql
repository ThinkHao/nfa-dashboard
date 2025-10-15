-- NFA Dashboard Install Script (MySQL 5.7 compatible)
-- Purpose: Create/alter required tables and seed data in a safe, idempotent way.
-- Notes:
--  - Uses information_schema checks and dynamic SQL for ALTER operations.
--  - Execute once on production, after taking a full backup.

SET NAMES utf8mb4;
SET time_zone = '+08:00';

-- -----------------------------
-- Core tables (IF NOT EXISTS)
-- -----------------------------

-- Settlement config
CREATE TABLE IF NOT EXISTS `nfa_settlement_config` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `daily_time` VARCHAR(5) NOT NULL COMMENT '每日结算时间，格式HH:MM',
  `weekly_day` TINYINT NOT NULL COMMENT '每周结算日，1-7',
  `weekly_time` VARCHAR(5) NOT NULL COMMENT '每周结算时间，格式HH:MM',
  `enabled` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用自动结算',
  `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `update_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='结算自动任务配置';

-- Init one row if empty
INSERT INTO `nfa_settlement_config` (`daily_time`,`weekly_day`,`weekly_time`,`enabled`)
SELECT '02:00',1,'02:00',1
WHERE NOT EXISTS (SELECT 1 FROM `nfa_settlement_config` LIMIT 1);

-- Settlement task
CREATE TABLE IF NOT EXISTS `nfa_settlement_task` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `task_type` VARCHAR(16) NOT NULL COMMENT 'daily/weekly',
  `task_date` DATE NOT NULL,
  `status` VARCHAR(16) NOT NULL COMMENT 'pending/running/success/failed',
  `start_time` DATETIME DEFAULT NULL,
  `end_time` DATETIME DEFAULT NULL,
  `processed_count` INT NOT NULL DEFAULT 0,
  `error_message` TEXT,
  `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `update_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_task_date` (`task_date`),
  KEY `idx_task_type` (`task_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='结算任务记录表';

-- Settlement results
CREATE TABLE IF NOT EXISTS `nfa_settlement_results` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `formula_id` BIGINT UNSIGNED NOT NULL,
  `formula_name` VARCHAR(128) NOT NULL,
  `formula_tokens` JSON NOT NULL,
  `region` VARCHAR(64) NOT NULL,
  `cp` VARCHAR(64) NOT NULL,
  `school_id` VARCHAR(64) NOT NULL,
  `school_name` VARCHAR(255) NOT NULL,
  `start_date` DATE NOT NULL,
  `end_date` DATE NOT NULL,
  `billing_days` INT NOT NULL DEFAULT 0,
  `total_95_flow` DECIMAL(20,6) NOT NULL DEFAULT 0,
  `average_95_flow` DECIMAL(20,6) NOT NULL DEFAULT 0,
  `customer_fee` DECIMAL(18,6) NULL,
  `network_line_fee` DECIMAL(18,6) NULL,
  `node_deduction_fee` DECIMAL(18,6) NULL,
  `final_fee` DECIMAL(18,6) NULL,
  `amount` DECIMAL(20,6) NULL,
  `currency` VARCHAR(8) NOT NULL DEFAULT 'CNY',
  `missing_days` INT NOT NULL DEFAULT 0,
  `missing_fields` JSON NULL,
  `calculation_detail` JSON NULL,
  `calculated_by` BIGINT UNSIGNED NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='结算公式结果缓存（展示/复用）';

-- -----------------------------
-- Migrations via procedure (MySQL 5.7 friendly)
-- -----------------------------
DROP PROCEDURE IF EXISTS apply_migrations;
DELIMITER //
CREATE PROCEDURE apply_migrations()
BEGIN
  -- Ensure unique key on nfa_settlement_results only if the table already exists
  IF EXISTS (
    SELECT 1 FROM information_schema.TABLES
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'nfa_settlement_results'
  ) THEN
    IF NOT EXISTS (
      SELECT 1 FROM information_schema.STATISTICS
      WHERE TABLE_SCHEMA = DATABASE()
        AND TABLE_NAME = 'nfa_settlement_results'
        AND INDEX_NAME = 'uk_formula_school_range'
    ) THEN
      SET @stmt := 'ALTER TABLE `nfa_settlement_results` 
        ADD UNIQUE KEY `uk_formula_school_range` (`formula_id`,`school_id`,`region`,`cp`,`start_date`,`end_date`)';
      PREPARE s1 FROM @stmt; EXECUTE s1; DEALLOCATE PREPARE s1;
    END IF;
  END IF;
END//
DELIMITER ;
CALL apply_migrations();
DROP PROCEDURE IF EXISTS apply_migrations;

-- End of script

-- =============================================================
-- Append: Consolidated incremental migrations (001~020 & extras)
-- Target: MySQL 5.7 compatible, idempotent
-- Notes:
-- - 基础脚本（已在生产执行）：nfa_school.sql、nfa_school_traffic.sql、nfa_school_settlement.sql、nfa_settlement_config.sql
-- - 以下为后续增量脚本的整合，均采用 IF NOT EXISTS / information_schema 守卫
-- =============================================================

-- 001_auth_users.sql
CREATE TABLE IF NOT EXISTS `users` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `username` VARCHAR(64) NOT NULL,
  `password_hash` VARCHAR(255) NOT NULL,
  `email` VARCHAR(128) NULL,
  `phone` VARCHAR(32) NULL,
  `status` TINYINT NOT NULL DEFAULT 1,
  `last_login_at` DATETIME NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_users_username` (`username`),
  KEY `idx_users_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 002_auth_roles.sql
CREATE TABLE IF NOT EXISTS `roles` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(64) NOT NULL,
  `description` VARCHAR(255) NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_roles_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 003_auth_permissions.sql
CREATE TABLE IF NOT EXISTS `permissions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `code` VARCHAR(128) NOT NULL,
  `name` VARCHAR(128) NOT NULL,
  `description` VARCHAR(255) NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_permissions_code` (`code`),
  KEY `idx_permissions_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 004_auth_user_roles.sql
CREATE TABLE IF NOT EXISTS `user_roles` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `role_id` BIGINT UNSIGNED NOT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_role` (`user_id`,`role_id`),
  KEY `idx_user_roles_user_id` (`user_id`),
  KEY `idx_user_roles_role_id` (`role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 005_auth_role_permissions.sql
CREATE TABLE IF NOT EXISTS `role_permissions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `role_id` BIGINT UNSIGNED NOT NULL,
  `permission_id` BIGINT UNSIGNED NOT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_role_permission` (`role_id`,`permission_id`),
  KEY `idx_role_permissions_role_id` (`role_id`),
  KEY `idx_role_permissions_permission_id` (`permission_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 006_operation_logs.sql
CREATE TABLE IF NOT EXISTS `operation_logs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NULL,
  `method` VARCHAR(16) NOT NULL,
  `path` VARCHAR(255) NOT NULL,
  `resource` VARCHAR(128) NULL,
  `action` VARCHAR(64) NULL,
  `status_code` INT NOT NULL,
  `success` TINYINT NOT NULL DEFAULT 1,
  `latency_ms` INT NULL,
  `ip` VARCHAR(64) NULL,
  `user_agent` VARCHAR(255) NULL,
  `error_message` VARCHAR(512) NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_operation_logs_user_id` (`user_id`),
  KEY `idx_operation_logs_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 007_seed_auth.sql（幂等）
INSERT INTO `roles` (`name`, `description`)
VALUES ('admin', 'System Administrator')
ON DUPLICATE KEY UPDATE `description` = VALUES(`description`);

INSERT INTO `permissions` (`code`, `name`, `description`) VALUES
  ('rates.customer.read', '客户费率-查看', 'View customer rate'),
  ('rates.customer.write', '客户费率-编辑', 'Edit customer rate'),
  ('rates.node.read', '节点费率-查看', 'View node rate'),
  ('rates.node.write', '节点费率-编辑', 'Edit node rate'),
  ('rates.final.read', '最终费率-查看', 'View final customer rate'),
  ('rates.final.write', '最终费率-编辑', 'Edit final customer rate'),
  ('settlement.calculate', '结算-计算', 'Run settlement calculation'),
  ('settlement.export', '结算-导出', 'Export settlement data'),
  ('bizobject.manage', '业务对象-管理', 'Manage business objects'),
  ('baseconfig.manage', '进制配置-管理', 'Manage base config'),
  ('operation_logs.read', '操作日志-查看', 'Read operation logs'),
  ('system.user.manage', '系统用户-管理', 'Manage system users'),
  ('system.role.manage', '系统角色-管理', 'Manage system roles')
ON DUPLICATE KEY UPDATE `name` = VALUES(`name`), `description` = VALUES(`description`);

INSERT INTO `users` (`username`, `password_hash`, `email`, `phone`, `status`)
VALUES ('admin', '$2a$10$OKljOBETJDI9ZlEpskETjOrp8796zWl/z5JEqeCAHl3nRG/mYnjO6', 'admin@example.com', NULL, 1)
ON DUPLICATE KEY UPDATE `password_hash`=VALUES(`password_hash`),`email`=VALUES(`email`),`status`=VALUES(`status`);

INSERT IGNORE INTO `user_roles` (`user_id`, `role_id`)
SELECT u.id, r.id FROM `users` u JOIN `roles` r ON r.name = 'admin' WHERE u.username = 'admin';

INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`)
SELECT r.id, p.id FROM `roles` r JOIN `permissions` p ON p.code IN (
  'rates.customer.read','rates.customer.write','rates.node.read','rates.node.write',
  'rates.final.read','rates.final.write','settlement.calculate','settlement.export',
  'bizobject.manage','baseconfig.manage','operation_logs.read','system.user.manage','system.role.manage')
WHERE r.name = 'admin';

-- 008_settlement_rates.sql（核心表，若不存在则创建最小必需结构）
CREATE TABLE IF NOT EXISTS `business_entities` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `entity_type` VARCHAR(50) NOT NULL,
  `entity_name` VARCHAR(100) NOT NULL,
  `contact_info` VARCHAR(255) NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_entities_name` (`entity_name`),
  KEY `idx_entities_type` (`entity_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='业务对象（费用归属）';

CREATE TABLE IF NOT EXISTS `rate_customer` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `region` VARCHAR(32) NOT NULL,
  `cp` VARCHAR(32) NOT NULL,
  `school_name` VARCHAR(128) NULL,
  `customer_fee` DECIMAL(18,6) NULL,
  `network_line_fee` DECIMAL(18,6) NULL,
  `general_fee` DECIMAL(18,6) NULL,
  `customer_fee_owner_id` BIGINT UNSIGNED NULL,
  `network_line_fee_owner_id` BIGINT UNSIGNED NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_rate_customer` (`region`,`cp`,`school_name`),
  KEY `idx_rate_customer_region` (`region`),
  KEY `idx_rate_customer_cp` (`cp`),
  KEY `idx_rate_customer_school` (`school_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='客户业务费率（NFA）';

CREATE TABLE IF NOT EXISTS `rate_final_customer` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `region` VARCHAR(32) NOT NULL,
  `cp` VARCHAR(32) NOT NULL,
  `school_name` VARCHAR(128) NOT NULL,
  `fee_type` ENUM('auto','config') NOT NULL DEFAULT 'auto',
  `customer_fee` DECIMAL(18,6) NULL,
  `customer_fee_owner_id` BIGINT UNSIGNED NULL,
  `network_line_fee` DECIMAL(18,6) NULL,
  `network_line_fee_owner_id` BIGINT UNSIGNED NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_rate_final_customer` (`region`,`cp`,`school_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='最终客户费率';

-- 009_business_types.sql + 010_seed_business_types.sql
CREATE TABLE IF NOT EXISTS `business_types` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `code` VARCHAR(64) NOT NULL UNIQUE,
  `name` VARCHAR(128) NOT NULL,
  `description` VARCHAR(255) NULL,
  `enabled` TINYINT(1) NOT NULL DEFAULT 1,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO `business_types` (code, name, description, enabled) VALUES
  ('customer','客户','客户费用归属',1),
  ('line_provider','线路提供商','线路费用归属',1),
  ('node','节点','节点/EDC费用归属',1),
  ('sales','销售','提成/激励归属',1)
ON DUPLICATE KEY UPDATE name=VALUES(name),description=VALUES(description),enabled=VALUES(enabled);

-- 011_init_final_customer_rates.sql（幂等初始化）
INSERT INTO rate_final_customer
  (region, cp, school_name, fee_type, customer_fee, customer_fee_owner_id, network_line_fee, network_line_fee_owner_id, created_at, updated_at)
SELECT rc.region, rc.cp, COALESCE(rc.school_name,'not_a_school'), 'auto', rc.customer_fee, rc.customer_fee_owner_id, rc.network_line_fee, rc.network_line_fee_owner_id, NOW(), NOW()
FROM rate_customer rc
ON DUPLICATE KEY UPDATE
  fee_type = IF(rate_final_customer.fee_type='config', rate_final_customer.fee_type, VALUES(fee_type)),
  customer_fee = IF(rate_final_customer.fee_type='config', rate_final_customer.customer_fee, VALUES(customer_fee)),
  customer_fee_owner_id = IF(rate_final_customer.fee_type='config', rate_final_customer.customer_fee_owner_id, VALUES(customer_fee_owner_id)),
  network_line_fee = IF(rate_final_customer.fee_type='config', rate_final_customer.network_line_fee, VALUES(network_line_fee)),
  network_line_fee_owner_id = IF(rate_final_customer.fee_type='config', rate_final_customer.network_line_fee_owner_id, VALUES(network_line_fee_owner_id)),
  updated_at = NOW();

-- 012_customer_fields_and_rules.sql（守卫式 ALTER + 表）
SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_customer' AND COLUMN_NAME='extra')=0,
  'ALTER TABLE `rate_customer` ADD COLUMN `extra` JSON NULL COMMENT ''自定义扩展字段(JSON)''', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_customer' AND COLUMN_NAME='last_sync_time')=0,
  'ALTER TABLE `rate_customer` ADD COLUMN `last_sync_time` DATETIME NULL COMMENT ''最近同步时间''', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_customer' AND COLUMN_NAME='last_sync_rule_id')=0,
  'ALTER TABLE `rate_customer` ADD COLUMN `last_sync_rule_id` BIGINT NULL COMMENT ''最近生效的同步规则ID''', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- 统一采用 rate_customer_custom_field_defs 定义表（替代早期 rate_customer_fields）
CREATE TABLE IF NOT EXISTS `rate_customer_custom_field_defs` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `field_key` VARCHAR(64) NOT NULL COMMENT '字段键名，建议前缀 ext_，小写字母/数字/下划线',
  `label` VARCHAR(64) NOT NULL COMMENT '字段显示名',
  `data_type` ENUM('string','number','integer','boolean','date','enum') NOT NULL COMMENT '数据类型',
  `required` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否必填',
  `default_value` JSON NULL COMMENT '默认值(JSON)',
  `validate_regex` VARCHAR(255) NULL COMMENT '正则校验，可空',
  `min` DECIMAL(20,6) NULL COMMENT '最小值(数值类型适用)',
  `max` DECIMAL(20,6) NULL COMMENT '最大值(数值类型适用)',
  `precision` INT NULL COMMENT '小数精度(数值类型适用)',
  `enum_options` JSON NULL COMMENT '枚举选项(JSON数组)',
  `usable_in_rules` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否可在同步规则中被设置',
  `enabled` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_rate_customer_field_key` (`field_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='客户费率自定义字段定义';

CREATE TABLE IF NOT EXISTS `rate_customer_sync_rules` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(100) NOT NULL,
  `enabled` TINYINT(1) NOT NULL DEFAULT 1,
  `priority` INT NOT NULL DEFAULT 1000,
  `scope_region` JSON NULL,
  `scope_cp` JSON NULL,
  `condition_expr` TEXT NULL,
  `fields_to_update` JSON NULL,
  `overwrite_strategy` ENUM('only_null','only_auto','always') NOT NULL DEFAULT 'only_null',
  `actions` JSON NOT NULL,
  `created_by` BIGINT NULL,
  `updated_by` BIGINT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_sync_rules_pri_enabled` (`enabled`,`priority`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='客户费率同步规则';

-- 008_settlement_rates.sql: 补充缺失的表定义
CREATE TABLE IF NOT EXISTS `rate_node` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `region` VARCHAR(32) NOT NULL,
  `cp` VARCHAR(32) NOT NULL,
  `cp_fee` DECIMAL(18,6) NULL,
  `cp_fee_owner_id` BIGINT UNSIGNED NULL,
  `node_construction_fee` DECIMAL(18,6) NULL,
  `node_construction_fee_owner_id` BIGINT UNSIGNED NULL,
  `rack_fee` DECIMAL(18,6) NULL,
  `rack_fee_owner_id` BIGINT UNSIGNED NULL,
  `other_fee` DECIMAL(18,6) NULL,
  `other_fee_owner_id` BIGINT UNSIGNED NULL,
  `settlement_type` VARCHAR(16) NOT NULL DEFAULT 'daily95',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_rate_node` (`region`,`cp`,`settlement_type`),
  KEY `idx_rate_node_region` (`region`),
  KEY `idx_rate_node_cp` (`cp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='节点业务费率（EDC）';

CREATE TABLE IF NOT EXISTS `settlement_customer` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `region` VARCHAR(32) NOT NULL,
  `cp` VARCHAR(32) NOT NULL,
  `school_name` VARCHAR(128) NOT NULL,
  `settlement_value` DECIMAL(18,6) NOT NULL,
  `settlement_time` DATETIME NOT NULL,
  `customer_fee` DECIMAL(18,6) NULL,
  `customer_bill` DECIMAL(18,6) NULL,
  `customer_fee_owner_id` BIGINT UNSIGNED NULL,
  `network_line_fee` DECIMAL(18,6) NULL,
  `network_line_bill` DECIMAL(18,6) NULL,
  `network_line_fee_owner_id` BIGINT UNSIGNED NULL,
  `node_deduction_fee` DECIMAL(18,6) NULL,
  `node_deduction_bill` DECIMAL(18,6) NULL,
  `node_deduction_fee_owner_id` BIGINT UNSIGNED NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_settlement_customer_region` (`region`),
  KEY `idx_settlement_customer_cp` (`cp`),
  KEY `idx_settlement_customer_school` (`school_name`),
  KEY `idx_settlement_customer_time` (`settlement_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='客户结算金额';

CREATE TABLE IF NOT EXISTS `settlement_node_daily95` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `region` VARCHAR(32) NOT NULL,
  `cp` VARCHAR(32) NOT NULL,
  `cp_fee` DECIMAL(18,6) NULL,
  `cp_bill` DECIMAL(18,6) NULL,
  `cp_fee_owner_id` BIGINT UNSIGNED NULL,
  `node_construction_fee` DECIMAL(18,6) NULL,
  `node_construction_bill` DECIMAL(18,6) NULL,
  `node_construction_fee_owner_id` BIGINT UNSIGNED NULL,
  `rack_fee` DECIMAL(18,6) NULL,
  `rack_bill` DECIMAL(18,6) NULL,
  `rack_fee_owner_id` BIGINT UNSIGNED NULL,
  `other_fee` DECIMAL(18,6) NULL,
  `other_bill` DECIMAL(18,6) NULL,
  `other_fee_owner_id` BIGINT UNSIGNED NULL,
  `settlement_value` DECIMAL(18,6) NOT NULL,
  `settlement_time` DATETIME NOT NULL,
  `daily95_fee` DECIMAL(18,6) NULL,
  `daily95_bill` DECIMAL(18,6) NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_node_daily95_region` (`region`),
  KEY `idx_node_daily95_cp` (`cp`),
  KEY `idx_node_daily95_time` (`settlement_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='节点日95结算金额';

CREATE TABLE IF NOT EXISTS `settlement_node_monthly95` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `region` VARCHAR(32) NOT NULL,
  `cp` VARCHAR(32) NOT NULL,
  `cp_fee` DECIMAL(18,6) NULL,
  `cp_bill` DECIMAL(18,6) NULL,
  `cp_fee_owner_id` BIGINT UNSIGNED NULL,
  `node_construction_fee` DECIMAL(18,6) NULL,
  `node_construction_bill` DECIMAL(18,6) NULL,
  `node_construction_fee_owner_id` BIGINT UNSIGNED NULL,
  `rack_fee` DECIMAL(18,6) NULL,
  `rack_bill` DECIMAL(18,6) NULL,
  `rack_fee_owner_id` BIGINT UNSIGNED NULL,
  `other_fee` DECIMAL(18,6) NULL,
  `other_bill` DECIMAL(18,6) NULL,
  `other_fee_owner_id` BIGINT UNSIGNED NULL,
  `settlement_value` DECIMAL(18,6) NOT NULL,
  `settlement_time` DATETIME NOT NULL,
  `monthly95_fee` DECIMAL(18,6) NULL,
  `monthly95_bill` DECIMAL(18,6) NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_node_monthly95_region` (`region`),
  KEY `idx_node_monthly95_cp` (`cp`),
  KEY `idx_node_monthly95_time` (`settlement_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='节点月95结算金额';

-- 013_alter_rate_final_customer_add_sync_meta.sql
SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_final_customer' AND COLUMN_NAME='last_sync_time')=0,
  'ALTER TABLE `rate_final_customer` ADD COLUMN `last_sync_time` DATETIME NULL COMMENT ''最近同步时间''', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_final_customer' AND COLUMN_NAME='last_sync_rule_id')=0,
  'ALTER TABLE `rate_final_customer` ADD COLUMN `last_sync_rule_id` BIGINT NULL COMMENT ''最近生效的同步规则ID''', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- rate_final_customer 缺失列补齐：node_deduction_fee / node_deduction_fee_owner_id
SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_final_customer' AND COLUMN_NAME='node_deduction_fee')=0,
  'ALTER TABLE `rate_final_customer` ADD COLUMN `node_deduction_fee` DECIMAL(18,6) NULL AFTER `network_line_fee_owner_id`', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_final_customer' AND COLUMN_NAME='node_deduction_fee_owner_id')=0,
  'ALTER TABLE `rate_final_customer` ADD COLUMN `node_deduction_fee_owner_id` BIGINT UNSIGNED NULL AFTER `node_deduction_fee`', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- 014_create_rate_sync_config.sql
CREATE TABLE IF NOT EXISTS `rate_sync_config` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `enabled` TINYINT(1) NOT NULL DEFAULT 1,
  `default_final_fee` DECIMAL(18,6) NOT NULL DEFAULT 1000,
  `max_batch` INT NOT NULL DEFAULT 1000,
  `notes` VARCHAR(255) NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='费率同步全局配置';

INSERT INTO `rate_sync_config` (`enabled`,`default_final_fee`,`max_batch`,`notes`)
SELECT 1,1000,1000,'默认配置' WHERE NOT EXISTS (SELECT 1 FROM `rate_sync_config`);

-- 015_create_rate_sync_audits.sql
CREATE TABLE IF NOT EXISTS `rate_sync_audits` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `rate_customer_id` BIGINT NULL,
  `region` VARCHAR(32) NOT NULL,
  `cp` VARCHAR(32) NOT NULL,
  `school_name` VARCHAR(128) NULL,
  `rule_id` BIGINT NULL,
  `action` VARCHAR(16) NOT NULL,
  `changed_fields` JSON NOT NULL,
  `overwrite_strategy` VARCHAR(16) NULL,
  `fields_whitelist` JSON NULL,
  `mode_snapshot` JSON NULL,
  `message` VARCHAR(255) NULL,
  `executed_by` BIGINT NULL,
  `executed_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_sync_audits_time` (`executed_at`),
  KEY `idx_sync_audits_rule` (`rule_id`),
  KEY `idx_sync_audits_customer` (`rate_customer_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='费率同步审计日志';

-- 016_alter_rate_customer_add_fee_mode.sql
SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_customer' AND COLUMN_NAME='fee_mode')=0,
  'ALTER TABLE `rate_customer` ADD COLUMN `fee_mode` ENUM(''auto'',''configed'') NOT NULL DEFAULT ''auto'' COMMENT ''配置模式：auto=自动, configed=手工'' AFTER `network_line_fee_owner_id`', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;
UPDATE `rate_customer` SET `fee_mode`='auto' WHERE `fee_mode` IS NULL OR `fee_mode`='';

-- 016_user_schools.sql（修正为 5.7 兼容 Collation）
CREATE TABLE IF NOT EXISTS `user_schools` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `school_id` VARCHAR(64) NOT NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_school` (`user_id`,`school_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_school_id` (`school_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 017_alter_rate_customer_add_general_fee_owner_id.sql（守卫）
SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_customer' AND COLUMN_NAME='general_fee_owner_id')=0,
  'ALTER TABLE `rate_customer` ADD COLUMN `general_fee_owner_id` BIGINT UNSIGNED NULL AFTER `general_fee`', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- 索引守卫
SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_customer' AND INDEX_NAME='idx_rate_customer_general_fee_owner_id')=0,
  'ALTER TABLE `rate_customer` ADD INDEX `idx_rate_customer_general_fee_owner_id` (`general_fee_owner_id`)', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- 017_alter_users_add_alias.sql（改为守卫，兼容 5.7）
SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='users' AND COLUMN_NAME='alias')=0,
  'ALTER TABLE `users` ADD COLUMN `alias` VARCHAR(64) NULL AFTER `username`', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- 018_create_settlement_formulas.sql
CREATE TABLE IF NOT EXISTS `nfa_settlement_formulas` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(64) NOT NULL,
  `description` VARCHAR(255) DEFAULT NULL,
  `tokens` JSON NOT NULL,
  `enabled` TINYINT(1) NOT NULL DEFAULT 1,
  `updated_by` VARCHAR(64) DEFAULT NULL,
  `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `update_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 019_add_settlement_formula_permissions.sql
INSERT INTO `permissions` (`code`,`name`,`description`) VALUES
  ('settlement.formula.read','结算公式-查看','Read settlement formulas'),
  ('settlement.formula.write','结算公式-编辑','Manage settlement formulas')
ON DUPLICATE KEY UPDATE `name`=VALUES(`name`),`description`=VALUES(`description`);

INSERT IGNORE INTO `role_permissions` (`role_id`,`permission_id`)
SELECT r.id, p.id FROM `roles` r JOIN `permissions` p ON p.code IN ('settlement.formula.read','settlement.formula.write') WHERE r.name='admin';

-- 020_create_settlement_results.sql 已在前段建立表与唯一键；此处仅保证权限与唯一键（防重复）
INSERT INTO `permissions`(`code`, `name`, `description`)
VALUES ('settlement.results.read', '结算结果-查看', 'Read settlement results overview')
ON DUPLICATE KEY UPDATE `name`=VALUES(`name`), `description`=VALUES(`description`);

INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`)
SELECT r.id, p.id FROM `roles` r JOIN `permissions` p ON p.code = 'settlement.results.read' WHERE r.name = 'admin';

-- 完毕

-- 额外守卫：nfa_settlement_config.last_execute_time（与代码模型一致）
SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='nfa_settlement_config' AND COLUMN_NAME='last_execute_time')=0,
  'ALTER TABLE `nfa_settlement_config` ADD COLUMN `last_execute_time` DATETIME NULL AFTER `enabled`', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;
