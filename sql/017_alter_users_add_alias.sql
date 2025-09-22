-- Add alias column to users table if not exists, and backfill optional display labels
ALTER TABLE `users`
  ADD COLUMN `alias` VARCHAR(64) NULL AFTER `username`;

-- Optional: if you want to backfill alias from username where alias is NULL and username not empty, uncomment below
-- UPDATE `users` SET `alias` = `username` WHERE `alias` IS NULL AND `username` <> '';
