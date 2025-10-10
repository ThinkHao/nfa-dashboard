-- Add alias column to users table
-- MySQL 8.0+
ALTER TABLE `users`
  ADD COLUMN IF NOT EXISTS `alias` VARCHAR(64) NULL AFTER `username`;

-- For MySQL < 8.0 (no IF NOT EXISTS). If running on <8.0, comment the above and use this guarded pattern:
-- NOTE: Run the following block manually if needed; not all clients allow procedural statements in plain SQL files.
-- DELIMITER //
-- CREATE PROCEDURE add_alias_column_if_missing()
-- BEGIN
--   IF NOT EXISTS (
--     SELECT 1 FROM INFORMATION_SCHEMA.COLUMNS
--     WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'users' AND COLUMN_NAME = 'alias'
--   ) THEN
--     ALTER TABLE `users` ADD COLUMN `alias` VARCHAR(64) NULL AFTER `username`;
--   END IF;
-- END //
-- DELIMITER ;
-- CALL add_alias_column_if_missing();
-- DROP PROCEDURE IF EXISTS add_alias_column_if_missing;

-- Optional: add an index on alias if you will query by it frequently
-- CREATE INDEX idx_users_alias ON `users` (`alias`);

-- Rollback (down):
-- ALTER TABLE `users` DROP COLUMN `alias`;
