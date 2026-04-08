CREATE TABLE IF NOT EXISTS `rate_customer_filter_rules` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(100) NOT NULL,
  `enabled` TINYINT(1) NOT NULL DEFAULT 1,
  `priority` INT NOT NULL DEFAULT 0,
  `scope_region` JSON NULL,
  `scope_cp` JSON NULL,
  `school_name_match_type` VARCHAR(16) NOT NULL DEFAULT '',
  `school_name_values` JSON NULL,
  `created_by` BIGINT UNSIGNED NULL,
  `updated_by` BIGINT UNSIGNED NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_rate_customer_filter_rules_priority` (`priority`),
  KEY `idx_rate_customer_filter_rules_enabled` (`enabled`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='客户业务费率过滤规则';

INSERT INTO `permissions` (`code`, `name`, `description`) VALUES
  ('rates.filter_rules.read', '客户费率过滤规则查看', '查看客户费率过滤规则列表'),
  ('rates.filter_rules.write', '客户费率过滤规则维护', '新增/修改/删除/排序/启停客户费率过滤规则')
ON DUPLICATE KEY UPDATE
  `name` = VALUES(`name`),
  `description` = VALUES(`description`);

INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`)
SELECT r.id, p.id
FROM `roles` r
JOIN `permissions` p ON p.code IN ('rates.filter_rules.read', 'rates.filter_rules.write')
WHERE r.name = 'admin';
