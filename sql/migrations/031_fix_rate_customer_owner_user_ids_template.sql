-- 修复模板：将非法归属ID按映射表修正为系统用户ID，无法映射的置空
-- 执行顺序建议：
-- 1) 先执行 030_audit_rate_customer_owner_user_ids.sql 导出待修复清单
-- 2) 填充临时映射表 tmp_rate_customer_owner_user_map
-- 3) 执行本脚本
-- 4) 再次执行 030 审计，确认结果集为空

START TRANSACTION;

-- 映射表（一次性）
DROP TEMPORARY TABLE IF EXISTS tmp_rate_customer_owner_user_map;
CREATE TEMPORARY TABLE tmp_rate_customer_owner_user_map (
  old_owner_id BIGINT UNSIGNED NOT NULL,
  new_user_id BIGINT UNSIGNED NULL,
  PRIMARY KEY (old_owner_id)
);

-- TODO: 填写映射，示例：
-- INSERT INTO tmp_rate_customer_owner_user_map (old_owner_id, new_user_id) VALUES
--   (900001, 101),
--   (900002, 102),
--   (900003, NULL); -- 无法映射则置空

-- customer_fee_owner_id
UPDATE rate_customer rc
LEFT JOIN users u_old ON u_old.id = rc.customer_fee_owner_id
LEFT JOIN tmp_rate_customer_owner_user_map m ON m.old_owner_id = rc.customer_fee_owner_id
LEFT JOIN users u_new ON u_new.id = m.new_user_id
SET rc.customer_fee_owner_id = CASE
  WHEN u_old.id IS NOT NULL THEN rc.customer_fee_owner_id
  WHEN u_new.id IS NOT NULL THEN u_new.id
  ELSE NULL
END
WHERE rc.customer_fee_owner_id IS NOT NULL;

-- network_line_fee_owner_id
UPDATE rate_customer rc
LEFT JOIN users u_old ON u_old.id = rc.network_line_fee_owner_id
LEFT JOIN tmp_rate_customer_owner_user_map m ON m.old_owner_id = rc.network_line_fee_owner_id
LEFT JOIN users u_new ON u_new.id = m.new_user_id
SET rc.network_line_fee_owner_id = CASE
  WHEN u_old.id IS NOT NULL THEN rc.network_line_fee_owner_id
  WHEN u_new.id IS NOT NULL THEN u_new.id
  ELSE NULL
END
WHERE rc.network_line_fee_owner_id IS NOT NULL;

-- general_fee_owner_id
UPDATE rate_customer rc
LEFT JOIN users u_old ON u_old.id = rc.general_fee_owner_id
LEFT JOIN tmp_rate_customer_owner_user_map m ON m.old_owner_id = rc.general_fee_owner_id
LEFT JOIN users u_new ON u_new.id = m.new_user_id
SET rc.general_fee_owner_id = CASE
  WHEN u_old.id IS NOT NULL THEN rc.general_fee_owner_id
  WHEN u_new.id IS NOT NULL THEN u_new.id
  ELSE NULL
END
WHERE rc.general_fee_owner_id IS NOT NULL;

-- channel_owner_user_id
UPDATE rate_customer rc
LEFT JOIN users u_old ON u_old.id = rc.channel_owner_user_id
LEFT JOIN tmp_rate_customer_owner_user_map m ON m.old_owner_id = rc.channel_owner_user_id
LEFT JOIN users u_new ON u_new.id = m.new_user_id
SET rc.channel_owner_user_id = CASE
  WHEN u_old.id IS NOT NULL THEN rc.channel_owner_user_id
  WHEN u_new.id IS NOT NULL THEN u_new.id
  ELSE NULL
END
WHERE rc.channel_owner_user_id IS NOT NULL;

COMMIT;
