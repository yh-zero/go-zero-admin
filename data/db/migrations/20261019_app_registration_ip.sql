-- Registration network metadata; run after 20261019_app_members.sql.
-- Existing users retain their data and have unknown (empty) registration metadata.
SET @app_registration_ip_column = (SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'app_users' AND COLUMN_NAME = 'registration_ip');
SET @app_registration_ip_ddl = IF(@app_registration_ip_column = 0,
  'ALTER TABLE app_users ADD COLUMN registration_ip VARCHAR(64) NOT NULL DEFAULT ''''',
  'SELECT 1');
PREPARE app_registration_ip_statement FROM @app_registration_ip_ddl;
EXECUTE app_registration_ip_statement;
DEALLOCATE PREPARE app_registration_ip_statement;

SET @app_registration_region_column = (SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'app_users' AND COLUMN_NAME = 'registration_region');
SET @app_registration_region_ddl = IF(@app_registration_region_column = 0,
  'ALTER TABLE app_users ADD COLUMN registration_region VARCHAR(255) NOT NULL DEFAULT ''''',
  'SELECT 1');
PREPARE app_registration_region_statement FROM @app_registration_region_ddl;
EXECUTE app_registration_region_statement;
DEALLOCATE PREPARE app_registration_region_statement;
