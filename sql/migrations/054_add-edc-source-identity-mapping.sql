-- Migration: add-edc-source-identity-mapping
-- contract: table=edc_entity_source_keys
-- contract: table=edc_entity_switch_events

-- A stable entity_id is the settlement identity.  Source-side (edc_name, sn)
-- values are mutable identities and must be persisted separately.
CREATE TABLE IF NOT EXISTS `edc_entity_source_keys` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `entity_id` BIGINT UNSIGNED NOT NULL,
  `edc_name` VARCHAR(255) NOT NULL,
  `sn` VARCHAR(255) NOT NULL DEFAULT '',
  `collector_ip` VARCHAR(64) DEFAULT NULL,
  `valid_from` DATETIME DEFAULT NULL,
  `valid_to` DATETIME DEFAULT NULL,
  `is_current` TINYINT(1) NOT NULL DEFAULT 0,
  `match_source` VARCHAR(32) NOT NULL DEFAULT 'bootstrap',
  `match_confidence` VARCHAR(16) NOT NULL DEFAULT 'confirmed',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_edc_source_name_sn` (`edc_name`, `sn`),
  KEY `idx_edc_source_entity` (`entity_id`),
  KEY `idx_edc_source_current` (`entity_id`, `is_current`),
  KEY `idx_edc_source_latest` (`edc_name`, `valid_from`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='EDC 源端身份到稳定实体的映射';

CREATE TABLE IF NOT EXISTS `edc_entity_switch_events` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `entity_id` BIGINT UNSIGNED NOT NULL,
  `old_edc_name` VARCHAR(255) DEFAULT NULL,
  `old_sn` VARCHAR(255) DEFAULT NULL,
  `old_collector_ip` VARCHAR(64) DEFAULT NULL,
  `new_edc_name` VARCHAR(255) NOT NULL,
  `new_sn` VARCHAR(255) NOT NULL,
  `new_collector_ip` VARCHAR(64) DEFAULT NULL,
  `first_seen_at` DATETIME NOT NULL,
  `switched_at` DATETIME DEFAULT NULL,
  `decision` VARCHAR(32) NOT NULL,
  `reason` VARCHAR(500) DEFAULT NULL,
  `execution_id` BIGINT UNSIGNED DEFAULT NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_edc_switch_entity_time` (`entity_id`, `first_seen_at`),
  KEY `idx_edc_switch_decision_time` (`decision`, `first_seen_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='EDC 源端身份切换审计事件';
