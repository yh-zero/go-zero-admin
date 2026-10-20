-- Automatic user avatar references; run with API/RPC stopped.
SET @avatar_column_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS
 WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='app_users' AND COLUMN_NAME='avatar_url');
SET @avatar_column_ddl = IF(@avatar_column_exists=0,
 'ALTER TABLE app_users ADD COLUMN avatar_url VARCHAR(1024) NOT NULL DEFAULT ''''', 'SELECT 1');
PREPARE avatar_column_statement FROM @avatar_column_ddl;
EXECUTE avatar_column_statement;
DEALLOCATE PREPARE avatar_column_statement;

-- Custom bucket installations must set this session variable before this script.
SET @avatar_public_bucket=COALESCE(NULLIF(@avatar_public_bucket,''),'gozero-public');

START TRANSACTION;
SELECT authority_id FROM sys_authorities WHERE authority_id=1 AND deleted_at IS NULL FOR UPDATE;
SELECT revision FROM sys_permission_versions WHERE id=1 FOR UPDATE;

-- Match the object path, including host aliases and optional cache query strings.
-- External/default avatars and non-public/non-active objects receive no reference.
INSERT INTO sys_file_references(file_id,object_type,object_id,created_at)
SELECT f.id,'sys_user_avatar',CAST(u.id AS CHAR),NOW(3)
FROM sys_users u JOIN sys_file_resources f
 ON RIGHT(SUBSTRING_INDEX(SUBSTRING_INDEX(REPLACE(TRIM(u.header_img),CHAR(92),'/'),'?',1),'#',1),CHAR_LENGTH(CONCAT('/',@avatar_public_bucket,'/',f.object_key)))=CONCAT('/',@avatar_public_bucket,'/',f.object_key)
WHERE u.deleted_at IS NULL AND f.visibility='public' AND f.status='active'
 AND NOT EXISTS(SELECT 1 FROM sys_file_references r WHERE r.file_id=f.id AND r.object_type='sys_user_avatar' AND r.object_id=CAST(u.id AS CHAR));

INSERT INTO sys_file_references(file_id,object_type,object_id,created_at)
SELECT f.id,'app_user_avatar',CAST(u.id AS CHAR),NOW(3)
FROM app_users u JOIN sys_file_resources f
 ON RIGHT(SUBSTRING_INDEX(SUBSTRING_INDEX(REPLACE(TRIM(u.avatar_url),CHAR(92),'/'),'?',1),'#',1),CHAR_LENGTH(CONCAT('/',@avatar_public_bucket,'/',f.object_key)))=CONCAT('/',@avatar_public_bucket,'/',f.object_key)
WHERE f.visibility='public' AND f.status='active'
 AND NOT EXISTS(SELECT 1 FROM sys_file_references r WHERE r.file_id=f.id AND r.object_type='app_user_avatar' AND r.object_id=CAST(u.id AS CHAR));

INSERT INTO sys_base_menu_btns(created_at,updated_at,name,`desc`,sys_base_menu_id,permission_key)
SELECT NOW(3),NOW(3),'avatar','设置 App 用户头像',m.id,'app-members-users:avatar'
FROM sys_base_menus m WHERE m.name='app-members-users' AND m.deleted_at IS NULL
 AND NOT EXISTS(SELECT 1 FROM sys_base_menu_btns b WHERE b.permission_key='app-members-users:avatar');

INSERT INTO sys_apis(created_at,updated_at,path,description,api_group,method)
SELECT NOW(3),NOW(3),'/v1/sys/app-members/avatar','设置 App 用户头像','appmembers','PUT'
WHERE NOT EXISTS(SELECT 1 FROM sys_apis WHERE path='/v1/sys/app-members/avatar' AND method='PUT' AND deleted_at IS NULL);

UPDATE sys_permission_versions SET revision=revision+1 WHERE id=1;
COMMIT;
