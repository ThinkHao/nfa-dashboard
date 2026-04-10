CREATE TABLE IF NOT EXISTS `traffic_scope_rules` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `rule_type` varchar(16) NOT NULL,
  `dimension_type` varchar(16) NOT NULL,
  `dimension_value` varchar(255) NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_traffic_scope_user` (`user_id`),
  KEY `idx_traffic_scope_lookup` (`user_id`, `rule_type`, `dimension_type`),
  CONSTRAINT `chk_traffic_scope_rule_type` CHECK (`rule_type` in ('allow', 'deny')),
  CONSTRAINT `chk_traffic_scope_dimension_type` CHECK (`dimension_type` in ('region', 'cp', 'school'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
