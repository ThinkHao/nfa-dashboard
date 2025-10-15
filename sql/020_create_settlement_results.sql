-- 结算结果缓存表与权限配置
START TRANSACTION;

CREATE TABLE IF NOT EXISTS `nfa_settlement_results` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `formula_id` BIGINT UNSIGNED NOT NULL COMMENT '引用 nfa_settlement_formulas.id',
  `formula_name` VARCHAR(128) NOT NULL COMMENT '公式名称快照',
  `formula_tokens` JSON NOT NULL COMMENT '公式 Token JSON 快照',
  `region` VARCHAR(64) NOT NULL COMMENT '省份/区域',
  `cp` VARCHAR(64) NOT NULL COMMENT '运营商/内容方',
  `school_id` VARCHAR(64) NOT NULL COMMENT '院校 ID',
  `school_name` VARCHAR(255) NOT NULL COMMENT '院校名称',
  `start_date` DATE NOT NULL COMMENT '结算起始日期（闭区间）',
  `end_date` DATE NOT NULL COMMENT '结算截止日期（闭区间）',
  `billing_days` INT NOT NULL DEFAULT 0 COMMENT '参与计算的天数',
  `total_95_flow` DECIMAL(20,6) NOT NULL DEFAULT 0 COMMENT '区间内 95 值累计（GB）',
  `average_95_flow` DECIMAL(20,6) NOT NULL DEFAULT 0 COMMENT '区间平均 95 值（GB）',
  `customer_fee` DECIMAL(18,6) NULL COMMENT '客户费率',
  `network_line_fee` DECIMAL(18,6) NULL COMMENT '线路费率',
  `node_deduction_fee` DECIMAL(18,6) NULL COMMENT '节点抵扣费率',
  `final_fee` DECIMAL(18,6) NULL COMMENT '最终客户费率',
  `amount` DECIMAL(20,6) NULL COMMENT '公式计算得出的结算金额',
  `currency` VARCHAR(8) NOT NULL DEFAULT 'CNY' COMMENT '币种',
  `missing_days` INT NOT NULL DEFAULT 0 COMMENT '缺失的天数',
  `missing_fields` JSON NULL COMMENT '缺失字段列表',
  `calculation_detail` JSON NULL COMMENT '计算明细 JSON',
  `calculated_by` BIGINT UNSIGNED NULL COMMENT '执行计算的用户 ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_formula_school_range` (`formula_id`, `school_id`, `region`, `cp`, `start_date`, `end_date`),
  KEY `idx_formula_range` (`formula_id`, `start_date`, `end_date`),
  KEY `idx_school` (`school_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='结算公式结果缓存（展示/复用）';

-- 权限：结算结果查看
INSERT INTO `permissions`(`code`, `name`, `description`)
VALUES ('settlement.results.read', '结算结果-查看', 'Read settlement results overview')
ON DUPLICATE KEY UPDATE `name` = VALUES(`name`), `description` = VALUES(`description`);

-- 授权给 admin 角色
INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`)
SELECT r.id, p.id
FROM `roles` r
JOIN `permissions` p ON p.code = 'settlement.results.read'
WHERE r.name = 'admin';

COMMIT;
