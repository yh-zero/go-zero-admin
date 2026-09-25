-- MySQL 8: unique active usernames, while keeping soft-deleted usernames reusable.
-- Run after selecting the application's database. No user records are changed or deleted.
-- If active duplicates exist, this migration stops. Review those accounts manually first.
-- The generated value uses the same utf8mb4_general_ci collation as username in this project.
-- Re-running is safe. DDL commits implicitly; if adding the index fails, resolve the cause
-- and re-run to finish the migration. The existing nonunique username index is preserved.
-- Rollback (removes only this migration's constraint and generated column):
-- ALTER TABLE sys_users DROP INDEX uk_sys_users_active_username, DROP COLUMN active_username;

DROP PROCEDURE IF EXISTS migrate_user_active_username_20260925;
DELIMITER $$
CREATE PROCEDURE migrate_user_active_username_20260925()
BEGIN
    IF EXISTS (
        SELECT username FROM sys_users
        WHERE deleted_at IS NULL AND username IS NOT NULL
        GROUP BY username HAVING COUNT(*) > 1
    ) THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'Active usernames are duplicated; review existing accounts before this migration';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = DATABASE() AND table_name = 'sys_users'
          AND column_name = 'username' AND data_type = 'varchar'
          AND character_maximum_length = 191 AND collation_name = 'utf8mb4_general_ci'
    ) THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'Unexpected username schema; review generated-column type and collation first';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = DATABASE() AND table_name = 'sys_users' AND column_name = 'active_username'
    ) THEN
        ALTER TABLE sys_users ADD COLUMN active_username VARCHAR(191)
            CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci
            GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL THEN username ELSE NULL END) STORED;
    ELSEIF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = DATABASE() AND table_name = 'sys_users'
          AND column_name = 'active_username' AND extra LIKE '%STORED GENERATED%'
          AND data_type = 'varchar' AND character_maximum_length = 191
          AND collation_name = 'utf8mb4_general_ci'
          AND LOWER(REPLACE(REPLACE(REPLACE(generation_expression, '`', ''), ' ', ''), '(', ''))
              LIKE '%casewhendeleted_atisnull%thenusernameelsenullend%'
    ) THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'An incompatible active_username column already exists; no existing data was changed';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.statistics
        WHERE table_schema = DATABASE() AND table_name = 'sys_users' AND index_name = 'uk_sys_users_active_username'
    ) THEN
        ALTER TABLE sys_users ADD UNIQUE INDEX uk_sys_users_active_username (active_username);
    ELSEIF (
        SELECT COUNT(*) FROM information_schema.statistics
        WHERE table_schema = DATABASE() AND table_name = 'sys_users' AND index_name = 'uk_sys_users_active_username'
    ) <> 1 OR NOT EXISTS (
        SELECT 1 FROM information_schema.statistics
        WHERE table_schema = DATABASE() AND table_name = 'sys_users'
          AND index_name = 'uk_sys_users_active_username' AND column_name = 'active_username'
          AND non_unique = 0 AND sub_part IS NULL
    ) THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'An incompatible active username index already exists; inspect it before continuing';
    END IF;
END$$
DELIMITER ;
CALL migrate_user_active_username_20260925();
DROP PROCEDURE migrate_user_active_username_20260925;
