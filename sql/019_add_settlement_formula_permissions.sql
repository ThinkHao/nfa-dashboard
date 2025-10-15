-- 新增结算公式权限并授予 admin 角色（幂等）
START TRANSACTION;

-- permissions 表中插入
INSERT INTO `permissions` (`code`, `name`, `description`) VALUES
  ('settlement.formula.read', '结算公式-查看', 'Read settlement formulas'),
  ('settlement.formula.write', '结算公式-编辑', 'Manage settlement formulas')
ON DUPLICATE KEY UPDATE `name`=VALUES(`name`), `description`=VALUES(`description`);

-- 授权给 admin 角色
INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`)
SELECT r.id, p.id
FROM `roles` r
JOIN `permissions` p ON p.code IN ('settlement.formula.read','settlement.formula.write')
WHERE r.name = 'admin';

COMMIT;
