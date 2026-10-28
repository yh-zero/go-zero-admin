-- Apply once to an existing database after a backup. No startup AutoMigrate.
CREATE TABLE IF NOT EXISTS sys_permission_versions (
 id bigint NOT NULL PRIMARY KEY,
 revision bigint unsigned NOT NULL
) ENGINE=InnoDB;
INSERT IGNORE INTO sys_permission_versions(id,revision) VALUES(1,1);

CREATE TABLE IF NOT EXISTS sys_permission_changes (
 id bigint NOT NULL AUTO_INCREMENT PRIMARY KEY,
 authority_id bigint NOT NULL,
 kind varchar(16) NOT NULL,
 before_revision bigint unsigned NOT NULL,
 after_revision bigint unsigned NOT NULL,
 `before` longtext NOT NULL,
 `after` longtext NOT NULL,
 actor_id bigint NOT NULL DEFAULT 0,
 actor_name varchar(64) NOT NULL DEFAULT '',
 trace_id varchar(64) NOT NULL DEFAULT '',
 created_at datetime(3) NOT NULL,
 KEY idx_permission_history_role (authority_id,id),
 KEY idx_permission_history_revision (after_revision)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Both historical names are varchar(191); retain their full combined key.
-- REDUNDANT row format limits index keys to 767 bytes, so expand it first.
ALTER TABLE sys_base_menu_btns ROW_FORMAT=DYNAMIC;
ALTER TABLE sys_base_menu_btns ADD COLUMN permission_key varchar(383) NULL;
UPDATE sys_base_menu_btns b JOIN sys_base_menus m ON m.id=b.sys_base_menu_id
SET b.permission_key=CONCAT(m.name,':',b.name) WHERE b.permission_key IS NULL;
-- Orphan definitions receive a stable fallback. A duplicate legacy code causes
-- the unique index below to fail visibly; resolve those definitions, do not
-- silently merge their grants.
UPDATE sys_base_menu_btns SET permission_key=CONCAT('legacy.button.',id) WHERE permission_key IS NULL;
ALTER TABLE sys_base_menu_btns MODIFY permission_key varchar(383) NOT NULL;
CREATE UNIQUE INDEX idx_sys_base_menu_btns_permission_key ON sys_base_menu_btns(permission_key);
-- New permission endpoints are catalogued by the existing API sync workflow;
-- no policies are inserted and no API permission is granted by this migration.
CREATE INDEX idx_device_revoked ON sys_device_sessions(revoked_at,id);
