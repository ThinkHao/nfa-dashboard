-- 015_create_rate_sync_audits.sql
-- 费率同步审计表

CREATE TABLE IF NOT EXISTS `rate_sync_audits` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `rate_customer_id` BIGINT NULL COMMENT '受影响的 rate_customer.id',
  `region` VARCHAR(32) NOT NULL,
  `cp` VARCHAR(32) NOT NULL,
  `school_name` VARCHAR(128) NULL,
  `rule_id` BIGINT NULL COMMENT '触发的规则ID',
  `action` VARCHAR(16) NOT NULL COMMENT 'set|nullify|skip',
  `changed_fields` JSON NOT NULL COMMENT '变更字段及旧/新值列表',
  `overwrite_strategy` VARCHAR(16) NULL,
  `fields_whitelist` JSON NULL,
  `mode_snapshot` JSON NULL COMMENT '*_mode 快照',
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

-- 说明：为部署简便未加外键；若需要可在后续迁移增加 FK。
