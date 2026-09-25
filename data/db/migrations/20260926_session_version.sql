-- Existing JWTs do not have a version and must sign in again after upgrading.
SET @session_column = (SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'sys_users' AND COLUMN_NAME = 'session_version');
SET @session_ddl = IF(@session_column = 0,
  'ALTER TABLE sys_users ADD COLUMN session_version BIGINT NOT NULL DEFAULT 1 COMMENT ''Session version''',
  'SELECT 1');
PREPARE session_statement FROM @session_ddl;
EXECUTE session_statement;
DEALLOCATE PREPARE session_statement;
