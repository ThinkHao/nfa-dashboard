-- Migration: add nfa school source region
-- contract: column=nfa_school.src_region
-- src_region remains nullable until the separately reviewed backfill is executed.
SET @col_exists := (
  SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'nfa_school'
    AND COLUMN_NAME = 'src_region'
);
SET @ddl := IF(
  @col_exists = 0,
  'ALTER TABLE `nfa_school` ADD COLUMN `src_region` varchar(20) NULL COMMENT ''服务源区域'' AFTER `region`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
-- Replace `none` with one or more contract lines when this migration adds runtime-required schema.
-- contract: table=example_table
-- contract: column=example_table.example_column
-- Add idempotent SQL here.
-- If this migration introduces runtime-required tables or columns,
-- update scripts/offline-deploy.sh assert_db_schema() before release.

