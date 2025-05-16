-- 结算配置表
CREATE TABLE IF NOT EXISTS `nfa_settlement_config` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `daily_time` varchar(5) NOT NULL DEFAULT '02:00' COMMENT '每日结算时间，格式为"02:00"',
  `weekly_day` int(1) NOT NULL DEFAULT '1' COMMENT '每周结算日，1-7表示周一到周日',
  `weekly_time` varchar(5) NOT NULL DEFAULT '02:00' COMMENT '每周结算时间，格式为"02:00"',
  `enabled` tinyint(1) NOT NULL DEFAULT '1' COMMENT '是否启用',
  `last_execute_time` datetime DEFAULT NULL COMMENT '上次执行时间',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='结算配置表';

-- 初始化配置
INSERT INTO `nfa_settlement_config` (`daily_time`, `weekly_day`, `weekly_time`, `enabled`) 
VALUES ('02:00', 1, '02:00', 1);

-- 结算任务记录表
CREATE TABLE IF NOT EXISTS `nfa_settlement_task` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `task_type` varchar(16) NOT NULL COMMENT '任务计算周期：daily(每日计算前一天)、weekly(每周计算前一周每天)',
  `task_date` date NOT NULL COMMENT '任务日期',
  `status` varchar(16) NOT NULL COMMENT '状态：pending、running、success、failed',
  `start_time` datetime DEFAULT NULL COMMENT '开始时间',
  `end_time` datetime DEFAULT NULL COMMENT '结束时间',
  `processed_count` int(11) NOT NULL DEFAULT '0' COMMENT '处理记录数',
  `error_message` text COMMENT '错误信息',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_task_date` (`task_date`),
  KEY `idx_task_type` (`task_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='结算任务记录表';
