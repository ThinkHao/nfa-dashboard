-- 016_user_schools.sql
-- 目的：建立用户与院校的关联关系，用于 v2 接口按用户过滤数据
-- 注意：仅建立关联，不做强制外键；由应用层保证 user_id 与 school_id 的合法性

CREATE TABLE IF NOT EXISTS user_schools (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id BIGINT UNSIGNED NOT NULL,
  school_id VARCHAR(64) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_user_school (user_id, school_id),
  KEY idx_user_id (user_id),
  KEY idx_school_id (school_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
