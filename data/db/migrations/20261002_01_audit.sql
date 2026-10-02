-- Append-only operation/login audit trail. Run with the existing migration tool.
-- No credential, request body or response body columns are stored.
SET NAMES utf8mb4 COLLATE utf8mb4_general_ci;
CREATE TABLE IF NOT EXISTS sys_audit_logs (
  id BIGINT NOT NULL AUTO_INCREMENT,
  actor_id BIGINT NOT NULL DEFAULT 0,
  actor_name VARCHAR(64) NOT NULL DEFAULT '',
  authority_id BIGINT NOT NULL DEFAULT 0,
  event_type VARCHAR(16) NOT NULL,
  module VARCHAR(64) NOT NULL,
  action VARCHAR(64) NOT NULL,
  object VARCHAR(256) NOT NULL DEFAULT '',
  path VARCHAR(256) NOT NULL DEFAULT '',
  method VARCHAR(16) NOT NULL DEFAULT '',
  result VARCHAR(16) NOT NULL,
  status_code BIGINT NOT NULL DEFAULT 0,
  ip VARCHAR(64) NOT NULL DEFAULT '',
  trace_id VARCHAR(64) NOT NULL DEFAULT '',
  duration_ms BIGINT NOT NULL DEFAULT 0,
  params TEXT NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_audit_time (created_at),
  KEY idx_audit_actor_time (actor_id, created_at),
  KEY idx_audit_type_time (event_type, created_at),
  KEY idx_audit_module_time (module, created_at),
  KEY idx_audit_result_time (result, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

START TRANSACTION;
INSERT INTO sys_apis (created_at, updated_at, path, method, api_group, description)
SELECT NOW(3), NOW(3), '/v1/sys/audit/getAuditLogList', 'GET', 'audit', '分页查询操作审计与登录日志'
WHERE NOT EXISTS (SELECT 1 FROM sys_apis WHERE path='/v1/sys/audit/getAuditLogList' AND method='GET' AND deleted_at IS NULL);
INSERT INTO casbin_rule (ptype, v0, v1, v2)
SELECT 'p', '1', '/v1/sys/audit/getAuditLogList', 'GET'
WHERE EXISTS (SELECT 1 FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL)
  AND NOT EXISTS (SELECT 1 FROM casbin_rule c WHERE c.ptype='p' AND c.v0='1'
    AND c.v1 COLLATE utf8mb4_general_ci='/v1/sys/audit/getAuditLogList' COLLATE utf8mb4_general_ci
    AND c.v2 COLLATE utf8mb4_general_ci='GET' COLLATE utf8mb4_general_ci);
COMMIT;
