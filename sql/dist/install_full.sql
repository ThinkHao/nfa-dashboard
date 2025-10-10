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
  -- Unique key for settlement results
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

  -- Add general_fee_owner_id to rate_final_customer if missing
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'rate_final_customer'
      AND COLUMN_NAME = 'general_fee_owner_id'
  ) THEN
    SET @stmt := 'ALTER TABLE `rate_final_customer` ADD COLUMN `general_fee_owner_id` BIGINT NULL';
    PREPARE s2 FROM @stmt; EXECUTE s2; DEALLOCATE PREPARE s2;
  END IF;

  -- Seed permission for settlement results
  INSERT INTO `permissions`(`code`,`name`,`description`)
  VALUES ('settlement.results.read','结算结果-查看','Read settlement results overview')
  ON DUPLICATE KEY UPDATE `name`=VALUES(`name`),`description`=VALUES(`description`);

  -- Grant to admin role if exists
  INSERT IGNORE INTO `role_permissions` (`role_id`,`permission_id`)
  SELECT r.id, p.id
  FROM `roles` r JOIN `permissions` p ON p.code = 'settlement.results.read'
  WHERE r.name = 'admin';
END//
DELIMITER ;
CALL apply_migrations();
DROP PROCEDURE IF EXISTS apply_migrations;

-- End of script
