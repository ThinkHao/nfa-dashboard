-- Migration: create EDC/NFA traffic comparison mappings
-- contract: table=edc_nfa_comparison_groups
-- contract: table=edc_nfa_comparison_group_members

-- A comparison group is the business-level key used to compare one or more
-- EDC entities with all NFA schools under one node source region and CP.
CREATE TABLE IF NOT EXISTS `edc_nfa_comparison_groups` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `group_name` VARCHAR(128) NOT NULL COMMENT '业务比较组名称',
  `nfa_src_region` VARCHAR(64) NOT NULL COMMENT 'NFA 节点源区域',
  `nfa_cp` VARCHAR(64) NOT NULL COMMENT 'NFA CP',
  `enabled` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
  `remark` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '备注',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_edc_nfa_comparison_group_name` (`group_name`),
  KEY `idx_edc_nfa_comparison_group_filter` (`nfa_src_region`, `nfa_cp`, `enabled`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='EDC 与 NFA 业务比较组';

CREATE TABLE IF NOT EXISTS `edc_nfa_comparison_group_members` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `group_id` BIGINT UNSIGNED NOT NULL COMMENT '比较组 ID',
  `entity_id` BIGINT UNSIGNED NOT NULL COMMENT 'EDC 实体 ID',
  `valid_from` DATETIME DEFAULT NULL COMMENT '成员映射生效时间（含）',
  `valid_to` DATETIME DEFAULT NULL COMMENT '成员映射失效时间（不含）',
  `enabled` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_edc_nfa_comparison_group_member` (`group_id`, `entity_id`, `valid_from`),
  KEY `idx_edc_nfa_comparison_member_entity` (`entity_id`, `enabled`),
  KEY `idx_edc_nfa_comparison_member_time` (`group_id`, `valid_from`, `valid_to`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='EDC 比较组成员及生效区间';

-- 业务确认后再写入实际 entity_id。示例口径：
--   BJ-Bilibili       -> nfa_src_region='北京市', nfa_cp='bilibili'
--   SH-jinshan-01/02  -> 同一比较组，nfa_src_region='上海市', nfa_cp='jinshan'
-- 不在迁移中自动按名称前缀填充成员，避免把未确认的 EDC 节点纳入统计。
