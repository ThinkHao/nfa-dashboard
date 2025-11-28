-- 增加自动结算与复算相关开关字段到 nfa_settlement_config
-- 保持默认值为 1（true），以维持旧行为

ALTER TABLE `nfa_settlement_config`
  ADD COLUMN `daily_enabled` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用每日自动结算' AFTER `enabled`,
  ADD COLUMN `weekly_enabled` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用每周自动结算' AFTER `daily_enabled`,
  ADD COLUMN `recalc_after_daily` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '日结算完成后是否自动复算' AFTER `weekly_enabled`,
  ADD COLUMN `recalc_after_weekly` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '周结算完成后是否自动复算' AFTER `recalc_after_daily`;

-- 可选：将已有记录统一置为启用，确保即刻与旧逻辑一致
UPDATE `nfa_settlement_config`
   SET `daily_enabled` = 1,
       `weekly_enabled` = 1,
       `recalc_after_daily` = 1,
       `recalc_after_weekly` = 1;
