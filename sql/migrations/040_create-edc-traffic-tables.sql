-- Migration: create-edc-traffic-tables
-- contract: table=edc_entities
-- contract: table=edc_traffic_5m
-- contract: table=edc_traffic_scope_rule_groups
-- contract: table=edc_traffic_scope_rule_conditions

CREATE TABLE IF NOT EXISTS `edc_entities` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `edc_name` VARCHAR(100) NOT NULL COMMENT 'EDC 原始设备/业务名称',
  `sn` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '设备序列号，可为空表示按 edc_name 匹配',
  `display_name` VARCHAR(128) NOT NULL COMMENT 'dashboard 展示名称',
  `region` VARCHAR(20) NOT NULL COMMENT '地区',
  `cp` VARCHAR(50) NOT NULL COMMENT '内容方',
  `enabled` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用同步和展示',
  `remark` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '备注',
  `data_hash` CHAR(32) NOT NULL DEFAULT '' COMMENT '映射内容 hash',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_edc_entity_name_sn` (`edc_name`, `sn`),
  KEY `idx_edc_entity_region_cp` (`region`, `cp`),
  KEY `idx_edc_entity_display_name` (`display_name`),
  KEY `idx_edc_entity_enabled` (`enabled`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='EDC 设备/业务映射表';

CREATE TABLE IF NOT EXISTS `edc_traffic_5m` (
  `bucket_5m` DATETIME NOT NULL COMMENT '5 分钟时间桶（本地时区）',
  `entity_id` BIGINT UNSIGNED NOT NULL COMMENT 'EDC 映射实体 ID',
  `region` VARCHAR(20) NOT NULL COMMENT '地区快照',
  `cp` VARCHAR(50) NOT NULL COMMENT '内容方快照',
  `display_name` VARCHAR(128) NOT NULL COMMENT '展示名称快照',
  `service_size` BIGINT NOT NULL DEFAULT 0 COMMENT '服务流量(bytes/5m)',
  `cache_size` BIGINT NOT NULL DEFAULT 0 COMMENT '回源流量(bytes/5m)',
  `record_count` INT NOT NULL DEFAULT 0 COMMENT '源端聚合行数',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`bucket_5m`, `entity_id`),
  KEY `idx_edc_traffic_entity_bucket` (`entity_id`, `bucket_5m`),
  KEY `idx_edc_traffic_region_cp_bucket` (`region`, `cp`, `bucket_5m`),
  KEY `idx_edc_traffic_bucket` (`bucket_5m`),
  CONSTRAINT `fk_edc_traffic_entity` FOREIGN KEY (`entity_id`) REFERENCES `edc_entities` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='EDC 流量 5 分钟事实表';

CREATE TABLE IF NOT EXISTS `edc_traffic_scope_rule_groups` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户 ID',
  `rule_type` VARCHAR(10) NOT NULL COMMENT 'allow 或 deny',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_edc_scope_group_user` (`user_id`),
  CONSTRAINT `chk_edc_scope_group_rule_type` CHECK (`rule_type` in ('allow', 'deny'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='EDC 流量可见范围规则组';

CREATE TABLE IF NOT EXISTS `edc_traffic_scope_rule_conditions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `group_id` BIGINT UNSIGNED NOT NULL COMMENT '规则组 ID',
  `dimension_type` VARCHAR(20) NOT NULL COMMENT 'region/cp/entity',
  `dimension_value` VARCHAR(128) NOT NULL COMMENT '维度值',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_edc_scope_condition_group` (`group_id`),
  KEY `idx_edc_scope_condition_lookup` (`dimension_type`, `dimension_value`),
  CONSTRAINT `chk_edc_scope_condition_dimension_type` CHECK (`dimension_type` in ('region', 'cp', 'entity')),
  CONSTRAINT `fk_edc_scope_condition_group` FOREIGN KEY (`group_id`) REFERENCES `edc_traffic_scope_rule_groups` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='EDC 流量可见范围条件';
