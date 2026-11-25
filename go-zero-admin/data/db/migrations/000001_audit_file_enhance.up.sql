-- 001：审计与文件增强（兼容已有开发库）
SET @has_sha256 := (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='sys_file' AND column_name='sha256');
SET @sql := IF(@has_sha256=0, 'ALTER TABLE `sys_file` ADD COLUMN `sha256` char(64) NOT NULL DEFAULT \'\' COMMENT \'文件内容 SHA-256，历史文件为空\' AFTER `size`', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @has_idx := (SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name='sys_file' AND index_name='idx_original_name');
SET @sql := IF(@has_idx=0, 'ALTER TABLE `sys_file` ADD INDEX `idx_original_name` (`original_name`)', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @has_idx := (SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name='sys_file' AND index_name='idx_mime');
SET @sql := IF(@has_idx=0, 'ALTER TABLE `sys_file` ADD INDEX `idx_mime` (`mime`)', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @has_idx := (SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name='sys_file' AND index_name='idx_created_at');
SET @sql := IF(@has_idx=0, 'ALTER TABLE `sys_file` ADD INDEX `idx_created_at` (`created_at`)', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @has_idx := (SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name='sys_audit_log' AND index_name='idx_username');
SET @sql := IF(@has_idx=0, 'ALTER TABLE `sys_audit_log` ADD INDEX `idx_username` (`username`)', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @has_idx := (SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name='sys_audit_log' AND index_name='idx_method');
SET @sql := IF(@has_idx=0, 'ALTER TABLE `sys_audit_log` ADD INDEX `idx_method` (`method`)', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @has_idx := (SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name='sys_audit_log' AND index_name='idx_ip');
SET @sql := IF(@has_idx=0, 'ALTER TABLE `sys_audit_log` ADD INDEX `idx_ip` (`ip`)', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

INSERT INTO `sys_api` (`id`,`group_name`,`method`,`path`,`title`) VALUES
(139,'sys/audit','POST','/v1/sys/audit/export','导出审计日志'),
(140,'sys/file','GET','/v1/sys/file/:id','文件详情与引用'),
(141,'sys/file','GET','/v1/sys/file/stats','文件存储统计'),
(142,'sys/file','POST','/v1/sys/file/upload-multipart','multipart 流式上传')
ON DUPLICATE KEY UPDATE `title`=VALUES(`title`);
