DELETE FROM `sys_api` WHERE `id` IN (139,140,141,142);

SET @has_idx := (SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name='sys_audit_log' AND index_name='idx_ip');
SET @sql := IF(@has_idx>0, 'ALTER TABLE `sys_audit_log` DROP INDEX `idx_ip`', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @has_idx := (SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name='sys_audit_log' AND index_name='idx_method');
SET @sql := IF(@has_idx>0, 'ALTER TABLE `sys_audit_log` DROP INDEX `idx_method`', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @has_idx := (SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name='sys_audit_log' AND index_name='idx_username');
SET @sql := IF(@has_idx>0, 'ALTER TABLE `sys_audit_log` DROP INDEX `idx_username`', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @has_idx := (SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name='sys_file' AND index_name='idx_created_at');
SET @sql := IF(@has_idx>0, 'ALTER TABLE `sys_file` DROP INDEX `idx_created_at`', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @has_idx := (SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name='sys_file' AND index_name='idx_mime');
SET @sql := IF(@has_idx>0, 'ALTER TABLE `sys_file` DROP INDEX `idx_mime`', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @has_idx := (SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name='sys_file' AND index_name='idx_original_name');
SET @sql := IF(@has_idx>0, 'ALTER TABLE `sys_file` DROP INDEX `idx_original_name`', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @has_sha256 := (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='sys_file' AND column_name='sha256');
SET @sql := IF(@has_sha256>0, 'ALTER TABLE `sys_file` DROP COLUMN `sha256`', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
