-- Migration: rate discount rule scopes
-- contract: table=rate_discount_rule_scope

CREATE TABLE IF NOT EXISTS `rate_discount_rule_scope` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `rule_id` BIGINT UNSIGNED NOT NULL,
    `scope_type` VARCHAR(32) NOT NULL,
    `scope_key` VARCHAR(128) NULL,
    `region` VARCHAR(32) NULL,
    `cp` VARCHAR(32) NULL,
    `school_name` VARCHAR(128) NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_discount_scope_rule` (`rule_id`),
    KEY `idx_discount_scope_key` (`scope_type`, `scope_key`),
    KEY `idx_discount_scope_school` (`scope_type`, `region`, `cp`, `school_name`(64)),
    CONSTRAINT `fk_discount_scope_rule` FOREIGN KEY (`rule_id`) REFERENCES `rate_discount_rule` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='客户费率折损规则适用范围';

INSERT INTO `rate_discount_rule_scope` (`rule_id`, `scope_type`, `scope_key`, `region`, `cp`, `school_name`)
SELECT
    r.`id`,
    r.`scope_type`,
    r.`scope_key`,
    CASE WHEN r.`scope_type` = 'region' THEN r.`scope_key` ELSE NULL END,
    CASE WHEN r.`scope_type` = 'cp' THEN r.`scope_key` ELSE NULL END,
    CASE WHEN r.`scope_type` = 'school' THEN r.`scope_key` ELSE NULL END
FROM `rate_discount_rule` r
WHERE NOT EXISTS (
    SELECT 1 FROM `rate_discount_rule_scope` s WHERE s.`rule_id` = r.`id`
);
