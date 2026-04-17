-- Migration: create-system-settings-table
-- contract: table=nfa_system_settings

CREATE TABLE IF NOT EXISTS `nfa_system_settings` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `hide_non_settlement_schools_in_traffic` TINYINT(1) NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO `nfa_system_settings` (`id`, `hide_non_settlement_schools_in_traffic`)
SELECT 1, 0
WHERE NOT EXISTS (
  SELECT 1 FROM `nfa_system_settings` WHERE `id` = 1
);

