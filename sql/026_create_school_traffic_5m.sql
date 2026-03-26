-- 026_create_school_traffic_5m.sql
-- 目的：新增 5 分钟聚合事实表，保障任意时间范围均按 5 分钟粒度查询。

CREATE TABLE IF NOT EXISTS `nfa_school_traffic_5m` (
  `bucket_5m` DATETIME NOT NULL COMMENT '5分钟时间桶（本地时区）',
  `region` VARCHAR(20) NOT NULL COMMENT '地区',
  `cp` VARCHAR(20) NOT NULL COMMENT '运营商',
  `school_id` VARCHAR(10) NOT NULL COMMENT '学校ID',
  `school_name` VARCHAR(128) NOT NULL COMMENT '学校名称',
  `total_recv` BIGINT NOT NULL DEFAULT 0 COMMENT '服务流量(bytes)',
  `total_send` BIGINT NOT NULL DEFAULT 0 COMMENT '回源流量(bytes)',
  `record_count` INT NOT NULL DEFAULT 0 COMMENT '聚合行数',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`bucket_5m`, `region`, `cp`, `school_id`),
  KEY `idx_region_cp_bucket` (`region`, `cp`, `bucket_5m`),
  KEY `idx_region_school_bucket_cp` (`region`, `school_name`, `bucket_5m`, `cp`),
  KEY `idx_school_bucket_cp` (`school_name`, `bucket_5m`, `cp`),
  KEY `idx_bucket_only` (`bucket_5m`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='学校流量5分钟聚合表';

-- 可选：如果数据量较大，可后续按月对 bucket_5m 做分区。

-- 增量聚合 SQL 模板（由任务调度器传入窗口）
-- 参数：@from_ts, @to_ts，建议窗口为 [last_cursor, now()-interval 5 minute)
-- 说明：重复执行同一窗口也可正确覆盖（幂等）。

-- INSERT INTO nfa_school_traffic_5m
-- (
--   bucket_5m, region, cp, school_id, school_name, total_recv, total_send, record_count
-- )
-- SELECT
--   FROM_UNIXTIME(UNIX_TIMESTAMP(create_time) - MOD(UNIX_TIMESTAMP(create_time), 300)) AS bucket_5m,
--   region,
--   cp,
--   school_id,
--   school_name,
--   SUM(total_recv) AS total_recv,
--   SUM(total_send) AS total_send,
--   COUNT(*) AS record_count
-- FROM nfa_school_traffic
-- WHERE create_time >= @from_ts AND create_time < @to_ts
-- GROUP BY bucket_5m, region, cp, school_id, school_name
-- ON DUPLICATE KEY UPDATE
--   total_recv = VALUES(total_recv),
--   total_send = VALUES(total_send),
--   record_count = VALUES(record_count),
--   school_name = VALUES(school_name),
--   updated_at = CURRENT_TIMESTAMP;
