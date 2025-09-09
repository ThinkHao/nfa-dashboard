-- 014_create_rate_sync_config.sql
-- 全局费率同步配置

CREATE TABLE IF NOT EXISTS `rate_sync_config` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `enabled` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用自动同步',
  `default_final_fee` DECIMAL(18,6) NOT NULL DEFAULT 1000 COMMENT '未命中规则时 final_fee 默认值',
  `max_batch` INT NOT NULL DEFAULT 1000 COMMENT '每批处理最大记录数',
  `notes` VARCHAR(255) NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='费率同步全局配置';

-- 若表为空，插入默认配置
INSERT INTO `rate_sync_config` (`enabled`, `default_final_fee`, `max_batch`, `notes`)
SELECT 1, 1000, 1000, '默认配置'
WHERE NOT EXISTS (SELECT 1 FROM `rate_sync_config`);
