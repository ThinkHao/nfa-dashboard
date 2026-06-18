-- Migration: remove settlement formula and results permissions
-- contract: none
-- 结算公式 / 结算结果（应用公式后的汇总）功能已下线，
-- 对应路由 /settlement/formulas、/settlement/results、/settlement/results/channels 已移除。
-- 此处清理孤立的 RBAC 权限（migrations 019 / 020 引入），保留底层数据表
-- nfa_settlement_formulas / nfa_settlement_results 不做删除（可逆、无运行时引用）。
-- 幂等：可重复执行。
START TRANSACTION;

DELETE FROM `role_permissions`
WHERE `permission_id` IN (
  SELECT `id` FROM `permissions`
  WHERE `code` IN ('settlement.formula.read', 'settlement.formula.write', 'settlement.results.read')
);

DELETE FROM `permissions`
WHERE `code` IN ('settlement.formula.read', 'settlement.formula.write', 'settlement.results.read');

COMMIT;
