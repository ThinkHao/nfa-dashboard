CREATE TABLE IF NOT EXISTS `traffic_scope_rule_groups` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `rule_type` varchar(16) NOT NULL,
  `legacy_rule_id` bigint unsigned DEFAULT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_traffic_scope_legacy_rule_id` (`legacy_rule_id`),
  KEY `idx_traffic_scope_group_user` (`user_id`),
  CONSTRAINT `chk_traffic_scope_group_rule_type` CHECK (`rule_type` in ('allow', 'deny'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `traffic_scope_rule_conditions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `group_id` bigint unsigned NOT NULL,
  `dimension_type` varchar(16) NOT NULL,
  `dimension_value` varchar(255) NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_traffic_scope_condition_group` (`group_id`),
  CONSTRAINT `chk_traffic_scope_condition_dimension_type` CHECK (`dimension_type` in ('region', 'cp', 'school')),
  CONSTRAINT `fk_traffic_scope_condition_group` FOREIGN KEY (`group_id`) REFERENCES `traffic_scope_rule_groups` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO `traffic_scope_rule_groups` (`user_id`, `rule_type`, `legacy_rule_id`, `created_at`, `updated_at`)
SELECT `user_id`, `rule_type`, `id`, `created_at`, `updated_at`
FROM `traffic_scope_rules`
WHERE NOT EXISTS (
  SELECT 1
  FROM `traffic_scope_rule_groups` g
  WHERE g.`legacy_rule_id` = `traffic_scope_rules`.`id`
);

INSERT INTO `traffic_scope_rule_conditions` (`group_id`, `dimension_type`, `dimension_value`, `created_at`)
SELECT g.`id`, r.`dimension_type`, r.`dimension_value`, r.`created_at`
FROM `traffic_scope_rules` r
JOIN `traffic_scope_rule_groups` g ON g.`legacy_rule_id` = r.`id`
WHERE NOT EXISTS (
  SELECT 1
  FROM `traffic_scope_rule_conditions` c
  WHERE c.`group_id` = g.`id`
);
