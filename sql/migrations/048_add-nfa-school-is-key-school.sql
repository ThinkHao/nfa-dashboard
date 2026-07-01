-- Migration: add nfa school is key school
-- contract: none
-- 为 nfa_school 增加 is_key_school 列（重点院校标记，源 nfa_ipgroup.check_status OR 聚合）。
-- 该列由 extractor 写入，dashboard 侧非运行时必需（含 DEFAULT 0），故 contract 标记为 none。
-- 幂等：列与索引存在时不重复添加。
SET @col_exists := (
  SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'nfa_school'
    AND COLUMN_NAME = 'is_key_school'
);
SET @ddl := IF(
  @col_exists = 0,
  'ALTER TABLE `nfa_school` ADD COLUMN `is_key_school` tinyint(1) NOT NULL DEFAULT 0 COMMENT ''是否重点院校(源 nfa_ipgroup.check_status OR 聚合)''',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists := (
  SELECT COUNT(*) FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'nfa_school'
    AND INDEX_NAME = 'idx_is_key_school'
);
SET @ddl := IF(
  @idx_exists = 0,
  'ALTER TABLE `nfa_school` ADD KEY `idx_is_key_school` (`is_key_school`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
