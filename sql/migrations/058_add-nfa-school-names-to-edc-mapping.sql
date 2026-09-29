-- Migration: add NFA school names to EDC mapping
-- contract: column=edc_nfa_comparison_groups.nfa_school_names

ALTER TABLE `edc_nfa_comparison_groups`
  ADD COLUMN `nfa_school_names` TEXT NULL COMMENT 'NFA 院校名称范围，JSON 数组；空值表示该区域和 CP 下全部院校' AFTER `nfa_cp`;
