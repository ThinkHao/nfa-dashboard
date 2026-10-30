-- Migration: preserve traffic report history when removing tasks
-- contract: column=traffic_report_tasks.deleted_at

ALTER TABLE `traffic_report_tasks`
  ADD COLUMN `deleted_at` DATETIME NULL,
  ADD KEY `idx_traffic_report_tasks_deleted_at` (`deleted_at`);
