-- Migration: create traffic report plans, runs and artifacts
-- contract: table=traffic_report_tasks
-- contract: table=traffic_report_runs
-- contract: table=traffic_report_artifacts
-- First-version export tasks are owned by the creating user and retain the
-- resolved query/window snapshot used to produce each artifact.

CREATE TABLE IF NOT EXISTS `traffic_report_tasks` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `owner_user_id` bigint unsigned NOT NULL,
  `name` varchar(200) NOT NULL,
  `kind` varchar(20) NOT NULL DEFAULT 'one_off',
  `active` tinyint(1) NOT NULL DEFAULT 1,
  `data_source_type` varchar(20) NOT NULL DEFAULT 'nfa',
  `schedule_type` varchar(20) DEFAULT NULL,
  `schedule_expr` varchar(100) DEFAULT NULL,
  `timezone` varchar(64) NOT NULL DEFAULT 'Asia/Shanghai',
  `window_selector` varchar(32) NOT NULL DEFAULT 'custom',
  `window_params` json,
  `params` json NOT NULL,
  `export_formats` json NOT NULL,
  `next_run_at` datetime DEFAULT NULL,
  `last_run_at` datetime DEFAULT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_traffic_report_tasks_owner` (`owner_user_id`),
  KEY `idx_traffic_report_tasks_due` (`active`, `next_run_at`),
  CONSTRAINT `fk_traffic_report_tasks_owner` FOREIGN KEY (`owner_user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `traffic_report_runs` (
  `id` varchar(36) NOT NULL,
  `task_id` bigint unsigned NOT NULL,
  `status` varchar(20) NOT NULL DEFAULT 'pending',
  `progress_pct` int NOT NULL DEFAULT 0,
  `progress_stage` varchar(200) DEFAULT NULL,
  `resolved_window` json,
  `resolved_params` json,
  `engine_version` varchar(32) NOT NULL DEFAULT 'go-v1',
  `row_count` bigint NOT NULL DEFAULT 0,
  `summary` json,
  `warning_message` text,
  `error_message` text,
  `started_at` datetime DEFAULT NULL,
  `finished_at` datetime DEFAULT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_traffic_report_runs_task` (`task_id`, `created_at`),
  CONSTRAINT `fk_traffic_report_runs_task` FOREIGN KEY (`task_id`) REFERENCES `traffic_report_tasks` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `traffic_report_artifacts` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `run_id` varchar(36) NOT NULL,
  `file_name` varchar(255) NOT NULL,
  `media_type` varchar(128) NOT NULL,
  `storage_path` varchar(500) NOT NULL,
  `file_size` bigint unsigned NOT NULL DEFAULT 0,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_traffic_report_artifacts_run` (`run_id`),
  CONSTRAINT `fk_traffic_report_artifacts_run` FOREIGN KEY (`run_id`) REFERENCES `traffic_report_runs` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO `permissions` (`code`, `name`, `description`)
VALUES
  ('traffic.report.read', '流量报表查看', '查看导出计划、运行记录和下载文件'),
  ('traffic.report.write', '流量报表创建', '创建、执行、暂停流量报表计划')
ON DUPLICATE KEY UPDATE `name` = VALUES(`name`), `description` = VALUES(`description`);

INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`)
SELECT r.id, p.id
FROM `roles` r
JOIN `permissions` p ON p.code IN ('traffic.report.read', 'traffic.report.write')
WHERE r.name = 'admin';
