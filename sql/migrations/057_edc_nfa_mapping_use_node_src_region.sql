-- Migrate EDC/NFA comparison mappings from school province to NFA node source region.
-- contract: column=edc_nfa_comparison_groups.nfa_src_region
SET @has_new_column := (
  SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'edc_nfa_comparison_groups'
    AND COLUMN_NAME = 'nfa_src_region'
);
SET @has_old_column := (
  SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'edc_nfa_comparison_groups'
    AND COLUMN_NAME = 'nfa_region'
);
SET @ddl := IF(
  @has_new_column = 0 AND @has_old_column = 1,
  'ALTER TABLE `edc_nfa_comparison_groups` CHANGE COLUMN `nfa_region` `nfa_src_region` VARCHAR(64) NOT NULL COMMENT ''NFA 节点源区域''',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @has_index := (
  SELECT COUNT(*) FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'edc_nfa_comparison_groups'
    AND INDEX_NAME = 'idx_edc_nfa_comparison_group_filter'
);
SET @ddl := IF(
  @has_index = 0,
  'ALTER TABLE `edc_nfa_comparison_groups` ADD KEY `idx_edc_nfa_comparison_group_filter` (`nfa_src_region`, `nfa_cp`, `enabled`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
