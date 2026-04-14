-- contract: none
-- 可选约束：为 rate_customer 四个归属字段增加 users 外键
-- 前置条件：先完成 030/031 审计与修复，确保不存在非法用户ID

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_customer' AND CONSTRAINT_NAME='fk_rate_customer_customer_fee_owner_user')=0,
  'ALTER TABLE `rate_customer` ADD CONSTRAINT `fk_rate_customer_customer_fee_owner_user` FOREIGN KEY (`customer_fee_owner_id`) REFERENCES `users`(`id`) ON DELETE SET NULL ON UPDATE CASCADE', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_customer' AND CONSTRAINT_NAME='fk_rate_customer_network_line_fee_owner_user')=0,
  'ALTER TABLE `rate_customer` ADD CONSTRAINT `fk_rate_customer_network_line_fee_owner_user` FOREIGN KEY (`network_line_fee_owner_id`) REFERENCES `users`(`id`) ON DELETE SET NULL ON UPDATE CASCADE', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_customer' AND CONSTRAINT_NAME='fk_rate_customer_general_fee_owner_user')=0,
  'ALTER TABLE `rate_customer` ADD CONSTRAINT `fk_rate_customer_general_fee_owner_user` FOREIGN KEY (`general_fee_owner_id`) REFERENCES `users`(`id`) ON DELETE SET NULL ON UPDATE CASCADE', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @ddl := IF((SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='rate_customer' AND CONSTRAINT_NAME='fk_rate_customer_channel_owner_user')=0,
  'ALTER TABLE `rate_customer` ADD CONSTRAINT `fk_rate_customer_channel_owner_user` FOREIGN KEY (`channel_owner_user_id`) REFERENCES `users`(`id`) ON DELETE SET NULL ON UPDATE CASCADE', 'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;
