-- 012_customer_fields_and_rules.sql
-- 客户费率：自定义字段定义与同步规则，以及 rate_customer 表新增列
-- 设计目标：
-- 1) 允许通过 Web 定义可扩展字段（保存在 rate_customer.extra JSON 中）
-- 2) 定义可编辑/可排序/可启停的同步规则，将学校信息与客户费率对齐
-- 3) 为 rate_customer 增加同步元数据列

-- 兼容 MySQL 8.x

-- 1) 为 rate_customer 增加列（若不存在）
-- MySQL 某些版本不支持 ALTER TABLE ... ADD COLUMN IF NOT EXISTS，改用 information_schema + 动态 SQL 幂等处理
-- extra
SET @ddl := IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'rate_customer'
       AND COLUMN_NAME = 'extra') = 0,
  'ALTER TABLE `rate_customer` ADD COLUMN `extra` JSON NULL COMMENT ''自定义扩展字段(JSON)''',
  'SELECT 1'
);
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- last_sync_time
SET @ddl := IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'rate_customer'
       AND COLUMN_NAME = 'last_sync_time') = 0,
  'ALTER TABLE `rate_customer` ADD COLUMN `last_sync_time` DATETIME NULL COMMENT ''最近同步时间''',
  'SELECT 1'
);
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- last_sync_rule_id
SET @ddl := IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'rate_customer'
       AND COLUMN_NAME = 'last_sync_rule_id') = 0,
  'ALTER TABLE `rate_customer` ADD COLUMN `last_sync_rule_id` BIGINT NULL COMMENT ''最近生效的同步规则ID''',
  'SELECT 1'
);
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- 可选索引：MySQL 不支持 CREATE INDEX IF NOT EXISTS，建议迁移工具控制幂等。
-- 如需强制创建，可使用如下语句，并忽略“已存在”的错误：
-- ALTER TABLE `rate_customer` ADD INDEX `idx_rate_customer_last_sync_rule_id`(`last_sync_rule_id`);

-- 2) 自定义字段定义表
CREATE TABLE IF NOT EXISTS `rate_customer_custom_field_defs` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `field_key` VARCHAR(64) NOT NULL COMMENT '字段键名，建议前缀 ext_，小写字母/数字/下划线',
  `label` VARCHAR(64) NOT NULL COMMENT '字段显示名',
  `data_type` ENUM('string','number','integer','boolean','date','enum') NOT NULL COMMENT '数据类型',
  `required` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否必填',
  `default_value` JSON NULL COMMENT '默认值(JSON)',
  `validate_regex` VARCHAR(255) NULL COMMENT '正则校验，可空',
  `min` DECIMAL(20,6) NULL COMMENT '最小值(数值类型适用)',
  `max` DECIMAL(20,6) NULL COMMENT '最大值(数值类型适用)',
  `precision` INT NULL COMMENT '小数精度(数值类型适用)',
  `enum_options` JSON NULL COMMENT '枚举选项(JSON数组)',
  `usable_in_rules` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否可在同步规则中被设置',
  `enabled` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_rate_customer_field_key` (`field_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='客户费率自定义字段定义';

-- 3) 同步规则表
CREATE TABLE IF NOT EXISTS `rate_customer_sync_rules` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(100) NOT NULL COMMENT '规则名称',
  `enabled` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '启用状态',
  `priority` INT NOT NULL DEFAULT 1000 COMMENT '优先级，数字越小越先应用',
  `scope_region` JSON NULL COMMENT '作用区域范围(JSON数组/模式)',
  `scope_cp` JSON NULL COMMENT '运营商范围(JSON数组/模式)',
  `condition_expr` TEXT NULL COMMENT '可选条件表达式，返回布尔',
  `fields_to_update` JSON NULL COMMENT '允许本规则更新的字段白名单(JSON数组)',
  `overwrite_strategy` ENUM('only_null','only_auto','always') NOT NULL DEFAULT 'only_null' COMMENT '覆盖策略',
  `actions` JSON NOT NULL COMMENT '规则动作：字段->值(常量/模板/表达式)',
  `created_by` BIGINT NULL,
  `updated_by` BIGINT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_sync_rules_pri_enabled` (`enabled`, `priority`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='客户费率同步规则';

-- 注意：此处未加外键约束以保持部署简便（如需可后续追加 FK 到 last_sync_rule_id）。
