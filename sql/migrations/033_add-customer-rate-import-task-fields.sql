-- Migration: add customer rate import task fields
-- contract: column=nfa_settlement_task.task_stage
-- contract: column=nfa_settlement_task.total_count
-- contract: column=nfa_settlement_task.task_meta

SET @ddl := IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
   WHERE TABLE_SCHEMA = DATABASE()
     AND TABLE_NAME = 'nfa_settlement_task'
     AND COLUMN_NAME = 'task_stage') = 0,
  'ALTER TABLE `nfa_settlement_task`
     ADD COLUMN `task_stage` VARCHAR(32) NOT NULL DEFAULT '''' COMMENT ''任务阶段'' AFTER `status`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
   WHERE TABLE_SCHEMA = DATABASE()
     AND TABLE_NAME = 'nfa_settlement_task'
     AND COLUMN_NAME = 'total_count') = 0,
  'ALTER TABLE `nfa_settlement_task`
     ADD COLUMN `total_count` INT NOT NULL DEFAULT 0 COMMENT ''总处理数'' AFTER `processed_count`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
   WHERE TABLE_SCHEMA = DATABASE()
     AND TABLE_NAME = 'nfa_settlement_task'
     AND COLUMN_NAME = 'task_meta') = 0,
  'ALTER TABLE `nfa_settlement_task`
     ADD COLUMN `task_meta` LONGTEXT NULL COMMENT ''任务元数据(JSON)'' AFTER `error_message`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

