-- 院校日95结算数据表
CREATE TABLE IF NOT EXISTS `nfa_school_settlement` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `school_id` varchar(64) NOT NULL COMMENT '院校ID',
  `school_name` varchar(255) NOT NULL COMMENT '院校名称',
  `region` varchar(64) NOT NULL COMMENT '省份',
  `cp` varchar(64) NOT NULL COMMENT '运营商',
  `settlement_value` bigint(20) NOT NULL DEFAULT '0' COMMENT '日95结算值(bits/s)',
  `settlement_time` datetime NOT NULL COMMENT '结算时间点(对应95值出现的时间)',
  `settlement_date` date NOT NULL COMMENT '结算日期',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_school_date` (`school_id`, `settlement_date`),
  KEY `idx_date` (`settlement_date`),
  KEY `idx_region_cp` (`region`, `cp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='院校日95结算数据表';
