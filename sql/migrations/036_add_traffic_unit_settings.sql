-- Migration: add-traffic-unit-settings
-- contract: column=nfa_system_settings.traffic_byte_unit_base

ALTER TABLE `nfa_system_settings`
  ADD COLUMN `traffic_byte_unit_base` INT NOT NULL DEFAULT 1024 AFTER `hide_non_settlement_schools_in_traffic`,
  ADD COLUMN `settlement_result_unit_base` INT NOT NULL DEFAULT 1024 AFTER `traffic_byte_unit_base`,
  ADD COLUMN `settlement_data_rate_unit` VARCHAR(16) NOT NULL DEFAULT 'Mbps' AFTER `settlement_result_unit_base`,
  ADD COLUMN `settlement_daily_detail_rate_unit` VARCHAR(16) NOT NULL DEFAULT 'Mbps' AFTER `settlement_data_rate_unit`,
  ADD COLUMN `settlement_single_user_rate_unit` VARCHAR(16) NOT NULL DEFAULT 'Gbps' AFTER `settlement_daily_detail_rate_unit`;

UPDATE `nfa_system_settings`
SET
  `traffic_byte_unit_base` = 1024,
  `settlement_result_unit_base` = 1024,
  `settlement_data_rate_unit` = 'Mbps',
  `settlement_daily_detail_rate_unit` = 'Mbps',
  `settlement_single_user_rate_unit` = 'Gbps'
WHERE `id` = 1;
