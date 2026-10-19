-- App accounts and membership; requires all earlier migrations.
-- Adds resources only. Existing roles gain no permissions until explicitly granted.
SET NAMES utf8mb4 COLLATE utf8mb4_general_ci;

CREATE TABLE IF NOT EXISTS app_users (
 id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
 nickname VARCHAR(64) NOT NULL,
 status VARCHAR(16) NOT NULL DEFAULT 'pending',
 session_version BIGINT NOT NULL DEFAULT 1,
 review_reason VARCHAR(500) NOT NULL DEFAULT '',
 reviewed_by BIGINT NOT NULL DEFAULT 0,
 reviewed_at DATETIME(3) NULL,
 created_at DATETIME(3) NOT NULL,
 updated_at DATETIME(3) NOT NULL,
 KEY idx_app_user_status(status,id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS app_user_identities (
 id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
 user_id BIGINT NOT NULL,
 kind VARCHAR(16) NOT NULL,
 identifier VARCHAR(254) COLLATE utf8mb4_bin NOT NULL,
 password_hash VARCHAR(255) NOT NULL,
 verified_at DATETIME(3) NULL,
 created_at DATETIME(3) NOT NULL,
 UNIQUE KEY uk_app_identity_user(user_id),
 UNIQUE KEY uk_app_identity_identifier(identifier)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS app_sessions (
 id VARCHAR(36) NOT NULL PRIMARY KEY,
 user_id BIGINT NOT NULL,
 refresh_hash VARCHAR(64) NOT NULL,
 previous_refresh_hash VARCHAR(64) NOT NULL DEFAULT '',
 refresh_generation BIGINT NOT NULL DEFAULT 1,
 session_version BIGINT NOT NULL,
 ip VARCHAR(64) NOT NULL,
 user_agent VARCHAR(512) NOT NULL,
 expires_at DATETIME(3) NOT NULL,
 revoked_at DATETIME(3) NULL,
 created_at DATETIME(3) NOT NULL,
 updated_at DATETIME(3) NOT NULL,
 KEY idx_app_session_user(user_id),
 KEY idx_app_session_expiry(expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS app_membership_plans (
 id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
 code VARCHAR(32) NOT NULL,
 name VARCHAR(64) NOT NULL,
 version BIGINT NOT NULL,
 entitlements JSON NOT NULL,
 enabled BOOLEAN NOT NULL,
 created_at DATETIME(3) NOT NULL,
 updated_at DATETIME(3) NOT NULL,
 UNIQUE KEY uk_app_plan_version(code,version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS app_membership_grants (
 id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
 user_id BIGINT NOT NULL,
 plan_id BIGINT NOT NULL,
 plan_code VARCHAR(32) NOT NULL,
 plan_version BIGINT NOT NULL,
 entitlements JSON NOT NULL,
 request_id VARCHAR(64) COLLATE utf8mb4_bin NOT NULL,
 days BIGINT NOT NULL,
 starts_at DATETIME(3) NOT NULL,
 ends_at DATETIME(3) NOT NULL,
 revoked_at DATETIME(3) NULL,
 granted_by BIGINT NOT NULL,
 revoked_by BIGINT NOT NULL DEFAULT 0,
 reason VARCHAR(500) NOT NULL,
 revoke_reason VARCHAR(500) NOT NULL DEFAULT '',
 created_at DATETIME(3) NOT NULL,
 UNIQUE KEY uk_app_grant_request(request_id),
 KEY idx_app_grant_user_end(user_id,ends_at),
 KEY idx_app_grant_revoked(revoked_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

START TRANSACTION;
INSERT INTO app_membership_plans(code,name,version,entitlements,enabled,created_at,updated_at)
SELECT 'vip','VIP 会员',1,JSON_ARRAY('profile.export'),1,UTC_TIMESTAMP(3),UTC_TIMESTAMP(3)
WHERE NOT EXISTS(SELECT 1 FROM app_membership_plans WHERE code='vip' AND version=1);

INSERT INTO sys_base_menus(created_at,updated_at,menu_level,parent_id,path,name,hidden,component,sort,active_name,keep_alive,default_menu,title,icon,close_tab)
SELECT NOW(3),NOW(3),0,0,seed.path,seed.name,0,seed.component,seed.sort,'',0,0,seed.title,'',0
FROM (
 SELECT 'app-members/users' path,'app-members-users' name,'views/business/app-members/users.vue' component,'App 用户' title,70 sort
 UNION ALL SELECT 'app-members/memberships','app-memberships','views/business/app-members/memberships.vue','App 会员',71
) seed
WHERE NOT EXISTS(SELECT 1 FROM sys_base_menus m WHERE m.deleted_at IS NULL AND (m.name=seed.name OR TRIM(BOTH '/' FROM m.path)=seed.path));

INSERT INTO sys_base_menu_btns(created_at,updated_at,name,`desc`,sys_base_menu_id,permission_key)
SELECT NOW(3),NOW(3),seed.name,seed.description,m.id,CONCAT(seed.menu,':',seed.name)
FROM (
 SELECT 'app-members-users' menu,'review' name,'审核 App 账号' description
 UNION ALL SELECT 'app-members-users','status','禁用或恢复 App 账号'
 UNION ALL SELECT 'app-members-users','revoke-session','撤销 App 设备'
 UNION ALL SELECT 'app-memberships','grant','开通或续期会员'
 UNION ALL SELECT 'app-memberships','revoke','撤销会员授予'
) seed
JOIN sys_base_menus m ON m.name=seed.menu AND m.deleted_at IS NULL
 AND m.component IN('views/business/app-members/users.vue','views/business/app-members/memberships.vue')
WHERE NOT EXISTS(SELECT 1 FROM sys_base_menu_btns b WHERE b.permission_key=CONCAT(seed.menu,':',seed.name));

INSERT INTO sys_apis(created_at,updated_at,path,description,api_group,method)
SELECT NOW(3),NOW(3),seed.path,seed.description,'appmembers',seed.method
FROM (
 SELECT '/v1/sys/app-members/users' path,'GET' method,'查询 App 用户' description
 UNION ALL SELECT '/v1/sys/app-members/status','PUT','审核或调整 App 账号状态'
 UNION ALL SELECT '/v1/sys/app-members/sessions','GET','查询 App 设备会话'
 UNION ALL SELECT '/v1/sys/app-members/session','DELETE','撤销 App 设备会话'
 UNION ALL SELECT '/v1/sys/app-members/plans','GET','查询会员方案'
 UNION ALL SELECT '/v1/sys/app-members/grants','GET','查询会员授予'
 UNION ALL SELECT '/v1/sys/app-members/grants','POST','开通或续期会员'
 UNION ALL SELECT '/v1/sys/app-members/grant','DELETE','撤销会员授予'
) seed
WHERE NOT EXISTS(SELECT 1 FROM sys_apis a WHERE a.path=seed.path AND a.method=seed.method AND a.deleted_at IS NULL);

UPDATE sys_permission_versions SET revision=revision+1 WHERE id=1;
COMMIT;

