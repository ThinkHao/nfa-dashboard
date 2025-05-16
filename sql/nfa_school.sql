CREATE TABLE `nfa_school` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `school_id` varchar(10) NOT NULL COMMENT '学校ID',
  `school_name` varchar(128) NOT NULL COMMENT '学校名称',
  `region` varchar(20) NOT NULL COMMENT '地区',
  `cp` varchar(20) NOT NULL COMMENT '运营商',
  `hash_uuids` text NOT NULL COMMENT '聚合前的所有hash_uuid列表',
  `primary_hash_uuid` varchar(128) NOT NULL COMMENT '用于聚合流量的主hash_uuid',
  `hash_count` int(11) NOT NULL DEFAULT '0' COMMENT 'hash_uuid数量',
  `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `data_hash` char(32) NOT NULL COMMENT '数据hash值，用于快速比较数据是否变化',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_school_cp` (`school_id`,`region`,`cp`),
  KEY `idx_school_name` (`school_name`),
  KEY `idx_region` (`region`),
  KEY `idx_cp` (`cp`),
  KEY `idx_primary_hash_uuid` (`primary_hash_uuid`),
  KEY `idx_data_hash` (`data_hash`)
) ENGINE=InnoDB AUTO_INCREMENT=1040 DEFAULT CHARSET=utf8mb4 COMMENT='nfa院校关系表';