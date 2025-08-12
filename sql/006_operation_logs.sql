-- Operation logs table
CREATE TABLE IF NOT EXISTS `operation_logs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NULL,
  `method` VARCHAR(16) NOT NULL,
  `path` VARCHAR(255) NOT NULL,
  `resource` VARCHAR(128) NULL,
  `action` VARCHAR(64) NULL,
  `status_code` INT NOT NULL,
  `success` TINYINT NOT NULL DEFAULT 1,
  `latency_ms` INT NULL,
  `ip` VARCHAR(64) NULL,
  `user_agent` VARCHAR(255) NULL,
  `error_message` VARCHAR(512) NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_operation_logs_user_id` (`user_id`),
  KEY `idx_operation_logs_created_at` (`created_at`),
  CONSTRAINT `fk_operation_logs_user` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
