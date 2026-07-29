-- Migration: add daily traffic volume unit base
-- contract: column=nfa_system_settings.daily_traffic_volume_unit_base

ALTER TABLE `nfa_system_settings`
  ADD COLUMN `daily_traffic_volume_unit_base` INT NOT NULL DEFAULT 1000 AFTER `traffic_byte_unit_base`;
