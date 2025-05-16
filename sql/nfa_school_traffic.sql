CREATE TABLE `nfa_school_traffic` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '流量日志时间',
  `school_id` varchar(10) NOT NULL COMMENT '学校ID',
  `school_name` varchar(128) NOT NULL COMMENT '学校名称',
  `region` varchar(20) NOT NULL COMMENT '地区',
  `cp` varchar(20) NOT NULL COMMENT '运营商',
  `hash_uuid` varchar(128) NOT NULL COMMENT '原始数据hash_uuid',
  `total_recv` bigint(20) NOT NULL DEFAULT '0' COMMENT '接收流量(bytes)',
  `total_send` bigint(20) NOT NULL DEFAULT '0' COMMENT '发送流量(bytes)',
  PRIMARY KEY (`id`),
  KEY `idx_time_region_school_cp` (`create_time`,`region`,`school_name`,`cp`),
  KEY `idx_create_time` (`create_time`),
  KEY `idx_region_time` (`region`,`create_time`),
  KEY `idx_school_time` (`school_name`,`create_time`),
  KEY `idx_cp_time` (`cp`,`create_time`),
  KEY `idx_hash_uuid` (`hash_uuid`)
) ENGINE=InnoDB AUTO_INCREMENT=338044172 DEFAULT CHARSET=utf8mb4 COMMENT='nfa院校流量数据表';