-- Migration: customer-rate-todos
-- contract: table=nfa_todos
-- contract: table=nfa_todo_assignees
-- contract: table=nfa_todo_events
-- contract: table=nfa_todo_settings

CREATE TABLE IF NOT EXISTS `nfa_todo_settings` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `customer_rate_user_ids` JSON NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO `nfa_todo_settings` (`id`, `customer_rate_user_ids`)
SELECT 1, NULL
WHERE NOT EXISTS (SELECT 1 FROM `nfa_todo_settings` WHERE `id` = 1);

CREATE TABLE IF NOT EXISTS `nfa_todos` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `todo_type` VARCHAR(64) NOT NULL,
  `target_id` BIGINT UNSIGNED NOT NULL,
  `status` VARCHAR(16) NOT NULL DEFAULT 'pending',
  `completed_at` DATETIME NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_nfa_todos_type_target` (`todo_type`, `target_id`),
  KEY `idx_nfa_todos_status_created` (`status`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `nfa_todo_assignees` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `todo_id` BIGINT UNSIGNED NOT NULL,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `ignored` TINYINT(1) NOT NULL DEFAULT 0,
  `ignored_at` DATETIME NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_nfa_todo_assignees_todo_user` (`todo_id`, `user_id`),
  KEY `idx_nfa_todo_assignees_user_ignored` (`user_id`, `ignored`),
  CONSTRAINT `fk_nfa_todo_assignees_todo` FOREIGN KEY (`todo_id`) REFERENCES `nfa_todos` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_nfa_todo_assignees_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `nfa_todo_events` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `todo_id` BIGINT UNSIGNED NOT NULL,
  `actor_user_id` BIGINT UNSIGNED NULL,
  `event_type` VARCHAR(32) NOT NULL,
  `old_status` VARCHAR(16) NULL,
  `new_status` VARCHAR(16) NULL,
  `detail` VARCHAR(255) NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_nfa_todo_events_todo_created` (`todo_id`, `created_at`),
  CONSTRAINT `fk_nfa_todo_events_todo` FOREIGN KEY (`todo_id`) REFERENCES `nfa_todos` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_nfa_todo_events_actor` FOREIGN KEY (`actor_user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
