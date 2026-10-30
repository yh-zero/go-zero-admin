-- +goose Up
-- 平台控制库：tenant_registry（仅此一张表；业务表在各租户独立库中）

SET NAMES utf8mb4;

CREATE TABLE tenant_registry (
  id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  slug             VARCHAR(32)  NOT NULL COMMENT '租户标识（英文小写，子域名前缀）',
  domain           VARCHAR(128) NOT NULL COMMENT '完整域名，如 t1.localhost',
  compose_project  VARCHAR(64)  NOT NULL COMMENT 'compose 项目名，如 gozero-t-t1',
  status           VARCHAR(16)  NOT NULL DEFAULT 'active' COMMENT '状态：active/stopped/error',
  mysql_root_password VARCHAR(128) NOT NULL COMMENT '该租户 MySQL root 密码（随机生成）',
  jwt_access_secret   VARCHAR(128) NOT NULL COMMENT '该租户后台 JWT 密钥（随机生成）',
  app_jwt_access_secret VARCHAR(128) NOT NULL COMMENT '该租户 App JWT 密钥（随机生成）',
  error_msg        VARCHAR(512) NOT NULL DEFAULT '' COMMENT '最近一次错误信息',
  created_by       BIGINT UNSIGNED NULL DEFAULT NULL COMMENT '创建操作员ID',
  created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at       DATETIME NULL DEFAULT NULL COMMENT '软删除（注销租户）',
  PRIMARY KEY (id),
  UNIQUE KEY uk_slug (slug),
  UNIQUE KEY uk_domain (domain),
  KEY idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='租户注册表';

-- +goose Down
DROP TABLE IF EXISTS tenant_registry;
