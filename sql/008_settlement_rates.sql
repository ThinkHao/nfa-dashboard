-- 业务对象表（费用归属方）
CREATE TABLE IF NOT EXISTS `business_entities` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `entity_type` VARCHAR(50) NOT NULL COMMENT '对象类型: customer,line_provider,node,sales 等',
  `entity_name` VARCHAR(100) NOT NULL COMMENT '对象名称，唯一',
  `contact_info` VARCHAR(255) NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_entities_name` (`entity_name`),
  KEY `idx_entities_type` (`entity_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='业务对象（费用归属）';

-- 表1：客户业务费率表（NFA 来源）
CREATE TABLE IF NOT EXISTS `rate_customer` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `region` VARCHAR(32) NOT NULL COMMENT '省份/区域',
  `cp` VARCHAR(32) NOT NULL COMMENT '内容方',
  `school_name` VARCHAR(128) NULL COMMENT '学校名称，留空表示通用',
  `customer_fee` DECIMAL(18,6) NULL COMMENT '客户费率(支出)',
  `network_line_fee` DECIMAL(18,6) NULL COMMENT '线路费率(支出)',
  `general_fee` DECIMAL(18,6) NULL COMMENT '通用费率（未指定具体院校时使用）',
  `customer_fee_owner_id` BIGINT UNSIGNED NULL COMMENT '客户费用归属对象ID',
  `network_line_fee_owner_id` BIGINT UNSIGNED NULL COMMENT '线路费用归属对象ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_rate_customer` (`region`,`cp`,`school_name`),
  KEY `idx_rate_customer_region` (`region`),
  KEY `idx_rate_customer_cp` (`cp`),
  KEY `idx_rate_customer_school` (`school_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='客户业务费率（NFA）';

-- 表2：节点业务费率表（EDC 来源）
CREATE TABLE IF NOT EXISTS `rate_node` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `region` VARCHAR(32) NOT NULL COMMENT '省份/区域',
  `cp` VARCHAR(32) NOT NULL COMMENT '内容方',
  `cp_fee` DECIMAL(18,6) NULL COMMENT '内容方费率(收入)',
  `cp_fee_owner_id` BIGINT UNSIGNED NULL COMMENT '内容方费用归属对象ID',
  `node_construction_fee` DECIMAL(18,6) NULL COMMENT '节点建设费率(支出)',
  `node_construction_fee_owner_id` BIGINT UNSIGNED NULL COMMENT '节点建设费用归属对象ID',
  `rack_fee` DECIMAL(18,6) NULL COMMENT '机柜费(固定支出)',
  `rack_fee_owner_id` BIGINT UNSIGNED NULL COMMENT '机柜费用归属对象ID',
  `other_fee` DECIMAL(18,6) NULL COMMENT '其他固定费用(支出)',
  `other_fee_owner_id` BIGINT UNSIGNED NULL COMMENT '其他费用归属对象ID',
  `settlement_type` VARCHAR(16) NOT NULL DEFAULT 'daily95' COMMENT '结算类型: daily95 | monthly95',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_rate_node` (`region`,`cp`,`settlement_type`),
  KEY `idx_rate_node_region` (`region`),
  KEY `idx_rate_node_cp` (`cp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='节点业务费率（EDC）';

-- 表3：最终客户费率表
CREATE TABLE IF NOT EXISTS `rate_final_customer` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `region` VARCHAR(32) NOT NULL,
  `cp` VARCHAR(32) NOT NULL,
  `school_name` VARCHAR(128) NOT NULL COMMENT '若来源EDC则用 not_a_school 占位',
  `final_fee` DECIMAL(18,6) NULL COMMENT '最终客户费率',
  `fee_type` VARCHAR(16) NOT NULL DEFAULT 'auto' COMMENT 'auto=自动刷写, config=手工配置',
  `customer_fee` DECIMAL(18,6) NULL COMMENT '客户费率(支出)',
  `customer_fee_owner_id` BIGINT UNSIGNED NULL,
  `network_line_fee` DECIMAL(18,6) NULL COMMENT '线路费率(支出)',
  `network_line_fee_owner_id` BIGINT UNSIGNED NULL,
  `node_deduction_fee` DECIMAL(18,6) NULL COMMENT '节点建设倒扣费率(支出)',
  `node_deduction_fee_owner_id` BIGINT UNSIGNED NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_rate_final_customer` (`region`,`cp`,`school_name`),
  KEY `idx_rate_final_customer_region` (`region`),
  KEY `idx_rate_final_customer_cp` (`cp`),
  KEY `idx_rate_final_customer_school` (`school_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='最终客户费率';

-- 表4：客户结算金额表
CREATE TABLE IF NOT EXISTS `settlement_customer` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `region` VARCHAR(32) NOT NULL,
  `cp` VARCHAR(32) NOT NULL,
  `school_name` VARCHAR(128) NOT NULL,
  `settlement_value` DECIMAL(18,6) NOT NULL COMMENT '结算值，如日95/月95数值',
  `settlement_time` DATETIME NOT NULL COMMENT '结算时间',
  `customer_fee` DECIMAL(18,6) NULL,
  `customer_bill` DECIMAL(18,6) NULL,
  `customer_fee_owner_id` BIGINT UNSIGNED NULL,
  `network_line_fee` DECIMAL(18,6) NULL,
  `network_line_bill` DECIMAL(18,6) NULL,
  `network_line_fee_owner_id` BIGINT UNSIGNED NULL,
  `node_deduction_fee` DECIMAL(18,6) NULL,
  `node_deduction_bill` DECIMAL(18,6) NULL,
  `node_deduction_fee_owner_id` BIGINT UNSIGNED NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_settlement_customer_region` (`region`),
  KEY `idx_settlement_customer_cp` (`cp`),
  KEY `idx_settlement_customer_school` (`school_name`),
  KEY `idx_settlement_customer_time` (`settlement_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='客户结算金额';

-- 表5：节点日95结算金额表
CREATE TABLE IF NOT EXISTS `settlement_node_daily95` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `region` VARCHAR(32) NOT NULL,
  `cp` VARCHAR(32) NOT NULL,
  `cp_fee` DECIMAL(18,6) NULL,
  `cp_bill` DECIMAL(18,6) NULL,
  `cp_fee_owner_id` BIGINT UNSIGNED NULL,
  `node_construction_fee` DECIMAL(18,6) NULL,
  `node_construction_bill` DECIMAL(18,6) NULL,
  `node_construction_fee_owner_id` BIGINT UNSIGNED NULL,
  `rack_fee` DECIMAL(18,6) NULL,
  `rack_bill` DECIMAL(18,6) NULL,
  `rack_fee_owner_id` BIGINT UNSIGNED NULL,
  `other_fee` DECIMAL(18,6) NULL,
  `other_bill` DECIMAL(18,6) NULL,
  `other_fee_owner_id` BIGINT UNSIGNED NULL,
  `settlement_value` DECIMAL(18,6) NOT NULL COMMENT '日95值',
  `settlement_time` DATETIME NOT NULL,
  `daily95_fee` DECIMAL(18,6) NULL COMMENT '日95结算单价',
  `daily95_bill` DECIMAL(18,6) NULL COMMENT '日95结算金额',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_node_daily95_region` (`region`),
  KEY `idx_node_daily95_cp` (`cp`),
  KEY `idx_node_daily95_time` (`settlement_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='节点日95结算金额';

-- 表6：节点月95结算金额表
CREATE TABLE IF NOT EXISTS `settlement_node_monthly95` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `region` VARCHAR(32) NOT NULL,
  `cp` VARCHAR(32) NOT NULL,
  `cp_fee` DECIMAL(18,6) NULL,
  `cp_bill` DECIMAL(18,6) NULL,
  `cp_fee_owner_id` BIGINT UNSIGNED NULL,
  `node_construction_fee` DECIMAL(18,6) NULL,
  `node_construction_bill` DECIMAL(18,6) NULL,
  `node_construction_fee_owner_id` BIGINT UNSIGNED NULL,
  `rack_fee` DECIMAL(18,6) NULL,
  `rack_bill` DECIMAL(18,6) NULL,
  `rack_fee_owner_id` BIGINT UNSIGNED NULL,
  `other_fee` DECIMAL(18,6) NULL,
  `other_bill` DECIMAL(18,6) NULL,
  `other_fee_owner_id` BIGINT UNSIGNED NULL,
  `settlement_value` DECIMAL(18,6) NOT NULL COMMENT '月95值',
  `settlement_time` DATETIME NOT NULL,
  `monthly95_fee` DECIMAL(18,6) NULL COMMENT '月95结算单价',
  `monthly95_bill` DECIMAL(18,6) NULL COMMENT '月95结算金额',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_node_monthly95_region` (`region`),
  KEY `idx_node_monthly95_cp` (`cp`),
  KEY `idx_node_monthly95_time` (`settlement_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='节点月95结算金额';
