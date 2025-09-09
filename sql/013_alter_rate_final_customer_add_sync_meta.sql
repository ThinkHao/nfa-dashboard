-- 013_alter_rate_final_customer_add_sync_meta.sql
-- 为 rate_final_customer 增加同步元数据列（若不存在）
-- 兼容 MySQL 8.x；使用 information_schema + 动态 SQL 确保幂等

-- last_sync_time
SET @ddl := IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'rate_final_customer'
       AND COLUMN_NAME = 'last_sync_time') = 0,
  'ALTER TABLE `rate_final_customer` ADD COLUMN `last_sync_time` DATETIME NULL COMMENT ''最近同步时间''',
  'SELECT 1'
);
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- last_sync_rule_id
SET @ddl := IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'rate_final_customer'
       AND COLUMN_NAME = 'last_sync_rule_id') = 0,
  'ALTER TABLE `rate_final_customer` ADD COLUMN `last_sync_rule_id` BIGINT NULL COMMENT ''最近生效的同步规则ID''',
  'SELECT 1'
);
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- 可选索引（建议由迁移工具保证幂等）：
-- ALTER TABLE `rate_final_customer` ADD INDEX `idx_final_customer_last_sync_rule_id`(`last_sync_rule_id`);
