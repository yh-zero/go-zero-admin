-- MySQL 8: enforce the same active-record uniqueness checked by the application.
-- No business rows are changed or deleted. Resolve any duplicate rows manually;
-- the preflight checks ALL constraints before this script performs any DDL.
-- Existing column collations remain authoritative. NULL generated markers let
-- multiple soft-deleted versions coexist while active records must be unique.
-- DDL commits implicitly. Re-running safely finishes a partially applied DDL.
-- Rollback: drop the uk_* indexes below, then active_unique from these 4 tables.

DROP PROCEDURE IF EXISTS migrate_business_unique_20260926;
DROP PROCEDURE IF EXISTS ensure_active_marker_20260926;
DROP PROCEDURE IF EXISTS ensure_active_index_20260926;
DELIMITER $$

CREATE PROCEDURE ensure_active_marker_20260926(IN target_table VARCHAR(64))
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = DATABASE() AND table_name = target_table AND column_name = 'active_unique'
    ) THEN
        SET @unique_ddl = CONCAT('ALTER TABLE `', target_table, '` ADD COLUMN active_unique TINYINT GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL THEN 1 ELSE NULL END) STORED');
        PREPARE unique_statement FROM @unique_ddl;
        EXECUTE unique_statement;
        DEALLOCATE PREPARE unique_statement;
    ELSEIF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = DATABASE() AND table_name = target_table AND column_name = 'active_unique'
          AND data_type = 'tinyint' AND extra LIKE '%STORED GENERATED%'
          AND LOWER(REPLACE(REPLACE(REPLACE(REPLACE(generation_expression, '`', ''), ' ', ''), '(', ''), ')', ''))
              = 'casewhendeleted_atisnullthen1elsenullend'
    ) THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'Incompatible active_unique column: inspect its generated expression before migration';
    END IF;
END$$

CREATE PROCEDURE ensure_active_index_20260926(IN target_table VARCHAR(64), IN index_key VARCHAR(64), IN index_columns VARCHAR(191))
BEGIN
    DECLARE actual_columns VARCHAR(191);
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.statistics
        WHERE table_schema = DATABASE() AND table_name = target_table AND index_name = index_key
    ) THEN
        SET @unique_ddl = CONCAT('ALTER TABLE `', target_table, '` ADD UNIQUE INDEX `', index_key, '` (', index_columns, ')');
        PREPARE unique_statement FROM @unique_ddl;
        EXECUTE unique_statement;
        DEALLOCATE PREPARE unique_statement;
    ELSE
        SELECT GROUP_CONCAT(column_name ORDER BY seq_in_index SEPARATOR ',') INTO actual_columns
        FROM information_schema.statistics
        WHERE table_schema = DATABASE() AND table_name = target_table AND index_name = index_key;
        IF actual_columns <> index_columns OR EXISTS (
            SELECT 1 FROM information_schema.statistics
            WHERE table_schema = DATABASE() AND table_name = target_table AND index_name = index_key
              AND (non_unique <> 0 OR sub_part IS NOT NULL)
        ) THEN
            SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'Incompatible active-record unique index: inspect it before migration';
        END IF;
    END IF;
END$$

CREATE PROCEDURE migrate_business_unique_20260926()
BEGIN
    IF EXISTS (SELECT name FROM sys_base_menus WHERE deleted_at IS NULL AND name IS NOT NULL GROUP BY name HAVING COUNT(*) > 1) THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'Duplicate active menu names: resolve sys_base_menus.name before migration';
    END IF;
    IF EXISTS (SELECT path FROM sys_base_menus WHERE deleted_at IS NULL AND path IS NOT NULL GROUP BY path HAVING COUNT(*) > 1) THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'Duplicate active menu paths: resolve sys_base_menus.path before migration';
    END IF;
    IF EXISTS (SELECT path, method FROM sys_apis WHERE deleted_at IS NULL AND path IS NOT NULL AND method IS NOT NULL GROUP BY path, method HAVING COUNT(*) > 1) THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'Duplicate active API routes: resolve sys_apis.path + method before migration';
    END IF;
    IF EXISTS (SELECT type FROM sys_dictionaries WHERE deleted_at IS NULL AND type IS NOT NULL GROUP BY type HAVING COUNT(*) > 1) THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'Duplicate active dictionary types: resolve sys_dictionaries.type before migration';
    END IF;
    IF EXISTS (SELECT sys_dictionary_id, value FROM sys_dictionary_info WHERE deleted_at IS NULL AND sys_dictionary_id IS NOT NULL AND value IS NOT NULL GROUP BY sys_dictionary_id, value HAVING COUNT(*) > 1) THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'Duplicate active dictionary values: resolve dictionary ID + value before migration';
    END IF;

    CALL ensure_active_marker_20260926('sys_base_menus');
    CALL ensure_active_index_20260926('sys_base_menus', 'uk_sys_menus_active_name', 'name,active_unique');
    CALL ensure_active_index_20260926('sys_base_menus', 'uk_sys_menus_active_path', 'path,active_unique');
    CALL ensure_active_marker_20260926('sys_apis');
    CALL ensure_active_index_20260926('sys_apis', 'uk_sys_apis_active_route', 'path,method,active_unique');
    CALL ensure_active_marker_20260926('sys_dictionaries');
    CALL ensure_active_index_20260926('sys_dictionaries', 'uk_sys_dictionaries_active_type', 'type,active_unique');
    CALL ensure_active_marker_20260926('sys_dictionary_info');
    CALL ensure_active_index_20260926('sys_dictionary_info', 'uk_sys_dictionary_info_active_value', 'sys_dictionary_id,value,active_unique');
END$$
DELIMITER ;
CALL migrate_business_unique_20260926();
DROP PROCEDURE migrate_business_unique_20260926;
DROP PROCEDURE ensure_active_marker_20260926;
DROP PROCEDURE ensure_active_index_20260926;
