-- Seed admin user, role, permissions and relations (idempotent)

START TRANSACTION;

-- 1) Roles
INSERT INTO `roles` (`name`, `description`)
VALUES ('admin', 'System Administrator')
ON DUPLICATE KEY UPDATE `description` = VALUES(`description`);

-- 2) Permissions
INSERT INTO `permissions` (`code`, `name`, `description`) VALUES
  ('rates.customer.read', '客户费率-查看', 'View customer rate'),
  ('rates.customer.write', '客户费率-编辑', 'Edit customer rate'),
  ('rates.node.read', '节点费率-查看', 'View node rate'),
  ('rates.node.write', '节点费率-编辑', 'Edit node rate'),
  ('rates.final.read', '最终费率-查看', 'View final customer rate'),
  ('rates.final.write', '最终费率-编辑', 'Edit final customer rate'),
  ('settlement.calculate', '结算-计算', 'Run settlement calculation'),
  ('settlement.export', '结算-导出', 'Export settlement data'),
  ('bizobject.manage', '业务对象-管理', 'Manage business objects'),
  ('baseconfig.manage', '进制配置-管理', 'Manage base config'),
  ('operation_logs.read', '操作日志-查看', 'Read operation logs'),
  ('system.user.manage', '系统用户-管理', 'Manage system users'),
  ('system.role.manage', '系统角色-管理', 'Manage system roles')
ON DUPLICATE KEY UPDATE `name` = VALUES(`name`), `description` = VALUES(`description`);

-- 3) Admin user (password = admin123, bcrypt hash provided)
INSERT INTO `users` (`username`, `password_hash`, `email`, `phone`, `status`) 
VALUES ('admin', '$2a$10$OKljOBETJDI9ZlEpskETjOrp8796zWl/z5JEqeCAHl3nRG/mYnjO6', 'admin@example.com', NULL, 1)
ON DUPLICATE KEY UPDATE 
  `password_hash` = VALUES(`password_hash`),
  `email` = VALUES(`email`),
  `status` = VALUES(`status`);

-- 4) Relations: user->role and role->permissions
-- user_roles
INSERT IGNORE INTO `user_roles` (`user_id`, `role_id`)
SELECT u.id, r.id FROM `users` u JOIN `roles` r ON r.name = 'admin' WHERE u.username = 'admin';

-- role_permissions (grant all listed permissions to admin role)
INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`)
SELECT r.id, p.id
FROM `roles` r
JOIN `permissions` p ON p.code IN (
  'rates.customer.read','rates.customer.write',
  'rates.node.read','rates.node.write',
  'rates.final.read','rates.final.write',
  'settlement.calculate','settlement.export',
  'bizobject.manage','baseconfig.manage',
  'operation_logs.read','system.user.manage','system.role.manage'
)
WHERE r.name = 'admin';

COMMIT;
