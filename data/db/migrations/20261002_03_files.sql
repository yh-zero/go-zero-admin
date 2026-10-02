-- Resource metadata is retained after OSS deletion for audit/history.
SET NAMES utf8mb4 COLLATE utf8mb4_general_ci;
CREATE TABLE IF NOT EXISTS sys_file_resources (
  id BIGINT NOT NULL AUTO_INCREMENT,
  object_key VARCHAR(256) COLLATE utf8mb4_bin NOT NULL,
  name VARCHAR(255) NOT NULL,
  mime VARCHAR(128) NOT NULL,
  size BIGINT NOT NULL,
  owner_id BIGINT NOT NULL,
  department_id BIGINT NOT NULL DEFAULT 0,
  visibility VARCHAR(16) NOT NULL DEFAULT 'public',
  status VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id), UNIQUE KEY uk_file_object_key (object_key),
  KEY idx_file_owner_status (owner_id,status),
  KEY idx_file_department_status (department_id,status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
CREATE TABLE IF NOT EXISTS sys_file_references (
  id BIGINT NOT NULL AUTO_INCREMENT,
  file_id BIGINT NOT NULL,
  object_type VARCHAR(64) COLLATE utf8mb4_bin NOT NULL,
  object_id VARCHAR(128) COLLATE utf8mb4_bin NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id), UNIQUE KEY uk_file_reference (file_id,object_type,object_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
START TRANSACTION;
INSERT INTO sys_apis (created_at,updated_at,path,method,api_group,description)
SELECT NOW(3),NOW(3),r.path,r.method,'files',r.description FROM (
  SELECT '/v1/sys/files/list' path,'GET' method,'按数据范围查询文件资源' description
  UNION ALL SELECT '/v1/sys/files/url','GET','获取公开或短时签名文件地址'
  UNION ALL SELECT '/v1/sys/files/resource','DELETE','删除未被引用的资源，失败可重试'
  UNION ALL SELECT '/v1/sys/files/reference','POST','登记文件业务引用，仅内置管理员'
  UNION ALL SELECT '/v1/sys/files/reference','DELETE','移除文件业务引用，仅内置管理员'
) r WHERE NOT EXISTS (SELECT 1 FROM sys_apis a WHERE a.deleted_at IS NULL
  AND a.path COLLATE utf8mb4_general_ci=r.path COLLATE utf8mb4_general_ci AND a.method COLLATE utf8mb4_general_ci=r.method COLLATE utf8mb4_general_ci);
INSERT INTO casbin_rule (ptype,v0,v1,v2)
SELECT 'p','1',a.path,a.method FROM sys_apis a WHERE a.deleted_at IS NULL AND a.path LIKE '/v1/sys/files/%'
  AND EXISTS (SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
  AND NOT EXISTS (SELECT 1 FROM casbin_rule c WHERE c.ptype='p' AND c.v0='1'
    AND c.v1 COLLATE utf8mb4_general_ci=a.path COLLATE utf8mb4_general_ci AND c.v2 COLLATE utf8mb4_general_ci=a.method COLLATE utf8mb4_general_ci);
COMMIT;
